package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	"net/http"

	"github.com/graph-gophers/graphql-transport-ws/graphqlws"
)

type Map map[string]interface{}

func main() {
	// init graphQL schema
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"hello": &graphql.Field{Type: graphql.String},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name: "Subscription",
			Fields: graphql.Fields{
				"sub_with_object": &graphql.Field{
					Type: graphql.NewObject(graphql.ObjectConfig{
						Name: "Obj",
						Fields: graphql.Fields{
							"field": &graphql.Field{
								Type: graphql.String,
							},
						},
					}),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						println("resolver")
						x, err := json.MarshalIndent(p.Source, "", "  ")
						if err != nil {
							fmt.Println("err", err)
						}
						fmt.Println(string(x))
						return p.Source, nil
					},
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						elements := []Map{
							{"field": "1"},
							{"field": "2"},
							{"field": "3"},
						}
						c := make(chan interface{}) // XXX only works with `chan interface{}`
						go func() {
							for _, r := range elements {
								select {
								case <-p.Context.Done():
									close(c)
									return
								case c <- r:
								}
								time.Sleep(1 * time.Second)
							}
							close(c)
						}()
						return c, nil
					},
				},
			},
		}),
	})

	if err != nil {
		panic(err)
	}

	h := handler.New(&handler.Config{
		Schema:     &schema,
		Pretty:     true,
		Playground: true, // XXX only works with playground (handler package issue)
	})

	// graphQL handler
	s := &SubscriptableSchema{Schema: schema}
	graphQLHandler := graphqlws.NewHandlerFunc(s, h)
	http.HandleFunc("/", graphQLHandler)
	println("http://localhost:8070")
	// start HTTP server
	if err := http.ListenAndServe(fmt.Sprintf("localhost:%d", 8070), nil); err != nil {
		panic(err)
	}
}

// SubscriptableSchema implements `graphql-transport-ws` `GraphQLService` interface: https://github.com/graph-gophers/graphql-transport-ws/blob/40c0484322990a129cac2f2d2763c3315230280c/graphqlws/internal/connection/connection.go#L53
// this struct should be in the Handler package
// you can pass `SubscriptableSchema` to `graphql-transport-ws` `NewHandlerFunc`
type SubscriptableSchema struct {
	Schema     graphql.Schema
	RootObject map[string]interface{}
}

// Subscribe method let you use SubscriptableSchema with graphql-transport-ws https://github.com/graph-gophers/graphql-transport-ws
func (self *SubscriptableSchema) Subscribe(ctx context.Context, queryString string, operationName string, variables map[string]interface{}) (<-chan interface{}, error) {
	c := graphql.Subscribe(graphql.Params{
		Schema:         self.Schema,
		Context:        ctx,
		OperationName:  operationName,
		RequestString:  queryString,
		RootObject:     self.RootObject,
		VariableValues: variables,
	})
	to := make(chan interface{})
	go func() {
		defer close(to)
		for {
			select {
			case <-ctx.Done():
				return
			case res, more := <-c:
				if !more {
					return
				}
				to <- res
			}
		}
	}()
	return to, nil
}
