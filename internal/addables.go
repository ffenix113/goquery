package internal

import (
	"fmt"
	"go/ast"
	"go/token"
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
				p.c.panicWithPos(s, "argument is not provided: "+gotExprName)
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
			p.c.panicWithPos(s, "argument is not provided: "+s.Name)
		}

		return NewSimple(param, fromArgs(argPos))
	})
	addAddableGenerator(func(p *whereBodyParser, s *ast.UnaryExpr, args map[string]int) Addable {
		switch s.Op {
		case token.NOT:
			return Not{p.getAddable(s.X, args)}
		default:
			p.c.panicWithPos(s, fmt.Sprintf("unknown unary operator %s", s.Op))
			return nil
		}
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
		p.c.panicWithPos(expr, "don't know how to convert type "+strTp)
	}

	for _, generator := range generators {
		if addable := generator(p, expr, args); addable != nil {
			return addable
		}
	}

	p.c.panicWithPos(expr, "didn't find any suitable generator for type "+strTp)

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
