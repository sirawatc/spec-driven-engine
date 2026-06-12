# Story: E2-S3 — systemCode Resolution

**Epic:** E2 — HTTP Runtime  
**Status:** Ready for Development  
**Depends on:** E1-S2  
**Blocks:** E5-S1

---

## Summary

Implement `ResolvePort` — a pure function that maps a `systemCode` string to a TCP port using the spec's systems registry. If the code is not registered, it returns `ErrSystemCodeNotFound` (engine code `21002`). Used by the pipeline in E5-S1.

---

## Acceptance Criteria

- [ ] Known `systemCode` → returns the correct port number, no error
- [ ] Unknown `systemCode` → returns `ErrSystemCodeNotFound` (code `21002`, HTTP 200)
- [ ] Pure function: `ResolvePort(systemCode string, systems map[string]spec.SystemDef) (int, *EngineError)`
- [ ] Unit tests: known code resolves correctly, unknown code returns 21002

---

## Implementation (`internal/handler/resolve.go`)

```go
package handler

import "engine-poc/internal/spec"

// ResolvePort looks up systemCode in the spec systems map and returns the TCP port.
// Returns ErrSystemCodeNotFound if the code is not registered.
func ResolvePort(systemCode string, systems map[string]spec.SystemDef) (int, *EngineError) {
    def, ok := systems[systemCode]
    if !ok {
        return 0, ErrSystemCodeNotFound
    }
    return def.Port, nil
}
```

---

## Unit Tests (`internal/handler/resolve_test.go`)

| Test | systemCode | systems map | Expected |
|------|-----------|-------------|----------|
| Known code | `"SERVICE_A"` | `{"SERVICE_A": {Port: 9001}}` | returns `9001`, nil |
| Unknown code | `"UNKNOWN"` | `{"SERVICE_A": {Port: 9001}}` | returns `0`, `ErrSystemCodeNotFound` |
| Empty map | `"SERVICE_A"` | `{}` | returns `0`, `ErrSystemCodeNotFound` |

---

## Verification

```bash
go test ./internal/handler/...
go build ./...
```
