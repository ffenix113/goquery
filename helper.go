package goquery

import (
	"reflect"

	"github.com/uptrace/bun"
)

var _ Helper = bunHelper{}

type bunHelper struct {
	fieldMap map[string]string
}

func NewBunHelper[T any](db *bun.DB) Helper {
	var t T

	tables := db.Dialect().Tables()
	tables.Register(&t)

	table := tables.Get(reflect.TypeOf(t))

	fieldMap := make(map[string]string)

	for sqlColumnName, structField := range table.FieldMap {
		fieldMap[structField.GoName] = sqlColumnName
	}

	return bunHelper{fieldMap: fieldMap}
}

func (b bunHelper) ColumnName(name string) string {
	columnName, ok := b.fieldMap[name]
	if !ok {
		panic("unknown field name: " + name)
	}

	return columnName
}
