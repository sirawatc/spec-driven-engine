# Story: E1-S1 — Project Scaffold

**Epic:** E1 — Engine Foundation  
**Status:** Ready for Development  
**Depends on:** nothing (first story)  
**Blocks:** E1-S2, E1-S3, E2-S1, E3-S2, E3-S3, E4-S1

---

## Summary

Set up the Go project structure for the Service Engine. This story produces a compilable binary with the correct package layout. No business logic — just the skeleton that all subsequent stories will fill.

---

## Acceptance Criteria

- [ ] `go.mod` created with module name `engine-poc`
- [ ] Directory structure matches architecture doc (see below)
- [ ] `cmd/engine/main.go` compiles and exits cleanly with a startup log line
- [ ] `chi` added as the only external dependency
- [ ] `go build ./...` passes with zero errors

---

## Directory Structure to Create

```
engine-poc/
├── cmd/
│   └── engine/
│       └── main.go
├── internal/
│   ├── spec/
│   │   ├── loader.go
│   │   └── model.go
│   ├── server/
│   │   └── server.go
│   ├── handler/
│   │   └── handler.go
│   ├── transformer/
│   │   └── transformer.go
│   ├── cbs/
│   │   ├── client.go
│   │   ├── tcp.go
│   │   └── mock.go
│   ├── codemap/
│   │   ├── mapper.go
│   │   └── standard.go
│   └── dashboard/
│       ├── handler.go
│       └── templates/
│           ├── layout.html
│           ├── overview.html
│           ├── endpoints.html
│           └── codes.html
├── spec/
│   └── v1/                  # versioned spec directory (populated in E7-S1)
├── mock/                    # mock fixtures directory (populated in E7-S1)
├── go.mod
└── go.sum
```

---

## Implementation Notes

### `go.mod`
```
module engine-poc

go 1.22
```

### `cmd/engine/main.go`
Minimal skeleton — just prints a startup message and exits. No wiring yet (that is E5-S2). Subsequent stories will expand this.

```go
package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Fprintln(os.Stdout, "Service Engine — starting")
}
```

### Internal package stubs
Each file under `internal/` must declare its package and contain at least one exported placeholder so `go build ./...` compiles without errors. Use the patterns below:

**`internal/spec/model.go`** — declare the top-level types (empty structs are fine):
```go
package spec

type Spec struct{}
type Endpoint struct{}
type SystemDef struct{}
type FieldDef struct{}
type FieldRule struct{}
type TransformRules struct{}
type ResponseCodeDef struct{}
```

**`internal/spec/loader.go`** — stub the public function:
```go
package spec

func Load(path string) (*Spec, error) {
    panic("not implemented")
}
```

**`internal/cbs/client.go`** — declare the interface:
```go
package cbs

type CBSMessage map[string]string
type BackendResponse []byte

type Client interface {
    Send(port int, msg CBSMessage) (BackendResponse, error)
}
```

All other packages (`server`, `handler`, `transformer`, `codemap`, `dashboard`) — minimal `package <name>` file with one unexported placeholder comment is sufficient. They must compile but need no exported symbols yet.

### Dashboard templates
Create empty HTML files (valid HTML shell is fine):
```html
<!-- layout.html -->
<!DOCTYPE html><html><body>{{template "content" .}}</body></html>
```
Other template files can be single-line placeholders.

### Adding `chi`
```bash
go get github.com/go-chi/chi/v5
```
Run this after creating `go.mod`. `go.sum` will be generated automatically.

---

## Files to Create

| File | Content |
|------|---------|
| `go.mod` | Module declaration |
| `cmd/engine/main.go` | Startup log + exit |
| `internal/spec/model.go` | Type stubs |
| `internal/spec/loader.go` | `Load()` stub |
| `internal/server/server.go` | Package stub |
| `internal/handler/handler.go` | Package stub |
| `internal/transformer/transformer.go` | Package stub |
| `internal/cbs/client.go` | `Client` interface + type declarations |
| `internal/cbs/tcp.go` | Package stub |
| `internal/cbs/mock.go` | Package stub |
| `internal/codemap/mapper.go` | Package stub |
| `internal/codemap/standard.go` | Package stub |
| `internal/dashboard/handler.go` | Package stub |
| `internal/dashboard/templates/layout.html` | HTML shell |
| `internal/dashboard/templates/overview.html` | HTML placeholder |
| `internal/dashboard/templates/endpoints.html` | HTML placeholder |
| `internal/dashboard/templates/codes.html` | HTML placeholder |

---

## Verification

Run after implementation:

```bash
go build ./...          # must pass with zero errors
go vet ./...            # must pass with zero warnings
./engine               # (after go build) prints startup message and exits 0
```

---

## Notes for Next Story

- E1-S2 will flesh out `internal/spec/model.go` and `internal/spec/loader.go`
- The `cbs.Client` interface declared here in `client.go` is the contract E3-S2 implements
- Do not add any logic beyond what the AC requires — stubs only
