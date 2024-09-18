package graph

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
)

type (
	Server struct {
		transports []graphql.Transport
		exec       *executor.Executor
	}

	POST struct{}
)

func New(es graphql.ExecutableSchema) *Server {
	return &Server{
		exec: executor.New(es),
	}
}

func NewHandler(es graphql.ExecutableSchema) *Server {
	srv := New(es)

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	return srv
}

func (s *Server) SetQueryCache(cache graphql.Cache[*ast.QueryDocument]) {
	s.exec.SetQueryCache(cache)
}

func (s *Server) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

func (s *Server) ServeHTTP(c context.Context, r *app.RequestContext) {
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(c, err)
			gqlErr, _ := err.(*gqlerror.Error)
			resp := &graphql.Response{Errors: []*gqlerror.Error{gqlErr}}
			r.JSON(consts.StatusUnprocessableEntity, resp)
		}
	}()

	c = graphql.StartOperationTrace(c)

	POST{}.Do(c, r, s.exec)
}

func statusFor(errs gqlerror.List) int {
	switch errcode.GetErrorKind(errs) {
	case errcode.KindProtocol:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusOK
	}
}

func (h POST) Do(c context.Context, r *app.RequestContext, exec graphql.GraphExecutor) {
	params := &graphql.RawParams{}
	start := graphql.Now()
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	if err := r.BindJSON(&params); err != nil {
		gqlErr := gqlerror.Errorf("could not get json from request body: %+v", err)
		resp := exec.DispatchError(c, gqlerror.List{gqlErr})
		r.JSON(consts.StatusBadRequest, resp)
		return
	}

	rc, opErr := exec.CreateOperationContext(c, params)
	if opErr != nil {
		resp := exec.DispatchError(c, opErr)
		r.JSON(consts.StatusUnprocessableEntity, resp)
		return
	}

	respH, respCtx := exec.DispatchOperation(c, rc)
	r.JSON(consts.StatusOK, respH(respCtx))
}
