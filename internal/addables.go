package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// addableGenerators has a mapping of ast expression to
// slice of addableGenerator's that could generate addable
// based on provided expression type.
var addableGenerators = map[string][]addableGenerator{}

type addableGenerator func(p *whereBodyParser, s ast.Expr, args map[string]int) Addable
type typedGenerator[T ast.Expr] func(p *whereBodyParser, s T, args map[string]int) Addable

func init() {
	addGenerator(func(p *whereBodyParser, s *ast.BasicLit, args map[string]int) Addable {
		return NewSimple(param, getArg(s))
	})
	addGenerator(func(p *whereBodyParser, s *ast.BinaryExpr, args map[string]int) Addable {
		return p.fromBinaryExpr(s, args)
	})
	addGenerator(func(p *whereBodyParser, s *ast.SelectorExpr, args map[string]int) Addable {
		// Supports more cases for arguments
		gotExprName := p.c.exprName(s)
		if !strings.HasPrefix(gotExprName, p.paramName+".") {
			argPos, ok := args[gotExprName]
			if !ok {
				p.c.panicWithPosf(s, "argument is not provided: "+gotExprName)
			}

			return NewSimple(param, fromArgs(argPos))
		}

		return NewColumn(s.Sel.Name)
	})
	addGenerator(func(p *whereBodyParser, s *ast.ParenExpr, args map[string]int) Addable {
		return Parens{p.exprToAddable(s.X, args)}
	})
	addGenerator(func(p *whereBodyParser, s *ast.Ident, args map[string]int) Addable {
		switch s.Name {
		case "true", "false":
			// True/false values do not have Obj set,
			// and if it is set - the value is re-defined.
			if s.Obj != nil {
				break
			}

			return NewSimple(param, raw(s.Name))
		}

		argPos, ok := args[s.Name]
		if !ok {
			p.c.panicWithPosf(s, "argument is not provided: "+s.Name)
		}

		return NewSimple(param, fromArgs(argPos))
	})
	addGenerator(func(p *whereBodyParser, s *ast.UnaryExpr, args map[string]int) Addable {
		switch s.Op {
		case token.NOT:
			return Not{p.getAddable(s.X, args)}
		default:
			p.c.panicWithPosf(s, "unknown unary operator %s", s.Op)
			return nil
		}
	})
	// Add some time functions
	addTypeFuncGenerator("time.Time", "After", "Before", "Equal")(
		func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			funcSelectorExpr := s.Fun.(*ast.SelectorExpr)

			var op string

			switch funcSelectorExpr.Sel.Name {
			case "After":
				op = tokenToOperation(token.GTR)
			case "Before":
				op = tokenToOperation(token.LSS)
			case "Equal":
				op = tokenToOperation(token.EQL)
			default:
				p.c.panicWithPosf(funcSelectorExpr.Sel, "time function is not supported: %q", funcSelectorExpr.Sel.Name)
			}

			return newBinary(
				p.exprToAddable(funcSelectorExpr.X, args),
				op,
				p.exprToAddable(s.Args[0], args),
			)
		},
	)

	addPackageFuncGenerator("time", "Now")(
		func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			return NewSimple("NOW()")
		},
	)

	addPackageFuncGenerator("strings", "ToUpper", "ToLower")(
		func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			var strF func(Addable) string
			switch s.Fun.(*ast.SelectorExpr).Sel.Name {
			case "ToUpper":
				strF = func(a Addable) string { return "upper(" + a.String() + ")" }
			case "ToLower":
				strF = func(a Addable) string { return "lower(" + a.String() + ")" }
			}

			return Wrapper{
				Addable: p.exprToAddable(s.Args[0], args),
				StringF: strF,
			}

		},
	)
	addPackageFuncGenerator("strings", "Contains")(
		func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			// ? LIKE '%' || ? || '%'
			return newBinary(
				p.exprToAddable(s.Args[0], args),
				"LIKE",
				Wrapper{
					Addable: p.exprToAddable(s.Args[1], args),
					StringF: func(a Addable) string {
						return `'%' || ` + a.String() + ` || '%'`
					},
				},
			)
		},
	)
	addPackageFuncGenerator("strings", "HasPrefix", "HasSuffix")(
		func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			// HasPrefix: ? LIKE ? || '%'
			// HasSuffix: ? LIKE '%' || ?
			var strF func(Addable) string
			switch s.Fun.(*ast.SelectorExpr).Sel.Name {
			case "HasPrefix":
				strF = func(a Addable) string { return a.String() + ` || '%'` }
			case "HasSuffix":
				strF = func(a Addable) string { return `'%' || ` + a.String() }
			}

			return newBinary(
				p.exprToAddable(s.Args[0], args),
				"LIKE",
				Wrapper{
					Addable: p.exprToAddable(s.Args[1], args),
					StringF: strF,
				},
			)

		},
	)
}

