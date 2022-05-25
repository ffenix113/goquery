package entity

import (
	"context"

	"github.com/uptrace/bun"
)

type DBSet[T any] interface {
	Where(filter func(val T) bool, args ...any) DBSet[T]
	First(ctx context.Context) (T, error)
}

type QueryFunc func(h Helper, query *bun.SelectQuery, args ...any)

type Helper interface {
	ColumnName(name string) string
}

type Caller struct {
	File string
	Line int
}

type Calls struct {
	Where map[Caller]QueryFunc
}
