package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unsafe"
)

type comparisonsAnd []Addable

func newComparisonAnd(comparisons ...Addable) comparisonsAnd {
	return comparisonsAnd(comparisons)
}

func (c comparisonsAnd) String() string {
	var buf strings.Builder

	for _, comparison := range c {
		if buf.Len() > 0 {
			buf.WriteString(" and ")
		}
		buf.WriteByte('(')
		buf.WriteString(comparison.String())
		buf.WriteByte(')')
	}

	return buf.String()
}

func (c comparisonsAnd) Args() []any {
	var args []any

	for _, comparison := range c {
		args = append(args, comparison.Args()...)
	}

	return args
}

type comparisonOr struct {
	left, right Addable
}

func newComparisonOr(left, right Addable) comparisonOr {
	return comparisonOr{left, right}
}

func (c comparisonOr) String() string {
	return fmt.Sprintf("(%s) or (%s)", c.left.String(), c.right.String())
}

func (c comparisonOr) Args() []any {
	return append(c.left.Args(), c.right.Args()...)
}

type comparison struct {
	Op    string
	Left  Addable
	Right Addable
	Arg   any // we will populate this with real value, not string representation
}

func newComparison(binaryExpr *ast.BinaryExpr) Addable {
	c := &comparison{}

	c.setLeft(binaryExpr.X)
	c.setOp(binaryExpr.Op)
	c.setRight(binaryExpr.Y)

	return c
}

func (c *comparison) setOp(op token.Token) {
	switch op {
	case token.EQL:
		c.Op = "="
	case token.GTR:
		c.Op = ">"
	case token.LSS:
		c.Op = "<"
	case token.NEQ:
		c.Op = "!="
	case token.LEQ:
		c.Op = "<="
	case token.GEQ:
		c.Op = ">="
	default:
		panic("unsupported operator: " + op.String())
	}
}

func (c *comparison) setLeft(s ast.Expr) {
	c.Left = exprToAddable(s)
}

func (c *comparison) setRight(s ast.Expr) {
	c.Right = exprToAddable(s)
}

func fromBinaryExpr(expr *ast.BinaryExpr) Addable {
	switch expr.Op {
	case token.LOR:
		return newComparisonOr(exprToAddable(expr.X), exprToAddable(expr.Y))
	case token.LAND:
		return newComparisonAnd(exprToAddable(expr.X), exprToAddable(expr.Y))
	}

	return newComparison(expr)
}

func exprToAddable(s ast.Expr) Addable {
	switch s := s.(type) {
	case *ast.BasicLit:
		return NewSimple("?", getArg(s))
	case *ast.BinaryExpr:
		return fromBinaryExpr(s)
	case *ast.SelectorExpr:
		return NewSimple("?", raw("bun.Ident(helper.ColumnName(\""+s.Sel.Name+"\"))"))
	case *ast.ParenExpr:
		// Just unwrap parenthesis.
		return exprToAddable(s.X)
	default:
		panic(fmt.Sprintf("unsupported binary argument type %T", s))
	}
}

func getArg(val *ast.BasicLit) (arg any) {
	switch val.Kind {
	case token.INT:
		arg, _ = strconv.Atoi(val.Value)
	case token.FLOAT:
		arg, _ = strconv.ParseFloat(val.Value, int(unsafe.Sizeof(int(0))))
	case token.STRING:
		arg, _ = strconv.Unquote(val.Value)
	default:
		panic(fmt.Sprintf("unsupported type: %s", val.Kind))
	}

	return arg
}

func (c *comparison) String() string {
	var buf strings.Builder

	buf.WriteString(c.Left.String())
	buf.WriteByte(' ')
	buf.WriteString(c.Op)
	buf.WriteByte(' ')
	buf.WriteString(c.Right.String())

	return buf.String()
}

func (c *comparison) Args() []any {
	return append(append([]any(nil), c.Left.Args()...), c.Right.Args()...)
}
