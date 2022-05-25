package entity

import (
	"reflect"

	"github.com/uptrace/bun"
)

var globalCallsMap = map[reflect.Type]any{}

type DBSetFactory[T any] interface {
	New() DBSet[T]
}

func New[T any](db *bun.DB, helper Helper) DBSetFactory[T] {
	return &dbSetEntity[T]{
		callsMap: getCallMapFromGlobal[T](),
		helper:   helper,
		db:       db,
	}
}

// DO NOT USE: this is only for generated code!
func SetGlobalEntity[T any](callsMap Calls) {
	var argType T
	typeArg := reflect.TypeOf(argType)
	if _, ok := globalCallsMap[typeArg]; ok {
		panic("global entity already set for type: " + typeArg.String())
	}

	globalCallsMap[typeArg] = callsMap
}

func getCallMapFromGlobal[T any]() Calls {
	var argType T
	typeArg := reflect.TypeOf(argType)

	callMap, ok := globalCallsMap[typeArg]
	if !ok {
		panic("global entity not found for type " + typeArg.String())
	}

	return callMap.(Calls)
}
