package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

const projectName = "goquery"
const interfaceName = "Queryable"

type Context struct {
	fileSet *token.FileSet
	astFile ast.Node

	PackageName string

	Data map[string]map[token.Position]QueryData // EntityName(type Arg) -> caller -> query

	typeInfo types.Info
}

type QueryData struct {
	Query string
	Args  []string
}

func newQueryData(addable Addable) QueryData {
	args := addable.Args()

	strArgs := make([]string, 0, len(args))
	for _, arg := range args {
		switch typed := arg.(type) {
		case raw:
			strArgs = append(strArgs, string(typed))
		case string:
			strArgs = append(strArgs, strconv.Quote(typed))
		default:
			strArgs = append(strArgs, fmt.Sprint(arg))
		}
	}

	return QueryData{
		Query: addable.String(),
		Args:  strArgs,
	}
}

func (c *Context) Visit(node ast.Node) (w ast.Visitor) {
	switch n := node.(type) {
	case *ast.File:
		c.PackageName = n.Name.Name
	case *ast.CallExpr:
		selector, ok := n.Fun.(*ast.SelectorExpr)
		if !ok {
			break
		}

		if selector.Sel.Name != "Where" {
			break
		}

		selectorType := c.typeInfo.TypeOf(selector.X)
		if selectorType.(*types.Named).Obj().Name() != interfaceName {
			break
		}

		identType := c.typeInfo.TypeOf(selector.X)

		identTypeObj := identType.(*types.Named).Obj()
		if identTypeObj.Pkg().Name() != projectName || identTypeObj.Name() != interfaceName {
			break
		}

		whereFunc := n.Args[0].(*ast.FuncLit)

		paramName := whereFunc.Type.Params.List[0].Names[0].Name
		bodyParser := whereBodyParser{
			c:         c,
			paramName: paramName,
		}
		// Get type
		typeName := getTypeArgName(identType)
		_ = typeName

		addable := bodyParser.parse(whereFunc.Body)
		typeCalls, ok := c.Data[typeName]
		if !ok {
			typeCalls = make(map[token.Position]QueryData)
			c.Data[typeName] = typeCalls
		}

		typeCalls[c.fileSet.Position(selector.Sel.Pos())] = newQueryData(addable)
	}
	return c
}

func getTypeArgName(identType types.Type) string {
	return identType.(*types.Named).
		TypeArgs().At(0).(*types.Named).Obj().Name()
}

func (p *whereBodyParser) parse(body *ast.BlockStmt) Addable {
	binaryExpr := body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)

	return p.parseBinaryExpression(binaryExpr)
}

type whereBodyParser struct {
	c         *Context
	paramName string
}

func (p *whereBodyParser) parseBinaryExpression(expr *ast.BinaryExpr) Addable {
	return fromBinaryExpr(expr)
}
