# Story: E1-S2 — Spec Loader

**Epic:** E1 — Engine Foundation  
**Status:** Ready for Development  
**Depends on:** E1-S1  
**Blocks:** E1-S3, E2-S2, E2-S3, E3-S1, E4-S2

---

## Summary

Implement `spec.Load(path string) (*Spec, error)` and all supporting types. This replaces the stubs created in E1-S1 with a fully typed, validated YAML parser. Every downstream component depends on the `Spec` struct produced here.

---

## Acceptance Criteria

- [ ] Parses `version`, `name`, `description`, `systems`, `response_codes`, `endpoints` from YAML
- [ ] `version` is required — missing version returns an error, not a panic
- [ ] Returns a descriptive error if required fields are missing or types are wrong
- [ ] `Spec` struct is fully typed (no `map[string]interface{}` fields)
- [ ] Logs `name`, `version`, and UTC load timestamp on successful load
- [ ] Unit tests: valid spec loads correctly, missing `version` errors, missing `name` errors, unknown top-level field is silently ignored
- [ ] `spec/v1/example.yaml` (created in E7-S1) loads without error — write `example.yaml` placeholder now to verify

---

## Types to Define (`internal/spec/model.go`)

Replace the stubs from E1-S1 with the full model:

```go
package spec

// Spec is the root in-memory representation of one parsed spec file.
type Spec struct {
    Version       string                     `yaml:"version"`
    Name          string                     `yaml:"name"`
    Description   string                     `yaml:"description"`
    Systems       map[string]SystemDef       `yaml:"systems"`
    ResponseCodes map[string]ResponseCodeDef `yaml:"response_codes"`
    Endpoints     []Endpoint                 `yaml:"endpoints"`
}

type SystemDef struct {
    Port        int    `yaml:"port"`
    Description string `yaml:"description"`
}

type ResponseCodeDef struct {
    Type        string `yaml:"type"`
    HTTPStatus  int    `yaml:"http_status"`
    Description string `yaml:"description"`
    CBSCode     string `yaml:"backend_code"`
}

type Endpoint struct {
    Path          string          `yaml:"path"`
    Method        string          `yaml:"method"`
    Summary       string          `yaml:"summary"`
    Description   string          `yaml:"description"`
    Request       RequestSchema   `yaml:"request"`
    Transform     TransformSpec   `yaml:"transform"`
    ResponseCodes []EndpointCode  `yaml:"response_codes"`
}

type RequestSchema struct {
    Fields map[string]FieldDef `yaml:"fields"`
}

type FieldDef struct {
    Type        string `yaml:"type"`
    Required    bool   `yaml:"required"`
    Description string `yaml:"description"`
}

type TransformSpec struct {
    Request  []FieldRule      `yaml:"request"`
    Response ResponseTransform `yaml:"response"`
}

// FieldRule defines one fixed-length field in the TCP request message.
type FieldRule struct {
    Name   string `yaml:"name"`
    Length int    `yaml:"length"`
    Value  string `yaml:"value"`  // literal or {{request.body.<field>}} template
    Align  string `yaml:"align"`  // "left" (default) or "right"
    Pad    string `yaml:"pad"`    // default " " (space)
}

type ResponseTransform struct {
    Fields  []ResponseField   `yaml:"fields"`
    Mapping map[string]string `yaml:"mapping"` // HTTP key → {{cbs.<field>}} template
}

// ResponseField defines one fixed-length field in the TCP response message.
type ResponseField struct {
    Name   string `yaml:"name"`
    Length int    `yaml:"length"`
}

type EndpointCode struct {
    Code        int    `yaml:"code"`
    Description string `yaml:"description"`
}
```

---

## Loader (`internal/spec/loader.go`)

```go
package spec

import (
    "fmt"
    "log"
    "os"
    "time"

    "gopkg.in/yaml.v3"
)

func Load(path string) (*Spec, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("spec: read %s: %w", path, err)
    }

    var s Spec
    if err := yaml.Unmarshal(data, &s); err != nil {
        return nil, fmt.Errorf("spec: parse %s: %w", path, err)
    }

    if err := validate(&s); err != nil {
        return nil, fmt.Errorf("spec: validate %s: %w", path, err)
    }

    log.Printf("spec loaded: name=%q version=%q at=%s", s.Name, s.Version, time.Now().UTC().Format(time.RFC3339))
    return &s, nil
}

func validate(s *Spec) error {
    if s.Version == "" {
        return fmt.Errorf("missing required field: version")
    }
    if s.Name == "" {
        return fmt.Errorf("missing required field: name")
    }
    for i, ep := range s.Endpoints {
        if ep.Path == "" {
            return fmt.Errorf("endpoint[%d]: missing path", i)
        }
        if ep.Method == "" {
            return fmt.Errorf("endpoint[%d] %s: missing method", i, ep.Path)
        }
        for j, rule := range ep.Transform.Request {
            if rule.Name == "" {
                return fmt.Errorf("endpoint %s: request transform[%d]: missing name", ep.Path, j)
            }
            if rule.Length <= 0 {
                return fmt.Errorf("endpoint %s: request transform[%d] %s: length must be > 0", ep.Path, j, rule.Name)
            }
        }
        for j, rf := range ep.Transform.Response.Fields {
            if rf.Name == "" {
                return fmt.Errorf("endpoint %s: response field[%d]: missing name", ep.Path, j)
            }
            if rf.Length <= 0 {
                return fmt.Errorf("endpoint %s: response field[%d] %s: length must be > 0", ep.Path, j, rf.Name)
            }
        }
    }
    return nil
}
```

---

## Dependencies to Add

```bash
go get gopkg.in/yaml.v3
```

---

## Unit Tests (`internal/spec/loader_test.go`)

Test cases required:
1. Valid minimal spec loads without error and fields are populated
2. Missing `version` → error containing "version"
3. Missing `name` → error containing "name"
4. Invalid YAML syntax → error
5. File not found → error
6. Unknown top-level YAML key → ignored (no error)
7. Endpoint with missing `path` → error
8. Transform field with `length: 0` → error

---

## Placeholder Example Spec

Create `spec/v1/example.yaml` as a minimal valid placeholder (full content added in E7-S1):

```yaml
version: "1.0.0"
name: "Service Engine - Example Domain"
description: "Resource query operations"

systems:
  SERVICE_A:
    port: 9001
    description: "backend service port"

endpoints:
  - path: /example/query
    method: POST
    summary: "Inquire resource value"
    request:
      fields:
        systemCode:
          type: string
          required: true
          description: "Routes request to backend port"
        resourceId:
          type: string
          required: true
          description: "Resource identifier to inquire"
    transform:
      request:
        - name: MSG_TYPE
          length: 4
          value: "0200"
        - name: RESOURCE_ID
          length: 16
          value: "{{request.body.resourceId}}"
      response:
        fields:
          - name: ResponseCode
            length: 8
          - name: FIELD_A
            length: 15
          - name: FIELD_B
            length: 3
        mapping:
          balance: "{{cbs.FIELD_A}}"
          currency: "{{cbs.FIELD_B}}"
```

---

## Verification

```bash
go test ./internal/spec/...    # all tests pass
go build ./...                 # still compiles
```
