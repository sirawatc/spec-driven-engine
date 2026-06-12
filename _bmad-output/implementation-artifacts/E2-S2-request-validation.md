# Story: E2-S2 — Request Body Validation

**Epic:** E2 — HTTP Runtime  
**Status:** Ready for Development  
**Depends on:** E1-S2  
**Blocks:** E5-S1

---

## Summary

Implement request body parsing and field validation as a reusable function. Given a raw HTTP request body and a `spec.RequestSchema`, it returns the parsed fields map or an engine error. Lives in `internal/handler` — used by the pipeline in E5-S1.

---

## Acceptance Criteria

- [ ] Non-JSON body → returns engine code `31000` (input parse error), HTTP 400
- [ ] Missing required field → returns engine code `32000` (input validation error), HTTP 400, error message names the missing field
- [ ] `systemCode` absent from root → returns engine code `32000`
- [ ] Extra fields not in spec are accepted silently
- [ ] Unit tests cover all four paths above

---

## Engine Error Type

Define a shared error type in `internal/handler/errors.go`:

```go
package handler

import "net/http"

// EngineError carries the 5-digit engine code and HTTP status for a pipeline failure.
type EngineError struct {
    Code       int
    HTTPStatus int
    Message    string
}

func (e *EngineError) Error() string { return e.Message }

// Pre-defined sentinel errors for the pipeline.
var (
    ErrParseRequest = &EngineError{Code: 31000, HTTPStatus: http.StatusBadRequest, Message: "Input parse error"}
    ErrSystemCodeNotFound = &EngineError{Code: 21002, HTTPStatus: http.StatusOK, Message: "Route name not found"}
    ErrInternal     = &EngineError{Code: 40000, HTTPStatus: http.StatusInternalServerError, Message: "Internal server error"}
)

func ErrValidation(field string) *EngineError {
    return &EngineError{
        Code:       32000,
        HTTPStatus: http.StatusBadRequest,
        Message:    "Input validation error: missing required field: " + field,
    }
}
```

---

## Validation Function (`internal/handler/validate.go`)

```go
package handler

import (
    "encoding/json"
    "io"
    "net/http"

    "engine-poc/internal/spec"
)

// ParseAndValidate reads the request body, parses JSON, and validates
// required fields against schema. Returns parsed body map or EngineError.
func ParseAndValidate(r *http.Request, schema spec.RequestSchema) (map[string]any, *EngineError) {
    body, err := io.ReadAll(r.Body)
    if err != nil {
        return nil, ErrParseRequest
    }

    var data map[string]any
    if err := json.Unmarshal(body, &data); err != nil {
        return nil, ErrParseRequest
    }

    // systemCode is always required at root
    if _, ok := data["systemCode"]; !ok {
        return nil, ErrValidation("systemCode")
    }

    // Validate spec-defined required fields
    for name, def := range schema.Fields {
        if def.Required {
            if _, ok := data[name]; !ok {
                return nil, ErrValidation(name)
            }
        }
    }

    return data, nil
}
```

---

## Unit Tests (`internal/handler/validate_test.go`)

| Test | Input | Expected |
|------|-------|----------|
| Valid body | `{"systemCode":"X","resourceId":"123"}` | returns map, no error |
| Invalid JSON | `not-json` | `ErrParseRequest` (code 31000) |
| Missing systemCode | `{"resourceId":"123"}` | `ErrValidation("systemCode")` (code 32000) |
| Missing required spec field | `{"systemCode":"X"}` (schema requires resourceId) | `ErrValidation("resourceId")` (code 32000) |
| Extra field not in schema | `{"systemCode":"X","resourceId":"1","extra":"y"}` | accepted, no error |

---

## Verification

```bash
go test ./internal/handler/...
go build ./...
```
