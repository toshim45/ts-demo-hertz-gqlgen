package main

import (
	"context"
	// "log"
	// "net/http"
	"os"

	// "github.com/99designs/gqlgen/graphql/playground"
	"github.com/toshim45/demo-hertz-gqlgen/graph"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	_ = graph.NewHertzHandler(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	h := server.Default()
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		q := c.Query("q")
		c.String(consts.StatusOK, "Pong!!! "+q)
	})

	h.Spin()
	// log.Println("hertz spinned up!!")

	// http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	// http.Handle("/query", srv)

	// log.Printf("connect to http://localhost:%s/ for GraphQL playground\n", port)
	// log.Fatal(http.ListenAndServe(":"+port, nil))
}
