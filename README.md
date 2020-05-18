run with
```
go mod download
go run main.go
```

things to notice

- only works with playground and not with graphiql because of a issue with the `handler` package
- to create channels you MUST use `chan interface{}`, because internally as `graphql-go` we use a type cast
- currently you must define a Resolver that returns `params.Root` or you must return the top level subscription field
