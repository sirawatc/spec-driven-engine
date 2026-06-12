# Architecture — Service Engine

**Version:** 0.2  
**Date:** 2026-06-13  
**Architect:** Winston  
**Status:** Final (all open questions resolved)

---

## 1. System Overview

The engine is a single Go binary that:
1. Loads a YAML spec on startup
2. Starts an HTTP server with routes defined in the spec
3. For each request: validates → transforms → calls backend via TCP → maps response codes → returns JSON
4. Serves a read-only dashboard UI derived from the same spec

```
┌─────────────────────────────────────────────────────────────┐
│                   Service Engine (Go)                  │
│                                                             │
│  ┌────────────┐    ┌─────────────────────────────────────┐  │
│  │            │    │            HTTP Server               │  │
│  │   Spec     │───▶│  /dashboard/**   → Dashboard Handler │  │
│  │   Loader   │    │  /<spec routes>  → Request Handler   │  │
│  │            │    └────────────────────┬────────────────┘  │
│  └────────────┘                         │                   │
│        │                    ┌───────────▼──────────────┐    │
│        │                    │      Request Pipeline     │    │
│        │                    │  1. Validate request      │    │
│        │                    │  2. Resolve systemCode    │    │
│        │                    │  3. Transform → TCP msg   │    │
│        │                    │  4. backend Client.Send()     │    │
│        │                    │  5. Map response code     │    │
│        │                    │  6. Transform → HTTP resp │    │
│        │                    └───────────┬──────────────┘    │
│        └────────────────────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
                                          │ TCP
                         ┌────────────────▼────────────────┐
                         │  backend (Mock in PoC / Real later)  │
                         │  Port per systemCode             │
                         └─────────────────────────────────┘
```

---

## 2. Component Breakdown

### 2.1 Spec Loader (`internal/spec`)

Responsibilities:
- Parse and validate YAML spec at startup
- Build typed in-memory `Spec` struct (routes, systems, transforms, code map)
- Log spec `name`, `version`, and load timestamp (audit trail)
- Return a structured error if spec is invalid — engine must not start with a bad spec

```
spec.Load(path string) (*Spec, error)
```

The `Spec` struct is the single source of truth passed to all other components. No component re-reads the file.

### 2.2 HTTP Server (`internal/server`)

- Standard library `net/http` + `chi` router (lightweight, no magic, idiomatic Go)
- Routes registered dynamically from `spec.Endpoints` at startup — not at request time
- Middleware stack (applied globally):
  - Request ID injection
  - Structured request/response logging
  - Panic recovery → `40000` response
- Dashboard routes mounted at `/dashboard`
- All other routes delegate to the Request Handler

**Trade-off on router choice:** `chi` vs stdlib mux vs `gin`/`echo`. Chi gives clean middleware chaining and path params without pulling in a full framework. Stdlib mux works but lacks middleware composability. Gin/Echo are overkill for this use case and add unnecessary opinions.

### 2.3 Request Handler (`internal/handler`)

The pipeline per request, in order:

```
1. Parse request body (JSON)
      ↓ error → 31000 (input parse error)
2. Validate against spec schema (required fields, types)
      ↓ error → 32000 (input validation error)
3. Extract systemCode → look up backend port in spec.Systems
      ↓ not found → 21002 (route name not found)
4. Transformer.ToTCP(request, endpoint.RequestTransform)
      ↓ error → 40000
5. CBSClient.Send(port, tcpMessage)
      ↓ error → 40000
6. CodeMapper.Map(cbsCode) → engineCode
7. Transformer.ToHTTP(cbsResponse, endpoint.ResponseTransform)
8. Write JSON response with engineCode + mapped body
```

Each step is a discrete function — easy to test in isolation.

### 2.4 Transformer (`internal/transformer`)

Maps fields between HTTP JSON and TCP fixed-length messages using rules defined in the spec.

Template syntax: `{{request.body.<field>}}` for request transforms, `{{cbs.<field>}}` for response transforms.

```go
type Transformer interface {
    ToTCP(req Request, rules []FieldRule) (CBSMessage, error)
    ToHTTP(cbs BackendResponse, rules []FieldRule) (map[string]any, error)
}
```

**Fixed-length serialization:** The TCP message is a byte sequence where each field occupies an exact byte range. The spec defines the ordered list of fields with name, length, value/template, alignment, and pad character. The transformer serializes request fields in spec order and parses response fields by cumulative offset.

```
Request message (bytes):
[MSG_TYPE: 4][RESOURCE_ID: 16][...padding/more fields...]
 0200         1234567890______

Response message (bytes):
[ResponseCode: 8][FIELD_A: 15][FIELD_B: 3][...]
 AA______                 000000012345.67THB
```

**Response code extraction:** The field named `ResponseCode` in the backend response is always extracted first and passed to the CodeMapper before the rest of the response is transformed.

### 2.5 backend Client (`internal/cbs`)

Interface-driven to enable the mock swap:

