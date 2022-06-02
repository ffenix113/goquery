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
type typedAddableGenerator[T ast.Expr] func(p *whereBodyParser, s T, args map[string]int) Addable

func init() {
	addAddableGenerator(func(p *whereBodyParser, s *ast.BasicLit, args map[string]int) Addable {
		return NewSimple(param, getArg(s))
	})
	addAddableGenerator(func(p *whereBodyParser, s *ast.BinaryExpr, args map[string]int) Addable {
		return p.fromBinaryExpr(s, args)
	})
	addAddableGenerator(func(p *whereBodyParser, s *ast.SelectorExpr, args map[string]int) Addable {
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
	addAddableGenerator(func(p *whereBodyParser, s *ast.ParenExpr, args map[string]int) Addable {
		return Parens{p.exprToAddable(s.X, args)}
	})
	addAddableGenerator(func(p *whereBodyParser, s *ast.Ident, args map[string]int) Addable {
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
	addAddableGenerator(func(p *whereBodyParser, s *ast.UnaryExpr, args map[string]int) Addable {
		switch s.Op {
		case token.NOT:
			return Not{p.getAddable(s.X, args)}
		default:
			p.c.panicWithPosf(s, "unknown unary operator %s", s.Op)
			return nil
		}
	})
	// Add some time functions
	addAddableGenerator(func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
		funcSelectorExpr, ok := s.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil
		}
		// Check that we call these methods on "time.Time" type.
		if !p.exprIsOfType(funcSelectorExpr.X, "time", "Time") {
			return nil
		}

		// selectorType := p.c.TypeInfo.TypeOf(selectorExpr.X)
		// selectorNamedType, ok := selectorType.(*types.Named)
		// if !ok {
		// 	return nil
		// }
		// // Check that type is "time.Time"
		// if obj := selectorNamedType.Obj(); obj.Pkg() == nil || obj.Pkg().Name() != "time" || obj.Name() != "Time" {
		// 	return nil
		// }

		var op token.Token

		switch funcSelectorExpr.Sel.Name {
		case "After":
			op = token.GTR
		case "Before":
			op = token.LSS
		case "Equal":
			op = token.EQL
		default:
			p.c.panicWithPosf(funcSelectorExpr.Sel, "time function is not supported: %q", funcSelectorExpr.Sel.Name)
		}

		return newAddableComparison(
			p.exprToAddable(funcSelectorExpr.X, args),
			op,
			p.exprToAddable(s.Args[0], args),
		)
	})

	addAddableGenerator(func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
		selector, ok := s.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil
		}

		if selector.Sel.Name != "Now" {
			return nil
		}

		if ident, ok := selector.X.(*ast.Ident); !ok || ident.Name != "time" {
			return nil
		}

		return NewSimple("NOW()")
	})
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

func addAddableGenerator[T ast.Expr](f typedAddableGenerator[T]) {
	var t T
	strTp := typeKey(t)

	addableGenerators[strTp] = append(addableGenerators[strTp], wrapper(f))
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
