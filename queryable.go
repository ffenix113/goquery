package goquery

import (
	"runtime"
	"strconv"

	"github.com/uptrace/bun"
)

type queryable[T any] struct {
	callsMap    Calls
	helper      Helper
	db          *bun.DB
	selectQuery *bun.SelectQuery
}

func (e *queryable[T]) New(query ...*bun.SelectQuery) Queryable[T] {
	newSet := *e

	if len(query) > 0 {
		newSet.selectQuery = query[0]
	} else {
		newSet.selectQuery = e.db.NewSelect()
	}

	return &newSet
}

func (e *queryable[T]) Where(_ func(val T) bool, args ...any) Queryable[T] {
	// Can't out-magic the language...
	// We still need to get the caller to fetch proper executor.
	_, file, line, _ := runtime.Caller(1)

	where, ok := e.callsMap.Where[Caller{file, line}]
	if !ok {
		panic("no 'Where' function. Definitely a bug! caller: " + file + ":" + strconv.Itoa(line))
	}

	where(e.helper, e.selectQuery, args...)

	return e
}

func (e *queryable[T]) Query() *bun.SelectQuery {
	return e.selectQuery
}
