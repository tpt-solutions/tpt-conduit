// Package api wires the durable workflow engine behind an authenticated
// GraphQL HTTP endpoint, mirroring the full mutation surface exposed by the
// engine.
package api

import (
	"encoding/json"
	"io"
	"net/http"

	gql "github.com/graphql-go/graphql"

	apigql "tptconduit/api/graphql"
	"tptconduit/engine"
)

// Server is an HTTP server exposing the conduit GraphQL API.
type Server struct {
	engine *engine.Engine
	schema gql.Schema
	auth   AuthConfig
	mux    *http.ServeMux
}

// NewServer builds an API server bound to the given engine and auth config.
func NewServer(e *engine.Engine, auth AuthConfig) (*Server, error) {
	schema, err := apigql.NewSchema(e)
	if err != nil {
		return nil, err
	}
	s := &Server{engine: e, schema: schema, auth: auth, mux: http.NewServeMux()}
	s.routes()
	return s, nil
}

// Handler returns the http.Handler (auth-wrapped) for embedding in other apps.
func (s *Server) Handler() http.Handler {
	return s.auth.Middleware(s.mux)
}

// Serve starts the HTTP server on the given address.
func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}

func (s *Server) routes() {
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("/graphql", s.handleGraphQL)
}

type graphqlRequest struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
	OperationName string         `json:"operationName"`
}

func (s *Server) handleGraphQL(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST is supported", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	var req graphqlRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Query == "" {
		if q := r.URL.Query().Get("query"); q != "" {
			req.Query = q
		}
	}
	result := gql.Do(gql.Params{
		Schema:         s.schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		OperationName:  req.OperationName,
		Context:        r.Context(),
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
