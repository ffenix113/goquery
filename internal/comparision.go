package internal

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"math/bits"
	"strconv"
	"strings"
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
			buf.WriteString(" AND ")
		}
		buf.WriteString(comparison.String())
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
	return fmt.Sprintf("%s OR %s", c.left.String(), c.right.String())
}

func (c comparisonOr) Args() []any {
	return append(c.left.Args(), c.right.Args()...)
}

type binary struct {
	Op    string
	Left  Addable
	Right Addable
}

func newComparison(parser *whereBodyParser, binaryExpr *ast.BinaryExpr) Addable {
	return newBinary(
		parser.exprToAddable(binaryExpr.X, parser.args),
		tokenToOperation(binaryExpr.Op),
		parser.exprToAddable(binaryExpr.Y, parser.args),
	)
}

func newBinary(left Addable, op string, right Addable) Addable {
	cmp := &binary{
		Left:  left,
		Right: right,
		Op:    op,
	}

	return cmp
}

func tokenToOperation(cmpToken token.Token) string {
	switch cmpToken {
	case token.EQL:
		return "="
	case token.GTR:
		return ">"
	case token.LSS:
		return "<"
	case token.NEQ:
		return "!="
	case token.LEQ:
		return "<="
	case token.GEQ:
		return ">="
	case token.SUB:
		return "-"
	case token.ADD:
		return "+"
	case token.MUL:
		return "*"
	case token.QUO:
		return "/"
	default:
		panic("unsupported operator: " + cmpToken.String())
	}
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
		c.panicWithPosf(expr, "unsupported expression type %T to get name from", expr)
		return ""
	}
}

func getArg(val *ast.BasicLit) (arg any) {
	switch val.Kind {
	case token.INT:
		arg, _ = strconv.ParseInt(val.Value, 10, bits.UintSize)
	case token.FLOAT:
		arg, _ = strconv.ParseFloat(val.Value, bits.UintSize)
	case token.STRING:
		arg, _ = strconv.Unquote(val.Value)
	default:
		panic(fmt.Sprintf("unsupported type: %s", val.Kind))
	}

	return arg
}

func (c *binary) String() string {
	var buf strings.Builder

	buf.WriteString(c.Left.String())
	buf.WriteByte(' ')
	buf.WriteString(c.Op)
	buf.WriteByte(' ')
	buf.WriteString(c.Right.String())

	return buf.String()
}

func (c *binary) Args() []any {
	return append(append([]any(nil), c.Left.Args()...), c.Right.Args()...)
}

func fromArgs(pos int) raw {
	return raw(fmt.Sprintf("args[%d]", pos))
}

func (c *Context) panicWithPosf(node ast.Node, msg string, args ...any) {
	formattedMsg := fmt.Sprintf(msg, args...)
	if _, ok := node.(ast.Expr); ok {
		var b bytes.Buffer

		if err := printer.Fprint(&b, c.FileSet, node); err != nil {
			fmt.Println(err)
		}

		panic(fmt.Sprintf("%s (expr: `%s`): %s", c.FileSet.Position(node.Pos()).String(), b.String(), formattedMsg))
	}

	panic(fmt.Sprintf("%s: %s", c.FileSet.Position(node.Pos()).String(), formattedMsg))
}
