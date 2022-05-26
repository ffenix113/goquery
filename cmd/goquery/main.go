package main

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
)

const projectName = "goquery"
const interfaceName = "Queryable"

func main() {
	file := os.Getenv("GOFILE")
	if file == "" {
		file = os.Args[1]
	}

	var err error
	file, err = filepath.Abs(file)
	if err != nil {
		panic(err)
	}

	c := Context{
		Data: map[string]map[token.Position]QueryData{},
	}

	if err := c.parseFile(file); err != nil {
		panic(err)
	}

	// ast.Print(c.fileSet, c.astFile)

	ast.Walk(&c, c.astFile)

	WriteBase(&c, file)
}

func (c *Context) parseFile(filePath string) error {
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

	// _ = ast.Print(fileSet, astFile)

	c.fileSet = fileSet
	c.astFile = astFile
	c.typeInfo = info

	return nil
}
