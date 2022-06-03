# GoQuery

.NET IQueryable-like query library for Go.

This generator allows you to write SQL query with ordinary functions:
```go
q = q.Where(func(p Product) bool { return p.CategoryId == 1 && p.UnitsInStock < 10 })
// And then query value with normal *bun.DB functionality
var product Product
err := q.Query().Model(&product).Scan(context.Background())
```
, just like you would be able with EF Core framework in .NET:
```cs
var products = context.Prducts.Where(p => p.CategoryId == 1 && p.UnitsInStock < 10);
```

### Getting started
**Note**: This project requires Go version 1.18 or higher as it uses generics.

To install the executable that will generate the queries run:
```bash
go install github.com/ffenix113/goquery/cmd/goquery@main
```

After this you can write code like this:
```go
//go:generate goquery
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
	// Get *bun.DB connection somehow
	db := getDB()
	// Create factory that will create Queryable for User.
	queryableUserFactory := goquery.NewFactory[User](db)
	// Create Queryable for User.
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
Please see `examples` package to see more uses and available functionality.

* Basic comparisons to constants and most of binary expressions.
```go
queryable.Where(func(user User) bool {
    return (user.Name == "John" && user.ID == 1) || user.ID >= 4
})
```
* Compare to true/false
```go
queryable.Where(func(book *Book) bool {
    // Same for false
    return book.IsSelling == true
})
```
* Use just boolean field from model and `!` operator
```go
queryable.Where(func(book *Book) bool {
    return book.IsSelling || !book.IsSelling
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
* Some minor time operations are supported(`Equal`, `Before` and `After`).
  (with `Add` method in the future)
```go
queryable.Where(func(user User) bool {
    return user.RegisteredAt.Before(time.Now()) || time.Now().After(user.NextUpdate)
})
```
* Some `strings` functions(`ToUpper`, `ToLower`, `Contains`, `HasPrefix` and `HasSuffix`).
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
Parser uses file and line number to understand which function should be called.
* Closures will work only if correct argument is provided to `Where` method.
* Only `*bun.DB` is supported as query execution mechanism.
* Passing fields(i.e. from structs and arguments) is not supported, 
as it will not be possible to provide right caller information.