func wrapper[T ast.Expr](f func(p *whereBodyParser, s T, args map[string]int) Addable) addableGenerator {
	return func(p *whereBodyParser, s ast.Expr, args map[string]int) Addable {
		return f(p, s.(T), args)
	}
}

func (p *whereBodyParser) getAddable(expr ast.Expr, args map[string]int) Addable {
	strTp := typeKey(expr)

	generators, ok := addableGenerators[strTp]
	if !ok {
		p.c.panicWithPosf(expr, "don't know how to convert type "+strTp)
	}

	for _, generator := range generators {
		if addable := generator(p, expr, args); addable != nil {
			return addable
		}
	}

	p.c.panicWithPosf(expr, "didn't find any suitable generator for type "+strTp)

	return nil
}

func addGenerator[T ast.Expr](f typedGenerator[T]) {
	var t T
	strTp := typeKey(t)

	addableGenerators[strTp] = append(addableGenerators[strTp], wrapper(f))
}

func addPackageFuncGenerator(packageName string, funcNames ...string) func(generator typedGenerator[*ast.CallExpr]) {
	funcNamesMap := map[string]struct{}{}
	for _, funcName := range funcNames {
		funcNamesMap[funcName] = struct{}{}
	}

	return func(generator typedGenerator[*ast.CallExpr]) {
		addGenerator(func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			selector, ok := s.Fun.(*ast.SelectorExpr)
			if !ok {
				return nil
			}

			if ident, ok := selector.X.(*ast.Ident); !ok || ident.Obj != nil || ident.Name != packageName {
				return nil
			}

			if _, ok := funcNamesMap[selector.Sel.Name]; !ok {
				return nil
			}

			return generator(p, s, args)
		})
	}
}

func addTypeFuncGenerator(typeName string, funcNames ...string) func(generator typedGenerator[*ast.CallExpr]) {
	funcNamesMap := map[string]struct{}{}
	for _, funcName := range funcNames {
		funcNamesMap[funcName] = struct{}{}
	}

	packageName, typeName := splitType(typeName)

	return func(generator typedGenerator[*ast.CallExpr]) {
		addGenerator(func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
			funcSelectorExpr, ok := s.Fun.(*ast.SelectorExpr)
			if !ok {
				return nil
			}
			// Check that we call these methods on "time.Time" type.
			if !p.exprIsOfType(funcSelectorExpr.X, packageName, typeName) {
				return nil
			}

			if _, ok := funcNamesMap[funcSelectorExpr.Sel.Name]; !ok {
				return nil
			}

			return generator(p, s, args)
		})
	}
}

func typeKey[T ast.Expr](e T) string {
	return fmt.Sprintf("%T", e)
}

func (p *whereBodyParser) exprIsOfType(t ast.Expr, packageName, typeName string) bool {
	tp := p.c.TypeInfo.TypeOf(t)

	namedTp, ok := tp.(*types.Named)
	if !ok {
		return false
	}
	if namedTp.Obj().Pkg() == nil {
		return false
	}

	pkgEqual := namedTp.Obj().Pkg().Name() == packageName

	typeEql := true
	if typeName != "" {
		typeEql = namedTp.Obj().Name() == typeName
	}

	return pkgEqual && typeEql
}

func splitType(input string) (packageName string, typeName string) {
	lastDot := strings.LastIndex(input, ".")

	return input[:lastDot], input[lastDot+1:]
}
