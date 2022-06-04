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

var typeMethods = map[string]map[string]typedGenerator[*ast.CallExpr]{}
var packageFuncs = map[string]map[string]typedGenerator[*ast.CallExpr]{}

func init() {
	// Basic generators, they are not specific
	// to some type, package or variable/const.
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

	addPackageFuncGenerator("time", "Now", TimePackage{}.now)

	addPackageFuncGenerator("strings", "Contains", StringsPackage{}.contains)
	addPackageFuncGenerator("strings", "ToLower", StringsPackage{}.toLower)
	addPackageFuncGenerator("strings", "ToUpper", StringsPackage{}.toUpper)
	addPackageFuncGenerator("strings", "HasPrefix", StringsPackage{}.hasPrefix)
	addPackageFuncGenerator("strings", "HasSuffix", StringsPackage{}.hasSuffix)

	addTypeFuncGenerator("time.Time", "After", TimeType{}.cmp(tokenToOperation(token.GTR)))
	addTypeFuncGenerator("time.Time", "Before", TimeType{}.cmp(tokenToOperation(token.LSS)))
	addTypeFuncGenerator("time.Time", "Equal", TimeType{}.cmp(tokenToOperation(token.EQL)))

	addTypeMethodGenerators()
	addPackageFuncGenerators()
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
		p.c.panicWithPosf(expr, "don't know how to convert type %q to Addable", strTp)
	}

	for _, generator := range generators {
		if addable := generator(p, expr, args); addable != nil {
			return addable
		}
	}

	p.c.panicWithPosf(expr, "didn't find any suitable generator for type %q", strTp)

	return nil
}

func addGenerator[T ast.Expr](f typedGenerator[T]) {
	var t T
	strTp := typeKey(t)

	addableGenerators[strTp] = append(addableGenerators[strTp], wrapper(f))
}

func addPackageFuncGenerator(packageName string, funcName string, generator typedGenerator[*ast.CallExpr]) {
	mp := packageFuncs[packageName]
	if mp == nil {
		mp = map[string]typedGenerator[*ast.CallExpr]{}
		packageFuncs[packageName] = mp
	}

	mp[funcName] = generator
}

func addPackageFuncGenerators() {
	addGenerator(func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
		selector, ok := s.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil
		}

		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Obj != nil {
			return nil
		}

		funcGenerators, ok := packageFuncs[ident.Name]
		if !ok {
			return nil
		}

		generator, ok := funcGenerators[selector.Sel.Name]
		if !ok {
			return nil
		}

		return generator(p, s, args)
	})
}

func addTypeFuncGenerator(typeName string, funcName string, generator typedGenerator[*ast.CallExpr]) {
	mp := typeMethods[typeName]
	if mp == nil {
		mp = map[string]typedGenerator[*ast.CallExpr]{}
		typeMethods[typeName] = mp
	}

	mp[funcName] = generator
}

func addTypeMethodGenerators() {
	addGenerator(func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
		funcSelectorExpr, ok := s.Fun.(*ast.SelectorExpr)
		if !ok {
			return nil
		}

		strExprType, ok := p.exprType(funcSelectorExpr.X)
		if !ok {
			return nil
		}

		typeMethodMap, ok := typeMethods[strExprType]
		if !ok {
			return nil
		}

		generator, ok := typeMethodMap[funcSelectorExpr.Sel.Name]
		if !ok {
			return nil
		}

		return generator(p, s, args)
	})
}

func typeKey[T ast.Expr](e T) string {
	return fmt.Sprintf("%T", e)
}

func (p *whereBodyParser) exprType(t ast.Expr) (string, bool) {
	tp := p.c.TypeInfo.TypeOf(t)

	namedTp, ok := tp.(*types.Named)
	if !ok {
		return "", false
	}

	if namedTp.Obj().Pkg() == nil {
		return "", false
	}

	return namedTp.Obj().Pkg().Name() + "." + namedTp.Obj().Name(), true
}
