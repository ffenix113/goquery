//go:generate go run ../cmd/goquery/main.go

package internal_test

import (
	"context"
	"database/sql"
	"math"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"

	"github.com/ffenix113/goquery"
)

type Extensive struct {
	StringCol  string
	StringCol2 string
	IntCol     int
	TimeCol    time.Time
}

const packageConst = 1

func TestSimpleAddables(t *testing.T) {
	tests := []struct {
		name   string
		f      func(q goquery.Queryable[*Extensive])
		result string
		args   []any
	}{
		{
			name: "cmps",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.StringCol == "eql" && e.StringCol > "gt" && e.StringCol < "lt" &&
						e.StringCol >= "gte" && e.IntCol <= 5
				})
			},
			result: `("string_col" = 'eql' AND "string_col" > 'gt' AND "string_col" < 'lt' AND "string_col" >= 'gte' AND "int_col" <= 5)`,
		},
		{
			name: "binary cmps",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.StringCol == "eql" && (e.StringCol > "gt" || e.StringCol == "another")
				})
			},
			result: `("string_col" = 'eql' AND ("string_col" > 'gt' OR "string_col" = 'another'))`,
		},
		{
			name: "arguments",
			f: func(q goquery.Queryable[*Extensive]) {
				stringArg, intVar := "arg", 88
				q.Where(func(e *Extensive) bool {
					return e.StringCol == stringArg || e.StringCol == e.StringCol2 && e.IntCol >= intVar
				}, stringArg, intVar)
			},
			result: `("string_col" = 'arg' OR "string_col" = "string_col2" AND "int_col" >= 88)`,
		},
		{
			name: "in",
			f: func(q goquery.Queryable[*Extensive]) {
				stringArgs := []string{"1", "2"}
				q.Where(func(e *Extensive) bool {
					return goquery.In(e.StringCol, stringArgs)
				}, stringArgs)
			},
			result: `("string_col" IN ('1', '2'))`,
		},
		{
			name: "is null",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return goquery.IsNull(e.StringCol)
				})
			},
			result: `("string_col" IS NULL)`,
		},
		{
			name: "string concat",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.StringCol == "1"+"2"
				})
			},
			result: `("string_col" = '1' || '2')`,
		},
		{
			name: "duration mult",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.TimeCol.Add(-3*time.Second) == time.Now()
				})
			},
			result: `("time_col" + -3 * INTERVAL '1 second' = NOW())`,
		},
		{
			name: "with const simple",
			f: func(q goquery.Queryable[*Extensive]) {
				const val = 0
				q.Where(func(e *Extensive) bool {
					return e.IntCol == val
				})
			},
			result: `("int_col" = 0)`,
		},
		{
			name: "with const from package-level same file",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.IntCol == packageConst
				})
			},
			result: `("int_col" = 1)`,
		},
		{
			name: "with const from package-level different file",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.IntCol == anotherFileConst
				})
			},
			result: `("int_col" = 3)`,
		},
		{
			name: "with const from another package",
			f: func(q goquery.Queryable[*Extensive]) {
				q.Where(func(e *Extensive) bool {
					return e.IntCol == math.MaxInt8
				})
			},
			result: `("int_col" = 127)`,
		},
	}

	for _, test := range tests {
		test := test
		db := getDB(t)

		factory := goquery.NewFactory[*Extensive](db)

		t.Run(test.name, func(t *testing.T) {
			q := factory.New()
			test.f(q)

			var wrapper iconnWrapper
			_, err := q.Query().Conn(&wrapper).Exec(context.Background())
			require.NoError(t, err)

			assert.Equal(t, test.result, wrapper.query[len("select * where "):])
			assert.Equal(t, test.args, wrapper.args)
		})
	}
}

func getDB(t testing.TB) *bun.DB {
	dbSource := os.Getenv("DB_SOURCE")
	if dbSource == "" {
		dbSource = "file::memory:?cache=shared"
	}

	sqldb, err := sql.Open(sqliteshim.ShimName, dbSource)
	require.NoError(t, err)

	sqldb.SetMaxIdleConns(1000)
	sqldb.SetConnMaxLifetime(0)

	t.Cleanup(func() {
		require.NoError(t, sqldb.Close())
	})

	return bun.NewDB(sqldb, sqlitedialect.New())
}

type iconnWrapper struct {
	bun.IConn
	query string
	args  []any
}

func (w *iconnWrapper) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	w.query = query
	w.args = args

	return nil, nil
}
