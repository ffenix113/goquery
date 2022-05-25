# Entity

.NET IQueryable-like query library for Go.

Currently it provides minimal support for required interface functionality,
and even less for filtering options.

### Getting started
**Note**: This project requires Go version 1.18 or higher as it uses generics.

To install the executable that will generate the queries run:
```bash
go install github.com/ffenix113/entity/cmd/entity@main
```

After this you can write code like this:
```go
//go:generate entity
package main

import (
	"context"

	"github.com/ffenix113/goquery"
)

type User struct {
	ID   int
	Name string
}

func main() {
	// Get *bun.DB connection
	db := getDB()
	// Create factory that will create Queryable for User.
	queryableUserFactory := goquery.NewFactory[User](db)
	// Create Queryable for User.
	//
	// This is needed because we need to 
	// have a new select query for each DBSet.
	// Otherwise, all methods on dbSet will be executed on 
	// the same query shared between all instances of Queryable.
	//
	// Or you can also do 
	// `queryableUserFactory.New(db.NewSelect().Model(...))`
	// to set the base query. Then when calling 
	// `queryable.Query()` resulting query will 
	// already contain the model.
	queryable := queryableUserFactory.New()

	// Add some filter expression
	queryable.Where(func(user User) bool {
		return user.Name == "John"
	})
	// Add some more filters for the same query
	queryable.Where(func(user User) bool {
		return user.ID == 1 || user.ID >= 5
	})

	// Execute query and get result back.
	var user User
	err := queryable.Query().Model(&user).Scan(context.Background())
	if err != nil {
        // Handle error
    }
}
```

Now run `go generate` which should result in a new file `<filename>_base.go`
which contains necessary definitions for resulting SQL queries.

### FAQ

Q: Is it production ready?  
A: Interfaces might change while this repo is under development so use at your own risk.