```go
type Client interface {
    Send(port int, msg CBSMessage) (BackendResponse, error)
}
```

Two implementations:

| Implementation | Used when |
|----------------|-----------|
| `TCPClient` | Real backend (post-PoC) |
| `MockClient` | PoC — returns configured responses per input |

`MockClient` is configured from a separate YAML fixtures file (not the main spec). This keeps the spec clean and lets QA control mock behavior independently.

```yaml
# mock-fixtures.yaml
- match:
    systemCode: "SERVICE_B"
    MSG_TYPE: "0200"
  response:
    RC: "OK"
    FIELD_A: "12345.67"
    FIELD_B: "THB"
- match:
    systemCode: "SERVICE_B"
    MSG_TYPE: "0200"
    RESOURCE_ID: "ERROR_ACCOUNT"
  response:
    RC: "BACKEND_ERR_001"
```

### 2.6 Response Code Mapper (`internal/codemap`)

Loads the standard 5-digit code table. Per-spec overrides merge on top.

```go
type Mapper struct {
    table map[string]EngineCode  // cbsCode → EngineCode
}

type EngineCode struct {
    Code       int    // e.g. 21001
    HTTPStatus int    // e.g. 200
    Message    string
}
```

Lookup logic:
1. Check spec-level `response_codes` overrides
2. Fall back to standard table
3. If still not found → `20000` (default business error, HTTP 200)

### 2.7 Dashboard Handler (`internal/dashboard`)

- Serves embedded static assets via Go `embed` — single binary, no file system dependency
- Reads from the in-memory `*Spec` (same pointer passed at startup)
- Routes:
  - `GET /dashboard` — overview: name, version, domain
  - `GET /dashboard/endpoints` — list of all endpoints
  - `GET /dashboard/endpoints/{path}` — endpoint detail: schema, transforms, response codes
  - `GET /dashboard/systems` — systemCode registry with ports
  - `GET /dashboard/codes` — full response code table

HTML rendered server-side with Go `html/template`. No JS framework needed for a read-only reference UI.

---

## 3. Spec Format Design

Fixed-length field rules use: `name`, `length`, `value` (literal or template), `align` (`left`/`right`), and `pad` character. Alignment defaults to `left`, pad defaults to space `" "` for strings and `"0"` for numbers. Fields are serialized in declaration order.

```yaml
version: "1.0.0"
name: "Service Engine - Example Domain"
description: "Handles payment and inquiry operations against the backend"

# systemCode (always at request body root) → TCP port
systems:
  SERVICE_B:
    port: 9001
    description: "backend service B port"
  SERVICE_A:
    port: 9002
    description: "backend service A port"

# Optional: extend or override the standard response code table
response_codes:
  22010:
    type: business_error
    http_status: 200
    description: "Insufficient balance"
    backend_code: "SVC_ERR_002"

endpoints:
  - path: /example/query
    method: POST
    summary: "Inquire resource value"
    description: "Returns current value for the given resource identifier"

    request:
      fields:
        systemCode:
          type: string
          required: true
          description: "Routes request to backend port. Always required."
        resourceId:
          type: string
          required: true
          description: "Resource identifier to inquire"

    transform:
      # Request: ordered fixed-length fields written to TCP message
      request:
        - name: MSG_TYPE
          length: 4
          value: "0200"           # literal constant
          align: left
          pad: " "
        - name: RESOURCE_ID
          length: 16
          value: "{{request.body.resourceId}}"
          align: left
          pad: " "

      # Response: ordered fixed-length fields read from TCP response
      # ResponseCode is always extracted first for code mapping
      response:
        fields:
          - name: ResponseCode
            length: 8
          - name: FIELD_A
            length: 15
          - name: FIELD_B
            length: 3
        # Mapping: backend parsed fields → HTTP response body keys
        mapping:
          balance:  "{{cbs.FIELD_A}}"
          currency: "{{cbs.FIELD_B}}"

    response_codes:
      # Endpoint-specific codes merged with global table
      - code: 22001
        description: "Account not found"
```

---

## 4. Project Structure

```
engine-poc/
├── cmd/
│   └── engine/
│       └── main.go              # Entry point: load spec, wire components, start server
├── internal/
│   ├── spec/
│   │   ├── loader.go            # spec.Load() — parse + validate YAML
│   │   └── model.go             # Spec, Endpoint, Transform, SystemDef types
│   ├── server/
│   │   └── server.go            # HTTP server setup, route registration, middleware
│   ├── handler/
│   │   └── handler.go           # Request pipeline (validate → transform → backend → respond)
│   ├── transformer/
│   │   ├── transformer.go       # Transformer interface + template engine
│   │   └── transformer_test.go
│   ├── cbs/
│   │   ├── client.go            # Client interface + CBSMessage/BackendResponse types
│   │   ├── tcp.go               # TCPClient implementation
│   │   └── mock.go              # MockClient implementation + fixture loader
│   ├── codemap/
│   │   ├── mapper.go            # CodeMapper: backend code → EngineCode
│   │   ├── standard.go          # Built-in standard code table
│   │   └── mapper_test.go
│   └── dashboard/
│       ├── handler.go           # Dashboard HTTP handlers
│       └── templates/           # Embedded HTML templates
│           ├── layout.html
│           ├── overview.html
│           ├── endpoints.html
│           └── codes.html
├── spec/
│   └── example.yaml             # Example spec for PoC endpoint
├── mock/
│   └── fixtures.yaml            # Mock backend response fixtures
├── go.mod
└── go.sum
```

