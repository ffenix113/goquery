package internal

import (
	"go/ast"
)

type TimePackage struct{}
type TimeType struct{}

func (TimePackage) now(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	return NewSimple("NOW()")
}

func (TimeType) cmp(op string) typedGenerator[*ast.CallExpr] {
	return func(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
		return newBinary(
			p.exprToAddable(s.Fun.(*ast.SelectorExpr).X, args),
			op,
			p.exprToAddable(s.Args[0], args),
		)
	}
}
