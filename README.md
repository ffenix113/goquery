# GoQuery

.NET IQueryable-like query library for Go.

Currently, it provides minimal support for required interface functionality,
and even less for filtering options.

### Background

As an example in EF Core it is possible to filter rows by providing expressions:
```cs
var products = context.Prducts.Where(p => p.CategoryId == 1 && p.UnitsInStock < 10);
```
While example is extremely basic and does not show all the features,  
it does represent what this project is trying to achieve.

#### Alternative: semi-manually written queries

In most of the SQL libraries for Go you need to manually write 
field names and comparisons, for example like in [GORM](https://gorm.io/docs/query.html#String-Conditions):
```go
// AND
db.Where("name = ? AND age >= ?", "jinzhu", "22").Find(&users)
// SELECT * FROM users WHERE name = 'jinzhu' AND age >= 22;
```
This means that when field is changed - you need to manually update all the 
queries that use it.

#### Alternative: generate query modifications

There are options to generate query modificators from Go structs or SQL tables 
like [go-queryset](https://github.com/jirfag/go-queryset) and [SQLBoiler](https://github.com/volatiletech/sqlboiler).

Those libraries generate query modifications like this:
```go
// go-queryset: https://github.com/jirfag/go-queryset#select-n-users-with-highest-rating
// Generated from Go struct definitions

var users []User
err := NewUserQuerySet(getGormDB()).
	RatingMarksGte(minMarks).
	OrderDescByRating().
	Limit(N).
	All(&users)
```
```go
// sqlboiler: https://github.com/volatiletech/sqlboiler#select
// Generated from SQL Table definitions

// Type safe variant
pilot, err := models.Pilots(models.PilotWhere.Name.EQ("Tim")).One(ctx, db)
```

Above example are closer to .NET implementation, but they are still not 
real query generation from function definition.

If only there was a way to transform in .NET
```cs
var products = context.Prducts.Where(p => p.CategoryId == 1 && p.UnitsInStock < 10);
```
to something like this in Go:
```go
queryable = queryable.Where(func(p Product) bool {
	return p.CategoryId == 1 && p.UnitsInStock < 10
})
// Query value with normal *bun.DB functionality
var product Product
err := queryable.Query().Model(&product).Scan(context.Background())
```

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

Now run `go generate` which should result in a new file `<filename>_goquery.go`
which contains necessary definitions for resulting SQL queries.

### What this project can currently do
* Basic comparisons to constants and most of binary expressions.
```go
queryable.Where(func(user User) bool {
    return (user.Name == "John" && user.ID == 1) || user.ID >= 4
})
```
* Using simple function as filter.  
Note: closures will not work properly.
```go
filter := func(user User) bool {
    return user.Name == "John"
}
queryable.Where(filter)
```
* Comparisons to other variables.  
In this case the argument must also be provided to `Where` method.
```go
queryable.Where(func(user User) bool {
    return user.Name == someName
}, someName)
```
Arguments to `Where` method should be supplied only once.
Generator will use appropriate argument position from passed ones:
```go
queryable.Where(func(user User) bool {
    return user.Name == someName || user.Name == someName2 || user.Name == someName
}, someName, someName2)
```

* Comparisons to other variables in other structs.
```go
queryable.Where(func(user User) bool {
    return user.Name == anotherUser.Name || user.ID == someStruct.User.ID
}, anotherUser.Name, someStruct.User.ID)
```
* Chaining calls to `Where` method.
```go
queryable.Where(func(user User) bool {
    return (user.Name == "John" && user.ID == 1) || user.ID >= 4
}).Where(func(u User) bool {
    return u.Name == anotherUser.Name || u.ID == anotherUser.ID
}, anotherUser.Name, anotherUser.ID)
```

### Limitations

As a rule of thumb pretty much everything that is not specifically 
mentioned as possible - may not work or break code generation. 
You are welcome to try though!

The biggest limitation that exists for this project is that 
all the possible combinations of functionality must be 
defined manually in the parser, which means that while separately 
some features may work this does not guarantee that together they will also work.

For some other limitation that can be stated:
* `Where` calls **must be** on separate lines!
* Closures will not work, as they by definition require values outside the current scope.
* Only `*bun.DB` is supported as query execution mechanism.
