package entity

import (
	"context"
	"runtime"
	"strconv"

	"github.com/uptrace/bun"
)

type dbSetEntity[T any] struct {
	callsMap    Calls
	helper      Helper
	db          *bun.DB
	selectQuery *bun.SelectQuery
}

func (e *dbSetEntity[T]) New() DBSet[T] {
	newSet := *e

	newSet.selectQuery = e.db.NewSelect()

	return &newSet
}

func (e *dbSetEntity[T]) Where(_ func(val T) bool, args ...any) DBSet[T] {
	// Can't out-magic the language...
	// We still need to get the caller to fetch proper executor.
	_, file, line, _ := runtime.Caller(1)

	where, ok := e.callsMap.Where[Caller{file, line}]
	if !ok {
		panic("no where function. Definitely a bug! caller: " + file + ":" + strconv.Itoa(line))
	}

	where(e.helper, e.selectQuery, args...)

	return e
}

func (e *dbSetEntity[T]) First(ctx context.Context) (T, error) {
	var model T

	return model, e.selectQuery.Model(&model).Limit(1).Scan(ctx)
}
