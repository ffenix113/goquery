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

	p *whereBodyParser
}

func newComparison(parser *whereBodyParser, binaryExpr *ast.BinaryExpr) Addable {
	c := &comparison{
		p: parser,
	}

	c.setLeft(binaryExpr.X)
	c.setOp(binaryExpr)
	c.setRight(binaryExpr.Y)

	return c
}

func (c *comparison) setOp(expr *ast.BinaryExpr) {
	op := expr.Op
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
		c.p.c.panicWithPos(expr.Pos(), "unsupported operator: "+op.String())
	}
}

func (c *comparison) setLeft(s ast.Expr) {
	c.Left = c.p.exprToAddable(s, c.p.args)
}

func (c *comparison) setRight(s ast.Expr) {
	c.Right = c.p.exprToAddable(s, c.p.args)
}

func (p *whereBodyParser) fromBinaryExpr(expr *ast.BinaryExpr, args map[string]int) Addable {
	switch expr.Op {
	case token.LOR:
		return newComparisonOr(p.exprToAddable(expr.X, args), p.exprToAddable(expr.Y, args))
	case token.LAND:
		return newComparisonAnd(p.exprToAddable(expr.X, args), p.exprToAddable(expr.Y, args))
	}

	return newComparison(p, expr)
}

func (p *whereBodyParser) exprToAddable(s ast.Expr, args map[string]int) Addable {
	switch s := s.(type) {
	case *ast.BasicLit:
		return NewSimple("?", getArg(s))
	case *ast.BinaryExpr:
		return p.fromBinaryExpr(s, args)
	case *ast.SelectorExpr:
		// Supports more cases for arguments
		gotExprName := exprName(s)
		if !strings.HasPrefix(gotExprName, p.paramName+".") {
			argPos, ok := args[gotExprName]
			if !ok {
				p.c.panicWithPos(s.Pos(), "argument is not provided: "+gotExprName)
			}

			return NewSimple("?", fromArgs(argPos))
		}

		return NewSimple("?", raw("bun.Ident(helper.ColumnName(\""+s.Sel.Name+"\"))"))
	case *ast.ParenExpr:
		return Parens{p.exprToAddable(s.X, args)}
	case *ast.Ident:
		argPos, ok := args[s.Name]
		if !ok {
			p.c.panicWithPos(s.Pos(), "argument is not provided: "+s.Name)
		}

		return NewSimple("?", fromArgs(argPos))
	default:
		p.c.panicWithPos(s.Pos(), fmt.Sprintf("unsupported binary argument type %T", s))

		return nil
	}
}

func exprName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return exprName(expr.X) + "." + expr.Sel.Name
	default:

		panic(fmt.Sprintf("unsupported expression type %T", expr))
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

func fromArgs(pos int) raw {
	return raw(fmt.Sprintf("args[%d]", pos))
}

func (c *Context) panicWithPos(pos token.Pos, msg string) {
	panic(fmt.Sprintf("%s: %s", c.fileSet.Position(pos).String(), msg))
}
