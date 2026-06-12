# Story: E4-S2 — Spec-Level Response Code Overrides

**Epic:** E4 — Response Code Mapper  
**Status:** Ready for Development  
**Depends on:** E4-S1, E1-S2  
**Blocks:** E5-S1

---

## Summary

Extend `Mapper` to accept per-spec `response_codes` overrides that merge on top of the standard table at construction time. An override wins for the same backend code. The standard table is never mutated.

---

## Acceptance Criteria

- [ ] `NewWithOverrides(overrides map[string]spec.ResponseCodeDef) *Mapper` merges overrides on top of standard table
- [ ] Override wins over standard table for the same backend code
- [ ] Standard table itself is never mutated (copy-on-construct)
- [ ] Unit test: override for a known backend code returns override values; non-overridden codes still return standard values

---

## Implementation (extend `internal/codemap/mapper.go`)

Add a constructor that accepts spec overrides:

```go
// NewWithOverrides creates a Mapper seeded with the standard table and merged overrides.
// Overrides take precedence; the standard table is not mutated.
func NewWithOverrides(overrides map[string]spec.ResponseCodeDef) *Mapper {
    table := make(map[string]EngineCode, len(standardTable)+len(overrides))
    for k, v := range standardTable {
        table[k] = v
    }
    for cbsCode, def := range overrides {
        table[cbsCode] = EngineCode{
            Code:       def.Code,   // NOTE: spec.ResponseCodeDef uses int key — see below
            HTTPStatus: def.HTTPStatus,
            Description: def.Description,
        }
    }
    return &Mapper{table: table}
}
```

> **Note on spec.ResponseCodeDef:** The YAML key for `response_codes` is the 5-digit engine code (e.g. `22010`), not the backend code. The `backend_code` field inside the definition holds the backend code string. Update `NewWithOverrides` to key by `def.CBSCode`:

```go
for _, def := range overrides {
    if def.CBSCode == "" {
        continue // skip overrides without a backend code
    }
    table[def.CBSCode] = EngineCode{
        Code:       engineCodeFromKey, // pass as separate arg or restructure
        HTTPStatus: def.HTTPStatus,
        Message:    def.Description,
    }
}
```

The simplest approach: `spec.ResponseCodeDef` includes the engine code as a field. Revise the model in E1-S2 if needed:

```go
type ResponseCodeDef struct {
    EngineCode  int    `yaml:"-"` // set from the map key during registry load
    Type        string `yaml:"type"`
    HTTPStatus  int    `yaml:"http_status"`
    Description string `yaml:"description"`
    CBSCode     string `yaml:"backend_code"`
}
```

Populate `EngineCode` in `spec.Load` after unmarshalling:

```go
for key, def := range s.ResponseCodes {
    code, _ := strconv.Atoi(key)
    def.EngineCode = code
    s.ResponseCodes[key] = def
}
```

---

## Unit Tests (`internal/codemap/mapper_test.go`, add)

```go
func TestOverride_WinsOverStandard(t *testing.T) {
    // Override AA to return a different message
    overrides := map[string]spec.ResponseCodeDef{
        "AA_OVERRIDE": {EngineCode: 19999, HTTPStatus: 200, Description: "custom", CBSCode: "OK"},
    }
    m := NewWithOverrides(...)
    got := m.Map("OK")
    // got.Code should be 19999 (override), not 10000 (standard)
}

func TestOverride_NonOverriddenStillStandard(t *testing.T) {
    // BACKEND_ERR_001 not overridden → still returns 21001
}
```

---

## Verification

```bash
go test ./internal/codemap/...
go build ./...
```
