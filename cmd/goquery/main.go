package main

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"

	"github.com/ffenix113/goquery/internal"
)

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

	c := internal.Context{
		Data: map[string]map[token.Position]internal.QueryData{},
	}

	if err := c.ParseFile(file); err != nil {
		panic(err)
	}

	// ast.Print(c.fileSet, c.astFile)

	ast.Walk(&c, c.AstFile)

	internal.WriteBase(&c, file)
}
