package internal

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"
)

const ProjectName = "goquery"
const InterfaceName = "Queryable"

type Context struct {
	FileSet *token.FileSet
	AstFile ast.Node

	PackageName string

	Data map[string]map[token.Position]QueryData // EntityName(type Arg) -> caller -> query

	TypeInfo types.Info
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

func (c *Context) ParseFile(filePath string) error {
	fileSet := token.NewFileSet()

	astFile, err := parser.ParseFile(fileSet, filePath, nil, 0)
	if err != nil {
		return err
	}

	// Type-check the package.
	// We create an empty map for each kind of input
	// we're interested in, and Check populates them.
	info := types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Instances: make(map[*ast.Ident]types.Instance),
		Defs:      make(map[*ast.Ident]types.Object),
	}

	conf := types.Config{
		// Importer: importer.Default(), // FIXME: Try to use Default importer
		Importer: importer.ForCompiler(fileSet, "source", nil),
	}

	dir := filepath.Dir(filePath)
	_, err = conf.Check(dir, fileSet, []*ast.File{astFile}, &info)
	if err != nil {
		return err
	}

	// _ = ast.Print(FileSet, AstFile)

	c.FileSet = fileSet
	c.AstFile = astFile
	c.TypeInfo = info

	return nil
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

		identType := c.TypeInfo.TypeOf(selector.X)

		identTypeObj := identType.(*types.Named).Obj()
		if identTypeObj.Pkg().Name() != ProjectName || identTypeObj.Name() != InterfaceName {
			break
		}

		whereFunc := c.unwrapArgFunc(n.Args[0])

		paramName := whereFunc.Type.Params.List[0].Names[0].Name
		bodyParser := whereBodyParser{
			c:         c,
			paramName: paramName,
			args:      c.getArgNames(n.Args[1:]...),
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

		typeCalls[c.FileSet.Position(selector.Sel.Pos())] = newQueryData(addable)
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

func (c *Context) getArgNames(exprs ...ast.Expr) map[string]int {
	if len(exprs) == 0 {
		return nil
	}

	names := make(map[string]int, len(exprs))
	for i, expr := range exprs {
		names[c.exprName(expr)] = i
	}

	return names
}

func (c *Context) unwrapArgFunc(expr ast.Expr) *ast.FuncLit {
	switch argType := expr.(type) {
	case *ast.Ident:
		switch declType := argType.Obj.Decl.(type) {
		case *ast.FuncDecl:
			return &ast.FuncLit{
				Type: declType.Type,
				Body: declType.Body,
			}
		case *ast.AssignStmt:
			return c.unwrapArgFunc(declType.Rhs[0])
		case *ast.Field:
			c.panicWithPos(expr, "cannot generate filter from field")
		default:
			c.panicWithPos(argType, fmt.Sprintf("cannot parse function from type %T", declType))
		}
	case *ast.FuncLit:
		return argType
	default:
		c.panicWithPos(expr, fmt.Sprintf("don't know how to unwrap type %T to function", expr))
	}
	return nil
}

func (p *whereBodyParser) parse(body *ast.BlockStmt) Addable {
	returnStmt, ok := body.List[0].(*ast.ReturnStmt)
	if !ok {
		p.c.panicWithPos(body.List[0], "filter function is expected to only have single return statement")
	}

	binaryExpr, ok := returnStmt.Results[0].(*ast.BinaryExpr)
	if !ok {
		p.c.panicWithPos(returnStmt.Results[0], "only binary expressions are supported currently")
	}

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