---

## 5. Architectural Decision Records

### ADR-001: chi router over stdlib mux
**Decision:** Use `github.com/go-chi/chi` for HTTP routing.  
**Reason:** Clean middleware composition without a full framework. Path parameters and sub-router support. Widely used in Go backends. Single dependency, no transitive bloat.  
**Trade-off:** One external dependency vs zero. Acceptable given chi's stability and narrow scope.

### ADR-002: Interface-based backend client
**Decision:** `cbs.Client` is an interface with `TCPClient` and `MockClient` implementations.  
**Reason:** Enables PoC to run without real backend. Swap is a single line at wire-up time in `main.go`. No test doubles needed in handler tests — inject `MockClient` directly.  
**Trade-off:** Slightly more indirection. Worth it for testability and the explicit PoC→production upgrade path.

### ADR-003: Dashboard rendered server-side with `html/template` + `embed`
**Decision:** No frontend framework. Go templates + embedded assets. Dashboard served by the engine binary itself.  
**Reason:** Single binary deployment is a hard constraint for a regulated environment. No build step, no CDN dependency, no JS runtime. The dashboard is read-only reference UI — no reactivity needed.  
**Trade-off:** Less interactive than an SPA. Acceptable given the use case is documentation browsing, not data entry.

### ADR-004: TCP wire format — Fixed-length fields
**Decision:** backend messages use fixed-length fields, serialized in declaration order.  
**Reason:** Confirmed by Sirawat. Each field occupies an exact byte range; no delimiters. Standard format for legacy backend systems.  
**Impact on spec:** Transform `request` block is an ordered list of `{name, length, value, align, pad}` rules. Transform `response.fields` block is an ordered list of `{name, length}` for parsing by cumulative offset.  
**Impact on Transformer:** `ToTCP` concatenates fields in order, padding/truncating to exact length. `ToHTTP` parses backend response by walking cumulative byte offsets.  
**For PoC:** MockClient receives and returns `map[string]string` (field name → string value) — the serialization layer is exercised in `TCPClient` tests, not in the mock path.

### ADR-005: Mock fixtures in separate file, not in spec
**Decision:** Mock backend responses live in `mock/fixtures.yaml`, separate from the engine spec.  
**Reason:** Keeps the spec clean and environment-agnostic. The spec describes production behavior; mock fixtures are a test concern. QA can add/modify mock responses without touching the spec.  
**Trade-off:** Two files to maintain. Worth it for separation of concerns.

---

## 6. Data Flow — PoC Endpoint

```
POST /example/query
{
  "systemCode": "SERVICE_A",
  "resourceId": "1234567890"
}

1. Handler parses body → valid JSON
2. Validator: systemCode ✓ root field present, resourceId ✓ present
3. SystemCode lookup: SERVICE_A → port 9002 ✓
4. Transformer.ToTCP (fixed-length serialization):
      Field[0]: MSG_TYPE    len=4  value="0200"         → "0200"
      Field[1]: RESOURCE_ID  len=16 value="1234567890"   → "1234567890      "
      Wire bytes: "02001234567890      "
5. MockClient.Send(9002, {MSG_TYPE:"0200", RESOURCE_ID:"1234567890"})
   → fixture match → {ResponseCode:"AA      ", FIELD_A:"000000012345.67", FIELD_B:"THB"}
6. CodeMapper.Map("OK") → {Code: 10000, HTTPStatus: 200, Message: "Success"}
7. Transformer.ToHTTP (parse response fields by offset, apply mapping):
      bytes[0:8]   → ResponseCode = "OK"       (trimmed)
      bytes[8:23]  → FIELD_A             = "12345.67" (trimmed)
      bytes[23:26] → FIELD_B             = "THB"      (trimmed)
      mapping: balance = FIELD_A, currency = FIELD_B
8. Response:

HTTP 200
{
  "code": 10000,
  "message": "Success",
  "data": {
    "balance": "12345.67",
    "currency": "THB"
  }
}
```

---

## 7. Resolved Decisions

| # | Question | Answer |
|---|----------|--------|
| Q1 | TCP wire format? | Fixed-length fields, ordered, no delimiters |
| Q2 | backend response code field name? | `ResponseCode` |
| Q3 | `systemCode` location in request? | Always root of request body |
| Q4 | Dashboard shows transformation rules? | Yes — dashboard shows everything in the spec |

No open questions remain. Architecture is complete.
