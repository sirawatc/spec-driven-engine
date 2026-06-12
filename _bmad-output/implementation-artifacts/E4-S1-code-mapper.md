# Story: E4-S1 — Standard Code Table and Mapper

**Epic:** E4 — Response Code Mapper  
**Status:** Ready for Development  
**Depends on:** E1-S1  
**Blocks:** E4-S2, E5-S1

---

## Summary

Implement `codemap.Mapper` with the full standard backend→engine code table. Given a raw backend `ResponseCode` string, it returns the 5-digit `EngineCode` with HTTP status. Unknown backend codes fall back to `20000`.

---

## Acceptance Criteria

- [ ] Standard table in `internal/codemap/standard.go` covers all codes from the Response Code Specification:
  - `AA` → `10000`, HTTP 200
  - `BACKEND_ERR_001`–`BACKEND_ERR_013` → `21001`–`21013`, HTTP 200
  - `SVC_ERR_001` → `22001`, HTTP 200
  - Unmapped → `20000`, HTTP 200 (default business error)
- [ ] `Mapper.Map(cbsCode string) EngineCode` returns correct result for every entry
- [ ] Unknown backend code returns `20000`, not an error
- [ ] Unit tests: every standard code, unknown code fallback

---

## Types (`internal/codemap/mapper.go`)

```go
package codemap

// EngineCode is the resolved 5-digit response code with its HTTP status.
type EngineCode struct {
    Code       int
    HTTPStatus int
    Message    string
}

// Mapper translates backend ResponseCode → EngineCode.
type Mapper struct {
    table map[string]EngineCode
}

// New creates a Mapper seeded with the standard table.
func New() *Mapper {
    table := make(map[string]EngineCode, len(standardTable))
    for k, v := range standardTable {
        table[k] = v
    }
    return &Mapper{table: table}
}

// Map returns the EngineCode for the given backend response code.
// Unknown codes return the default business error (20000).
func (m *Mapper) Map(cbsCode string) EngineCode {
    if code, ok := m.table[cbsCode]; ok {
        return code
    }
    return m.table["__default__"]
}
```

---

## Standard Table (`internal/codemap/standard.go`)

```go
package codemap

import "net/http"

// standardTable is the built-in backend→engine code mapping.
// Source: [internal-docs]
var standardTable = map[string]EngineCode{
    // Success
    "OK": {Code: 10000, HTTPStatus: http.StatusOK, Message: "Success"},

    // Business errors — BACKEND_ERR series
    "BACKEND_ERR_001": {Code: 21001, HTTPStatus: http.StatusOK, Message: "Backend internal error"},
    "BACKEND_ERR_002": {Code: 21002, HTTPStatus: http.StatusOK, Message: "Route name not found"},
    "BACKEND_ERR_003": {Code: 21003, HTTPStatus: http.StatusOK, Message: "Transaction timeout"},
    "BACKEND_ERR_004": {Code: 21004, HTTPStatus: http.StatusOK, Message: "Program exception error"},
    "BACKEND_ERR_005": {Code: 21005, HTTPStatus: http.StatusOK, Message: "Operation reversal rejected"},
    "BACKEND_ERR_006": {Code: 21006, HTTPStatus: http.StatusOK, Message: "System is not ready to process"},
    "BACKEND_ERR_007": {Code: 21007, HTTPStatus: http.StatusOK, Message: "Duplicate application key"},
    "BACKEND_ERR_008": {Code: 21008, HTTPStatus: http.StatusOK, Message: "Unknown data format"},
    "BACKEND_ERR_009": {Code: 21009, HTTPStatus: http.StatusOK, Message: "Conversion not found"},
    "BACKEND_ERR_010": {Code: 21010, HTTPStatus: http.StatusOK, Message: "Host unavailable"},
    "BACKEND_ERR_011": {Code: 21011, HTTPStatus: http.StatusOK, Message: "System watchdog failed to start job"},
    "BACKEND_ERR_012": {Code: 21012, HTTPStatus: http.StatusOK, Message: "Invalid source id"},
    "BACKEND_ERR_013": {Code: 21013, HTTPStatus: http.StatusOK, Message: "Unknown error"},

    // Business errors — SVC_ERR series
    "SVC_ERR_001": {Code: 22001, HTTPStatus: http.StatusOK, Message: "No records found"},

    // Default: unmapped backend code
    "__default__": {Code: 20000, HTTPStatus: http.StatusOK, Message: "Business error"},
}
```

---

## Unit Tests (`internal/codemap/mapper_test.go`)

Test cases:

```go
func TestMap_AA(t *testing.T)          { /* AA → 10000, HTTP 200 */ }
func TestMap_DSP0001(t *testing.T)     { /* BACKEND_ERR_001 → 21001 */ }
func TestMap_DSP0013(t *testing.T)     { /* BACKEND_ERR_013 → 21013 */ }
func TestMap_SVC_ERR_001(t *testing.T)     { /* SVC_ERR_001 → 22001 */ }
func TestMap_Unknown(t *testing.T)     { /* "GARBAGE" → 20000, HTTP 200 */ }
func TestMap_AllDSP(t *testing.T)      { /* loop DSP0001–DSP0013, verify sequential codes */ }
```

---

## Verification

```bash
go test ./internal/codemap/...
go build ./...
```
