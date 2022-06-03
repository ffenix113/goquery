//go:generate go run ../cmd/goquery/...

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	entity "github.com/ffenix113/goquery"
)

type Book struct {
	Title     string
	Author    string
	Released  int
	IsSelling bool
}

type a struct {
	b struct {
		c string
	}
}

var globDB = func() *bun.DB {
	dbSource := os.Getenv("DB_SOURCE")
	sqldb, err := sql.Open(sqliteshim.ShimName, dbSource)
	if err != nil {
		panic(err)
	}

	return bun.NewDB(sqldb, sqlitedialect.New())
}()

func newQueryable[T any]() entity.Queryable[T] {
	return entity.NewFactory[T](globDB).New()
}

func main() {
	// basic()
	// pointerAndMultiline()
	// passingToAFunction()
	// closure()
	// badClosure()
	// compareToTrue()
	// unaryBoolean()
	// someTimeFuncs()
	// stringFuncs()
	// stringPrefixSuffixFuncs()
}

func basic() {
	q := newQueryable[Book]()

	var c a

	q.Where(func(book Book) bool {
		// Direct comparison with a constant works.
		return book.Released == 2003 || book.Released == 2000
	}).Where(func(bk Book) bool { // Chaining also works.
		// Comparison against outside variables works
		// as long as outside variable is provided as argument
		// to `Where` method.
		return bk.Title == c.b.c
	}, c.b.c).Where(filter)
}

func filter(b Book) bool {
	return b.Released > 0
}

func pointerAndMultiline() {
	q := newQueryable[*Book]()

	q.Where(func(book *Book) bool {
		// Direct comparison with a constant works.
		return book.Released == 2003 ||
			book.Released == 2000
	})
}

func passingToAFunction() {
	q := newQueryable[*Book]()

	addWhere(q)
}

func addWhere(q entity.Queryable[*Book]) {
	q.Where(func(b *Book) bool { return b.Title == "AnotherTitle" })
}

// Passing filter as a field will not work
// as it will not be possible to provide
// necessary caller information from runtime.
func filterFromArgument() {
	// q := newQueryable[*Book]()
	//
	// func(f func(b *Book) bool) {
	// 	q.Where(f)
	// }(func(b *Book) bool {
	// 	return b.Title == "AnotherTitle"
	// })
}

func closure() {
	q := newQueryable[*Book]()

	a := 10

	f := func(b *Book) bool {
		return b.Released == a
	}

	// Only the name of the variable should match
	q.Where(f, a)
}

func badClosure() {
	q := newQueryable[*Book]()

	a := 10

	f := func(b *Book) bool {
		return b.Released == a
	}

	{
		// As only name of the variable should currently match
		// it is possible to define wrong usages like this:
		a := "string"
		q.Where(f, a)
	}
}

func compareToTrue() {
	q := newQueryable[*Book]()

	q.Where(func(book *Book) bool {
		return book.IsSelling == true
	})
}

func unaryBoolean() {
	q := newQueryable[*Book]()

	var b bool

	q.Where(func(book *Book) bool {
		return !!b || !book.IsSelling
	}, b)
}

type str struct {
	Time time.Time
}

func someTimeFuncs() {
	q := newQueryable[*str]()
	q.Where(func(s *str) bool {
		return s.Time.After(time.Now()) || time.Now().Equal(time.Now())
	})
}

func stringFuncs() {
	newQueryable[*Book]().Where(func(b *Book) bool {
		return b.Title == strings.ToUpper(b.Title) || strings.Contains(b.Title, "contains")
	})
}

func stringPrefixSuffixFuncs() {
	newQueryable[*Book]().Where(func(b *Book) bool {
		return strings.HasPrefix(b.Author, "prefix") || strings.HasSuffix(b.Title, "suffix")
	})
}

// Just some helpers

func runSingleQuery[T any](q entity.Queryable[T]) (res T) {
	if err := q.Query().Model(&res).
		Scan(context.Background()); err != nil {
		panic(err)
	}

	return res
}

func runAllQuery[T any](q entity.Queryable[T]) (res []T) {
	if err := q.Query().Model(&res).
		Scan(context.Background()); err != nil {
		panic(err)
	}

	return res
}

func encode(val any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	if err := enc.Encode(val); err != nil {
		panic(err)
	}
}
