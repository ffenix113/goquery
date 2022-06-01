package internal

import (
	"fmt"
	"go/ast"
)

// raw is a string representation that will not
// be quoted when added as an argument.
type raw string

type Addable interface {
	String() string
	Args() []any
}

type Simple struct {
	StringVal string
	Arg       []any
}

func (r raw) String() string {
	return string(r)
}

func (raw) Args() []any {
	return nil
}

func NewColumn(name string) *Simple {
	return NewSimple(param, raw("bun.Ident(helper.ColumnName(\""+name+"\"))"))
}

func NewSimple(val string, args ...any) *Simple {
	return &Simple{
		StringVal: val,
		Arg:       args,
	}
}

func (s Simple) String() string {
	return s.StringVal
}

func (s Simple) Args() []any {
	return s.Arg
}

type Expression struct{ ast.Expr }

func (e Expression) String() string {
	switch t := e.Expr.(type) {
	case *ast.SelectorExpr:
		return t.X.(*ast.Ident).Name + "." + t.Sel.Name
	case *ast.BasicLit:
		return "?"
	default:
		panic(fmt.Sprintf("unknown expression to stringify: %T", e.Expr))
	}
}

func (e Expression) Args() []any {
	return nil
}

type Parens struct {
	Addable
}

func (p Parens) String() string {
	return "(" + p.Addable.String() + ")"
}

type Not struct {
	Addable
}

func (n Not) String() string {
	return "not (" + n.Addable.String() + ")"
}
