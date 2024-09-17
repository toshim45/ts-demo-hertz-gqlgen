package graph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	// "github.com/99designs/gqlgen/graphql/handler/transport"
)

type (
	Server struct {
		transports []graphql.Transport
		exec       *executor.Executor
	}
)

func New(es graphql.ExecutableSchema) *Server {
	return &Server{
		exec: executor.New(es),
	}
}

func NewHertzHandler(es graphql.ExecutableSchema) *Server {
	srv := New(es)

	// srv.AddTransport(HertzPOST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})
	return srv
}

func (s *Server) AddTransport(transport graphql.Transport) {
	s.transports = append(s.transports, transport)
}

func (s *Server) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.exec.SetErrorPresenter(f)
}

func (s *Server) SetRecoverFunc(f graphql.RecoverFunc) {
	s.exec.SetRecoverFunc(f)
}

func (s *Server) SetQueryCache(cache graphql.Cache[*ast.QueryDocument]) {
	s.exec.SetQueryCache(cache)
}

func (s *Server) SetParserTokenLimit(limit int) {
	s.exec.SetParserTokenLimit(limit)
}

func (s *Server) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundFields(f graphql.FieldMiddleware) {
	s.exec.AroundFields(f)
}

// AroundRootFields is a convenience method for creating an extension that only implements field middleware
func (s *Server) AroundRootFields(f graphql.RootFieldMiddleware) {
	s.exec.AroundRootFields(f)
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Server) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Server) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func (s *Server) getTransport(r *http.Request) graphql.Transport {
	for _, t := range s.transports {
		if t.Supports(r) {
			return t
		}
	}
	return nil
}

func (s *Server) ServeHertzHTTP(c context.Context, r *app.RequestContext) {
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(c, err)
			gqlErr, _ := err.(*gqlerror.Error)
			resp := &graphql.Response{Errors: []*gqlerror.Error{gqlErr}}
			r.JSON(consts.StatusUnprocessableEntity, resp)
		}
	}()

	c = graphql.StartOperationTrace(c)

	t := HertzPOST{}
	t.Do(c, r, s.exec)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(r.Context(), err)
			gqlErr, _ := err.(*gqlerror.Error)
			resp := &graphql.Response{Errors: []*gqlerror.Error{gqlErr}}
			b, _ := json.Marshal(resp)
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write(b)
		}
	}()

	r = r.WithContext(graphql.StartOperationTrace(r.Context()))

	transport := s.getTransport(r)
	if transport == nil {
		sendErrorf(w, http.StatusBadRequest, "transport not supported")
		return
	}

	transport.Do(w, r, s.exec)
}

func sendError(w http.ResponseWriter, code int, errors ...*gqlerror.Error) {
	w.WriteHeader(code)
	b, err := json.Marshal(&graphql.Response{Errors: errors})
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(b)
}

func sendErrorf(w http.ResponseWriter, code int, format string, args ...any) {
	sendError(w, code, &gqlerror.Error{Message: fmt.Sprintf(format, args...)})
}

type OperationFunc func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler

func (r OperationFunc) ExtensionName() string {
	return "InlineOperationFunc"
}

func (r OperationFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return errors.New("OperationFunc can not be nil")
	}
	return nil
}

func (r OperationFunc) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	return r(ctx, next)
}

type ResponseFunc func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response

func (r ResponseFunc) ExtensionName() string {
	return "InlineResponseFunc"
}

func (r ResponseFunc) Validate(schema graphql.ExecutableSchema) error {
	if r == nil {
		return errors.New("ResponseFunc can not be nil")
	}
	return nil
}

func (r ResponseFunc) InterceptResponse(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	return r(ctx, next)
}

type FieldFunc func(ctx context.Context, next graphql.Resolver) (res any, err error)

func (f FieldFunc) ExtensionName() string {
	return "InlineFieldFunc"
}

func (f FieldFunc) Validate(schema graphql.ExecutableSchema) error {
	if f == nil {
		return errors.New("FieldFunc can not be nil")
	}
	return nil
}

func (f FieldFunc) InterceptField(ctx context.Context, next graphql.Resolver) (res any, err error) {
	return f(ctx, next)
}

type DummyResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type HertzPOST struct{}

func (h HertzPOST) Do(c context.Context, r *app.RequestContext, exec graphql.GraphExecutor) {
	params := &graphql.RawParams{}
	start := graphql.Now()
	// params.Headers = r.Request.Header
	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	if err := r.BindJSON(&params); err != nil {
		gqlErr := gqlerror.Errorf("could not get json from request body: %+v", err)
		resp := exec.DispatchError(c, gqlerror.List{gqlErr})
		r.JSON(consts.StatusBadRequest, resp)
	}

	r.JSON(consts.StatusOK, DummyResponse{ID: "id-1", Name: "name-1"})
}
