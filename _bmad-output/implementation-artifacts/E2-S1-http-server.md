# Story: E2-S1 — HTTP Server with Dynamic Route Registration

**Epic:** E2 — HTTP Runtime  
**Status:** Ready for Development  
**Depends on:** E1-S3  
**Blocks:** E5-S2

---

## Summary

Start a chi HTTP server and register versioned routes from the `SpecRegistry` at startup. No request logic yet — routes just need to exist and route to a handler placeholder. Middleware stack goes in here.

---

## Acceptance Criteria

- [ ] Server starts on port from `ENGINE_PORT` env var (default `8080`)
- [ ] Routes registered from all versions in `SpecRegistry` with version prefix: `/<version>/<path>`
- [ ] `chi` middleware: request ID injection, structured request log (method, path, status, latency), panic recovery
- [ ] `GET /health` → `200 OK`, body `{"status":"ok"}`
- [ ] Server startup log includes bound address and all registered version prefixes
- [ ] Path not in any spec → `404`
- [ ] Two versions with same logical path are independent routes

---

## Implementation (`internal/server/server.go`)

```go
package server

import (
    "fmt"
    "net/http"
    "os"
    "sort"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"

    "engine-poc/internal/spec"
)

type Server struct {
    router *chi.Mux
    port   string
}

// HandlerFactory produces an http.HandlerFunc for a given spec version and endpoint.
// Injected so server package stays decoupled from handler package.
type HandlerFactory func(version string, endpoint spec.Endpoint) http.HandlerFunc

func New(registry spec.SpecRegistry, factory HandlerFactory) *Server {
    r := chi.NewRouter()

    // Middleware
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)

    // Health check — always available
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })

    // Register versioned routes from registry
    versions := sortedVersions(registry)
    for _, version := range versions {
        s := registry[version]
        v := version // capture for closure
        r.Route("/"+v, func(r chi.Router) {
            for _, ep := range s.Endpoints {
                method := ep.Method
                path := ep.Path
                handler := factory(v, ep)
                r.Method(method, path, handler)
            }
        })
    }

    port := os.Getenv("ENGINE_PORT")
    if port == "" {
        port = "8080"
    }

    return &Server{router: r, port: port}
}

func (s *Server) Start() error {
    addr := ":" + s.port
    fmt.Printf("server listening on %s\n", addr)
    return http.ListenAndServe(addr, s.router)
}

func sortedVersions(registry spec.SpecRegistry) []string {
    versions := make([]string, 0, len(registry))
    for v := range registry {
        versions = append(versions, v)
    }
    sort.Strings(versions)
    return versions
}
```

---

## Handler Placeholder (used until E5-S1)

In `internal/handler/handler.go`, add a placeholder factory for wiring:

```go
package handler

import (
    "net/http"
    "engine-poc/internal/spec"
)

// PlaceholderFactory returns a stub handler for use before E5-S1 is implemented.
func PlaceholderFactory(version string, endpoint spec.Endpoint) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusNotImplemented)
        w.Write([]byte(`{"code":40000,"message":"not implemented","data":null}`))
    }
}
```

---

## Startup Log Format

On start, log each registered version and its endpoint count:

```
server listening on :8080
registered version v1: 1 endpoint(s)
registered version v2: 1 endpoint(s)
```

---

## Verification

```bash
go build ./...
# start with example spec
ENGINE_SPEC_DIR=spec go run ./cmd/engine

# in another terminal:
curl localhost:8080/health                        # → {"status":"ok"}
curl -X POST localhost:8080/v1/example/query    # → {"code":40000,...} (placeholder)
curl localhost:8080/not-a-path                    # → 404
```
