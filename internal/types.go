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

// Wrapper takes an addable and wraps it.
//
// For example if string representation of Addable
// should be concatenated with something else like:
// `Addable.String()` to `'%' || Addable.String() || '%'".
//
// Wrapper defined above would look like this:
//	Wrapper{
//		Addable: addable,
//		StringF: func(a Addable) string {
//			return `'%' || ` + a.String() + ` || '%'`
//		}
//	}
type Wrapper struct {
	Addable
	StringF func(a Addable) string
	ArgsF   func(a Addable) []any
}

func (w Wrapper) String() string {
	if w.StringF != nil {
		return w.StringF(w.Addable)
	}

	return w.Addable.String()
}

func (w Wrapper) Args() []any {
	if w.ArgsF != nil {
		return w.ArgsF(w.Addable)
	}

	return w.Addable.Args()
}
