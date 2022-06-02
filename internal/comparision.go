package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"unsafe"
)

const param = "?"

type comparisonsAnd []Addable

func newComparisonAnd(comparisons ...Addable) comparisonsAnd {
	return comparisons
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
	return newAddableComparison(
		parser.exprToAddable(binaryExpr.X, parser.args),
		binaryExpr.Op,
		parser.exprToAddable(binaryExpr.Y, parser.args),
	)
}

func newAddableComparison(left Addable, op token.Token, right Addable) Addable {
	cmp := &comparison{
		Left:  left,
		Right: right,
	}
	cmp.setOp(op)

	return cmp
}

func (c *comparison) setOp(cmpToken token.Token) {
	switch cmpToken {
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
		panic("unsupported operator: " + cmpToken.String())
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
	return p.getAddable(s, args)
}

func (c *Context) exprName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return c.exprName(expr.X) + "." + expr.Sel.Name
	default:
		c.panicWithPosf(expr, "unsupported expression type %T", expr)
		return ""
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

func (c *Context) panicWithPosf(node ast.Node, msg string, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	panic(fmt.Sprintf("%s: %s", c.FileSet.Position(node.Pos()).String(), formattedMsg))
}
