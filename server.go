package main

import (
	"context"
	"os"

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

	h := graph.NewHandler(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	s := server.Default()
	s.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		q := c.Query("q")
		c.String(consts.StatusOK, "Pong!!! "+q)
	})

	s.POST("/graphql", h.ServeHTTP)

	s.Spin()
}
