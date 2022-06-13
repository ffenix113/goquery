package internal

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"strconv"

	"golang.org/x/tools/go/packages"
)

const ProjectName = "goquery"
const InterfaceName = "Queryable"

type Context struct {
	FileSet *token.FileSet
	AstFile *ast.File

	PackageName string

	Data map[string]map[token.Position]QueryData // EntityName(type Arg) -> caller -> query

	TypeInfo *types.Info
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
	c.FileSet = token.NewFileSet()
	c.AstFile, c.TypeInfo = c.getTypeInfo(filePath, c.FileSet)

	// _ = ast.Print(FileSet, AstFile)

	return nil
}

func (c *Context) getTypeInfo(filePath string, fileSet *token.FileSet) (astFile *ast.File, typesInfo *types.Info) {
	pkgs, err := packages.Load(&packages.Config{
		Tests: true,
		Fset:  fileSet,
		Mode:  packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
	}, filepath.Dir(filePath))
	if err != nil {
		panic(err)
	}

	for _, pkg := range pkgs {
		for i, fileName := range pkg.GoFiles {
			if fileName == filePath {
				return pkg.Syntax[i], pkg.TypesInfo
			}
		}
	}

	panic("file not found in parsed packages")
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
			c.panicWithPosf(expr, "cannot generate filter from field")
		default:
			c.panicWithPosf(argType, "cannot parse function from type %T", declType)
		}
	case *ast.FuncLit:
		return argType
	default:
		c.panicWithPosf(expr, "don't know how to unwrap type %T to function", expr)
	}
	return nil
}

func (p *whereBodyParser) parse(body *ast.BlockStmt) Addable {
	returnStmt, ok := body.List[0].(*ast.ReturnStmt)
	if !ok {
		p.c.panicWithPosf(body.List[0], "filter function is expected to only have single return statement")
	}

	switch tpd := returnStmt.Results[0].(type) {
	case *ast.BinaryExpr:
		return p.parseBinaryExpression(tpd)
	case *ast.SelectorExpr:
		return p.parseSelectorExpression(tpd)
	case *ast.UnaryExpr:
		return p.parseUnaryExpression(tpd)
	case *ast.CallExpr:
		return p.parseCallExpression(tpd)
	default:
		p.c.panicWithPosf(returnStmt.Results[0], "expression with type %T cannot be used as filter currently", tpd)
		return nil
	}
}

type whereBodyParser struct {
	c         *Context
	paramName string
	args      map[string]int
}

func (p *whereBodyParser) parseBinaryExpression(expr *ast.BinaryExpr) Addable {
	return p.fromBinaryExpr(expr, p.args)
}

func (p *whereBodyParser) parseSelectorExpression(expr *ast.SelectorExpr) Addable {
	receiver, ok := expr.X.(*ast.Ident)
	if !ok || receiver.Name != p.paramName {
		p.c.panicWithPosf(expr, "only possible to to select field from parameter")
		return nil
	}

	cmp := binary{
		Left:  NewColumn(expr.Sel.Name),
		Right: NewSimple(param, true),
		Op:    tokenToOperation(token.EQL),
	}

	return &cmp
}

func (p *whereBodyParser) parseUnaryExpression(expr *ast.UnaryExpr) Addable {
	return p.getAddable(expr, p.args)
}

func (p *whereBodyParser) parseCallExpression(expr *ast.CallExpr) Addable {
	return p.getAddable(expr, p.args)
}
