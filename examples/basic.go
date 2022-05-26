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

func basic() {
	dbSource := os.Getenv("DB_SOURCE")
	sqldb, err := sql.Open(sqliteshim.ShimName, dbSource)
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	dbBookSet := entity.NewFactory[Book](db)

	setImpl := dbBookSet.New()

	var c a

	setImpl.Where(func(book Book) bool {
		// Direct comparison with a constant works.
		return book.Released == 2003 || book.Released == 2000
	}).Where(func(bk Book) bool { // Chaining also works.
		// Comparison against outside variables works
		// as long as outside variable is provided as argument
		// to `Where` method.
		return bk.Title == c.b.c
	}, c.b.c).Where(filter)

	var resultBook Book
	if err := setImpl.Query().Model(&resultBook).
		Scan(context.Background()); err != nil {
		panic(err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "\t")
	enc.Encode(resultBook)
}

func filter(b Book) bool {
	return b.Released > 0
}
