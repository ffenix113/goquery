//go:generate go run ../cmd/goquery/main.go

package internal

import (
	"context"
	"database/sql"
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
