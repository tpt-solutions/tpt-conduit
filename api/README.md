# /api — API layer (Phase 2)

Implements the API surface described in `spec.txt` and `TODO.md`.

- **GraphQL** (`api/graphql`, `api/server.go`): queries for `tickets`,
  `ticket(id)`, `workflows`, `workflow(name, version)`, `runs`, `run(id)` and
  the event `timeline`; mutations for `createTicket`, `approve`, `reject`,
  `cancel`. Built on `github.com/graphql-go/graphql` and wraps the engine
  directly. See `api/server_test.go` for schema-conformance and auth tests.
- **gRPC** (`api/proto`, `api/grpc`): protobuf service definitions mirroring the
  core mutations and read APIs for high-performance, service-to-service use.
  Generated with `protoc` (see `api/proto/conduit.proto`); the Go bindings and a
  server wrapping the engine live in `api/grpc`.
- **Auth** (`api/auth.go`): single-tenant auth middleware accepting either Basic
  `user:pass` or an `X-API-Key` (also `Authorization: Bearer <key>`). All routes
  are gated; unauthenticated requests receive `401`.

Run the server with an in-memory engine and basic auth, e.g.:

```go
e := engine.NewEngine(engine.NewInMemoryEventLog(), engine.NewInMemoryStore(), 4)
srv, _ := api.NewServer(e, api.AuthConfig{Username: "admin", Password: "secret", APIKeys: []string{"key-1"}})
log.Fatal(srv.Serve(":8080"))
```
