package main

import (
	"context"

	"github.com/toshim45/demo-hertz-gqlgen/graph"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)


func main() {
	h := graph.NewHandler(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	s := server.Default()
	s.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		q := c.Query("q")
		c.String(consts.StatusOK, "Pong!!! "+q)
	})

	s.POST("/graphql", h.ServeHTTP)

	s.Spin()
}
