# Story: E5-S1 — Handler Pipeline Wiring

**Epic:** E5 — Request Handler Pipeline  
**Status:** Ready for Development  
**Depends on:** E2-S2, E2-S3, E3-S1, E3-S2, E4-S1, E4-S2  
**Blocks:** E5-S2

---

## Summary

Wire all components into the complete request processing pipeline: validate → resolve systemCode → transform to TCP → call backend → extract response code → transform to HTTP → respond. This is the central piece of the engine. Replace the placeholder factory from E2-S1.

---

## Acceptance Criteria

- [ ] Handler constructed with: `*spec.Spec`, `cbs.Client`, `*codemap.Mapper`
- [ ] Pipeline executes in order; any step failure short-circuits to error response
- [ ] Success response shape: `{"code": 10000, "message": "Success", "data": {...}}`
- [ ] Error response shape: `{"code": 32000, "message": "...", "data": null}`
- [ ] Business error (backend non-success code) → HTTP 200 with appropriate engine code
- [ ] Internal error (backend client error, transformer error) → HTTP 500, code `40000`
- [ ] Integration tests via `MockClient`: happy path, missing field, unknown systemCode, backend business error

---

## Handler (`internal/handler/handler.go`)

```go
package handler

import (
    "encoding/json"
    "net/http"

    "engine-poc/internal/cbs"
    "engine-poc/internal/codemap"
    "engine-poc/internal/spec"
    "engine-poc/internal/transformer"
)

type Handler struct {
    spec   *spec.Spec
    client cbs.Client
    mapper *codemap.Mapper
}

func New(s *spec.Spec, client cbs.Client, mapper *codemap.Mapper) *Handler {
    return &Handler{spec: s, client: client, mapper: mapper}
}

// Factory returns a HandlerFactory compatible with server.New().
func (h *Handler) Factory() func(version string, endpoint spec.Endpoint) http.HandlerFunc {
    return func(version string, endpoint spec.Endpoint) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            h.handle(w, r, endpoint)
        }
    }
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request, endpoint spec.Endpoint) {
    // Step 1: Parse and validate request body
    body, engineErr := ParseAndValidate(r, endpoint.Request)
    if engineErr != nil {
        writeError(w, engineErr)
        return
    }

    // Step 2: Resolve systemCode → backend port
    systemCode, _ := body["systemCode"].(string)
    port, engineErr := ResolvePort(systemCode, h.spec.Systems)
    if engineErr != nil {
        writeError(w, engineErr)
        return
    }

    // Step 3: Transform HTTP body → backend message + wire bytes
    cbsMsg, _, err := transformer.ToTCP(body, endpoint.Transform.Request)
    if err != nil {
        writeError(w, ErrInternal)
        return
    }

    // Step 4: Call backend service
    rawResp, err := h.client.Send(port, cbsMsg)
    if err != nil {
        writeError(w, ErrInternal)
        return
    }

    // Step 5: Parse response fields and extract ResponseCode
    parsed, err := transformer.ParseResponseFields(rawResp, endpoint.Transform.Response.Fields)
    if err != nil {
        writeError(w, ErrInternal)
        return
    }
    engineCode := h.mapper.Map(parsed["ResponseCode"])

    // Step 6: Transform backend response → HTTP body map
    httpData, err := transformer.ToHTTP(rawResp, endpoint.Transform.Response)
    if err != nil {
        writeError(w, ErrInternal)
        return
    }

    writeSuccess(w, engineCode, httpData)
}
```

---

## Response Writers (`internal/handler/response.go`)

```go
package handler

import (
    "encoding/json"
    "net/http"

    "engine-poc/internal/codemap"
)

type apiResponse struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    any    `json:"data"`
}

func writeSuccess(w http.ResponseWriter, code codemap.EngineCode, data map[string]any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code.HTTPStatus)
    json.NewEncoder(w).Encode(apiResponse{
        Code:    code.Code,
        Message: code.Message,
        Data:    data,
    })
}

func writeError(w http.ResponseWriter, err *EngineError) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(err.HTTPStatus)
    json.NewEncoder(w).Encode(apiResponse{
        Code:    err.Code,
        Message: err.Message,
        Data:    nil,
    })
}
```

---

## Integration Tests (`internal/handler/handler_test.go`)

Use `httptest.NewRecorder()` and inject a `MockClient` directly. Load the placeholder spec from E1-S2.

| Test | Request | Expected code | Expected HTTP status |
|------|---------|---------------|---------------------|
| Happy path | valid body, known systemCode | 10000 | 200 |
| Invalid JSON | `not-json` | 31000 | 400 |
| Missing resourceId | `{"systemCode":"SERVICE_A"}` | 32000 | 400 |
| Unknown systemCode | `{"systemCode":"NOPE","resourceId":"123"}` | 21002 | 200 |
| backend business error | `{"systemCode":"SERVICE_A","resourceId":"ERROR_ACCOUNT"}` | 21001 | 200 |

---

## Verification

```bash
go test ./internal/handler/...
go build ./...
```
