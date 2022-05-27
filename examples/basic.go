//go:generate go run ../cmd/goquery/...

package examples

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	entity "github.com/ffenix113/goquery"
)

type Book struct {
	Title    string
	Author   string
	Released int
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
