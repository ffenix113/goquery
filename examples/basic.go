//go:generate entity

package main

import (
	"context"
	"database/sql"
	"os"

	"github.com/davecgh/go-spew/spew"
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

func main() {
	dbSource := os.Getenv("DB_SOURCE")
	sqldb, err := sql.Open(sqliteshim.ShimName, dbSource)
	if err != nil {
		panic(err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())

	dbBookSet := entity.NewFactory[Book](db)

	setImpl := dbBookSet.New()

	setImpl.Where(func(book Book) bool {
		return book.Released == 2003
	}).Where(func(bk Book) bool {
		return bk.Title == "a"
	})

	var resultBook Book
	if err := setImpl.Query().Model(&resultBook).Scan(context.Background()); err != nil {
		panic(err)
	}

	spew.Dump(resultBook)
}
