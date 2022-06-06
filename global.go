package goquery

import (
	"fmt"
	"reflect"

	"github.com/uptrace/bun"
)

var globalCallsMap = map[reflect.Type]Calls{}

type Factory[T any] interface {
	// New creates new Queryable.
	//
	// This method accepts optional base select query
	// to specify model for example.
	New(...*bun.SelectQuery) Queryable[T]
}

func NewFactory[T any](db *bun.DB, helper ...Helper) Factory[T] {
	var t T
	if tp := reflect.TypeOf(t); tp.Kind() != reflect.Pointer || tp.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("input type argument must be a pointer to a struct, but got %T", t))
	}

	var selectedHelper Helper
	if len(helper) > 0 {
		selectedHelper = helper[0]
	} else {
		selectedHelper = NewBunHelper[T](db)
	}

	return &queryable[T]{
		callsMap: getCallMapFromGlobal[T](),
		helper:   selectedHelper,
		db:       db,
	}
}

// DO NOT USE: this is only for generated code!
func AddToGlobalEntity[T any](callsMap Calls) {
	var argType T
	typeArg := reflect.TypeOf(argType)
	if _, ok := globalCallsMap[typeArg]; !ok {
		globalCallsMap[typeArg] = callsMap

		return
	}

	callers := getCallMapFromGlobal[T]()

	for caller, queryFunc := range callsMap.Where {
		callers.Where[caller] = queryFunc
	}
}

func getCallMapFromGlobal[T any]() Calls {
	var argType T
	typeArg := reflect.TypeOf(argType)

	callMap, ok := globalCallsMap[typeArg]
	if !ok {
		panic("global entity not found for type " + typeArg.String())
	}

	return callMap
}
