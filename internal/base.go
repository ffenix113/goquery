package internal

import (
	"bytes"
	"embed"
	"os"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"
)

//go:embed base.tpl
var baseTplFS embed.FS

var funcMap = template.FuncMap{
	"join": strings.Join,
}

func WriteBase(c *Context, goFilePath string) {
	baseTpl, err := template.New("base.tpl").Funcs(funcMap).ParseFS(baseTplFS, "base.tpl")
	if err != nil {
		panic(err)
	}

	var outputBuf bytes.Buffer

	if err := baseTpl.Execute(&outputBuf, c); err != nil {
		panic(err)
	}

	processed, err := imports.Process("", outputBuf.Bytes(), nil)
	if err != nil {
		_, _ = outputBuf.WriteTo(os.Stdout)
		panic(err)
	}

	baseFile, err := os.OpenFile(createdBaseTplFilePath(goFilePath), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}

	defer baseFile.Close()

	if _, err := baseFile.Write(processed); err != nil {
		panic(err)
	}

}

func createdBaseTplFilePath(goFilePath string) string {
	const suffix = "_goquery.go"
	const testSuffix = "_goquery_test.go"

	if !strings.HasSuffix(goFilePath, ".go") {
		panic("file path must end with '.go'")
	}

	if strings.HasSuffix(goFilePath, "_test.go") {
		return goFilePath[:len(goFilePath)-8] + testSuffix
	}

	return goFilePath[:len(goFilePath)-3] + suffix
}
