# Story: E5-S2 — Startup Wiring in main.go

**Epic:** E5 — Request Handler Pipeline  
**Status:** Ready for Development  
**Depends on:** E5-S1, E2-S1  
**Blocks:** E7-S1

---

## Summary

Wire all components together in `main.go` so `go run ./cmd/engine` produces a running engine. Uses `ENGINE_SPEC_DIR` to load the spec registry, constructs all components, and starts the HTTP server. The mock fixtures path is also configurable.

---

## Acceptance Criteria

- [ ] Loads spec registry from `ENGINE_SPEC_DIR` (default: `spec/`)
- [ ] Loads mock fixtures from `ENGINE_MOCK_FIXTURES` (default: `mock/fixtures.yaml`)
- [ ] Constructs: `SpecRegistry → Mapper (with overrides) → MockClient → Handler → Server`
- [ ] If spec load fails → exit code `1`, log the error
- [ ] If fixture load fails → exit code `1`, log the error
- [ ] On successful start: logs spec name, all versions, bound address
- [ ] `curl -X POST localhost:8080/v1/example/query -H 'Content-Type: application/json' -d '{"systemCode":"SERVICE_A","resourceId":"1234567890"}'` returns `{"code":10000,...}`

---

## Implementation (`cmd/engine/main.go`)

```go
package main

import (
    "log"
    "os"

    "engine-poc/internal/cbs"
    "engine-poc/internal/codemap"
    "engine-poc/internal/handler"
    "engine-poc/internal/server"
    "engine-poc/internal/spec"
)

func main() {
    // Load spec registry
    specDir := spec.SpecDirFromEnv()
    registry, err := spec.LoadRegistry(specDir)
    if err != nil {
        log.Printf("failed to load spec registry: %v", err)
        os.Exit(1)
    }

    // Load mock fixtures
    fixturePath := os.Getenv("ENGINE_MOCK_FIXTURES")
    if fixturePath == "" {
        fixturePath = "mock/fixtures.yaml"
    }

    // Build one handler per spec version
    // Server needs a HandlerFactory that knows which spec to use per version
    handlerMap := make(map[string]*handler.Handler, len(registry))
    for version, s := range registry {
        // Build mapper with spec-level overrides
        mapper := codemap.NewWithOverrides(s.ResponseCodes)

        // Build mock client with response field lengths from this spec's endpoints
        // For PoC: use first endpoint's response fields (single-endpoint spec)
        var respFields []spec.ResponseField
        if len(s.Endpoints) > 0 {
            respFields = s.Endpoints[0].Transform.Response.Fields
        }

        mockClient, err := cbs.LoadMockClient(fixturePath, respFields)
        if err != nil {
            log.Printf("failed to load mock fixtures from %s: %v", fixturePath, err)
            os.Exit(1)
        }

        handlerMap[version] = handler.New(s, mockClient, mapper)
    }

    // HandlerFactory routes each versioned request to the correct handler
    factory := func(version string, endpoint spec.Endpoint) func(http.ResponseWriter, *http.Request) {
        h := handlerMap[version]
        return h.Factory()(version, endpoint)
    }

    srv := server.New(registry, factory)
    if err := srv.Start(); err != nil {
        log.Printf("server error: %v", err)
        os.Exit(1)
    }
}
```

> **Note:** `http.ResponseWriter` and `*http.Request` imports are needed — add `"net/http"` to imports.

---

## Environment Variables Summary

| Var | Default | Purpose |
|-----|---------|---------|
| `ENGINE_SPEC_DIR` | `spec/` | Directory containing versioned spec subdirectories |
| `ENGINE_MOCK_FIXTURES` | `mock/fixtures.yaml` | Mock backend fixture file |
| `ENGINE_PORT` | `8080` | HTTP listen port |

---

## Manual Verification

```bash
go run ./cmd/engine

# In another terminal:
curl localhost:8080/health
# → {"status":"ok"}

curl -X POST localhost:8080/v1/example/query \
  -H 'Content-Type: application/json' \
  -d '{"systemCode":"SERVICE_A","resourceId":"1234567890"}'
# → {"code":10000,"message":"Success","data":{"balance":"12345.67","currency":"THB"}}

curl -X POST localhost:8080/v1/example/query \
  -H 'Content-Type: application/json' \
  -d '{"systemCode":"SERVICE_A","resourceId":"ERROR_ACCOUNT"}'
# → {"code":21001,"message":"Backend internal error","data":null}
```

---

## Verification

```bash
go build ./...
go run ./cmd/engine   # engine starts, logs versions and address
```
