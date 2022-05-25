//go:generate go run /home/eugene/GoProjects/entity/cmd/entity/...

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/ffenix113/entity"
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

	dbBookSet := entity.New[Book](db, BookHelper{})

	setImpl := dbBookSet.New()

	setImpl.Where(func(book Book) bool {
		return book.Released == 2003
	})

	resultBook, err := setImpl.First(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", resultBook)
}

type BookHelper struct{}

func (b BookHelper) ColumnName(name string) string {
	return strings.ToLower(name)
}
