package goquery

import (
	"github.com/uptrace/bun"
)

type Queryable[T any] interface {
	// Where adds a filter condition to select query.
	//
	// Note: many of the filtering function that
	// you will use might not be supported.
	// Basic comparison against constants are supported though.
	// See documentation for more info.
	Where(filter func(val T) bool, args ...any) Queryable[T]
	// Query returns a *bun.SelectQuery that is
	// used by this Queryable.
	Query() *bun.SelectQuery
}

type QueryFunc func(h Helper, query *bun.SelectQuery, args ...any)

type Helper interface {
	// ColumnName must return SQL column name for the given field.
	// Field name will be given as defined in a Go struct.
	ColumnName(fieldName string) string
}

type Caller struct {
	File string
	Line int
}

type Calls struct {
	Where map[Caller]QueryFunc
}
