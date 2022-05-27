package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
)

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

		identType := c.typeInfo.TypeOf(selector.X)

		identTypeObj := identType.(*types.Named).Obj()
		if identTypeObj.Pkg().Name() != projectName || identTypeObj.Name() != interfaceName {
			break
		}

		var whereFunc *ast.FuncLit
		switch argType := n.Args[0].(type) {
		case *ast.Ident:
			funcDecl := argType.Obj.Decl.(*ast.FuncDecl)
			whereFunc = &ast.FuncLit{
				Type: funcDecl.Type,
				Body: funcDecl.Body,
			}
		case *ast.FuncLit:
			whereFunc = argType
		}

		paramName := whereFunc.Type.Params.List[0].Names[0].Name
		bodyParser := whereBodyParser{
			c:         c,
			paramName: paramName,
			args:      getArgNames(n.Args[1:]...),
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
	argType := identType.(*types.Named).TypeArgs().At(0)

	ptr, isPtr := argType.(*types.Pointer)
	for isPtr {
		argType = ptr.Elem()
		ptr, isPtr = argType.(*types.Pointer)
	}

	return argType.(*types.Named).Obj().Name()
}

func getArgNames(exprs ...ast.Expr) map[string]int {
	if len(exprs) == 0 {
		return nil
	}

	names := make(map[string]int, len(exprs))
	for i, expr := range exprs {
		names[exprName(expr)] = i
	}

	return names
}

func (p *whereBodyParser) parse(body *ast.BlockStmt) Addable {
	binaryExpr := body.List[0].(*ast.ReturnStmt).Results[0].(*ast.BinaryExpr)

	return p.parseBinaryExpression(binaryExpr)
}

type whereBodyParser struct {
	c         *Context
	paramName string
	args      map[string]int
}

func (p *whereBodyParser) parseBinaryExpression(expr *ast.BinaryExpr) Addable {
	return p.fromBinaryExpr(expr, p.args)
}
