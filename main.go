package main

import (
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
								time.Sleep(1 * time.Second)
								select {
								case <-p.Context.Done():
									close(c)
									return
								case c <- r:
								}
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
	s := &graphql.SubscriptableSchema{Schema: schema}
	graphQLHandler := graphqlws.NewHandlerFunc(s, h)
	http.HandleFunc("/", graphQLHandler)
	println("http://localhost:8070")
	// start HTTP server
	if err := http.ListenAndServe(fmt.Sprintf("localhost:%d", 8070), nil); err != nil {
		panic(err)
	}
}
