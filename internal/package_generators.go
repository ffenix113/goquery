package internal

import (
	"go/ast"
)

type StringsPackage struct{}

func (StringsPackage) hasPrefix(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	return newBinary(
		p.exprToAddable(s.Args[0], args),
		"LIKE",
		Wrapper{
			Addable: p.exprToAddable(s.Args[1], args),
			StringF: func(a Addable) string { return a.String() + ` || '%'` },
		},
	)
}

func (StringsPackage) hasSuffix(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	return newBinary(
		p.exprToAddable(s.Args[0], args),
		"LIKE",
		Wrapper{
			Addable: p.exprToAddable(s.Args[1], args),
			StringF: func(a Addable) string { return `'%' || ` + a.String() },
		},
	)
}

func (StringsPackage) contains(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
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
}

func (StringsPackage) toLower(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	return Wrapper{
		Addable: p.exprToAddable(s.Args[0], args),
		StringF: func(a Addable) string { return "lower(" + a.String() + ")" },
	}
}

func (StringsPackage) toUpper(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	return Wrapper{
		Addable: p.exprToAddable(s.Args[0], args),
		StringF: func(a Addable) string { return "upper(" + a.String() + ")" },
	}
}

type GoQueryPackage struct{}

func (GoQueryPackage) isNull(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	return Wrapper{
		Addable: p.exprToAddable(s.Args[0], args),
		StringF: func(a Addable) string {
			return a.String() + " IS NULL"
		},
	}
}

func (GoQueryPackage) in(p *whereBodyParser, s *ast.CallExpr, args map[string]int) Addable {
	if _, ok := args[p.c.exprName(s.Args[1])]; !ok {
		p.c.panicWithPosf(s.Args[1], "argument is not present in Where arguments")
	}

	return Wrapper{
		Addable: p.exprToAddable(s.Args[0], args),
		StringF: func(a Addable) string {
			return a.String() + " IN (?)"
		},
		ArgsF: func(a Addable) []any {
			return append(a.Args(), raw("bun.In("+fromArgs(0)+")"))
		},
	}
}
