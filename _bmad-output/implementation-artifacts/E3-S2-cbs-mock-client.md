# Story: E3-S2 — backend Client Interface and Mock

**Epic:** E3 — backend Integration Layer  
**Status:** Ready for Development  
**Depends on:** E1-S1  
**Blocks:** E3-S3, E5-S1

---

## Summary

Flesh out the `cbs.Client` interface (stubbed in E1-S1) with correct type definitions, and implement `MockClient` driven by a YAML fixture file. The mock is what the PoC uses instead of a real TCP connection.

---

## Acceptance Criteria

- [ ] `Client` interface: `Send(port int, msg CBSMessage) (BackendResponse, error)`
- [ ] `CBSMessage` = `map[string]string` (field name → trimmed string value)
- [ ] `BackendResponse` = `[]byte` (raw fixed-length bytes)
- [ ] `MockClient` loaded from `mock/fixtures.yaml`
- [ ] Fixture match: first fixture where ALL `match` key-values match the message fields wins
- [ ] No matching fixture → returns error (not a business error — test setup problem)
- [ ] Unit tests: match found, no match returns error, first-match-wins ordering

---

## Client Interface (`internal/cbs/client.go`)

Replace E1-S1 stub with:

```go
package cbs

// CBSMessage is the pre-serialization representation of a backend request:
// field name → string value (already resolved from templates, not yet padded into bytes).
type CBSMessage map[string]string

// BackendResponse is the raw fixed-length byte slice received from the backend service.
type BackendResponse []byte

// Client sends a backend message to the given TCP port and returns the raw response.
type Client interface {
    Send(port int, msg CBSMessage) (BackendResponse, error)
}
```

---

## Mock Client (`internal/cbs/mock.go`)

```go
package cbs

import (
    "fmt"
    "os"
    "strings"

    "gopkg.in/yaml.v3"
)

// Fixture is one entry in mock/fixtures.yaml.
type Fixture struct {
    Match    map[string]string `yaml:"match"`
    Response map[string]string `yaml:"response"` // field name → value (unpadded)
}

// MockClient matches incoming messages against loaded fixtures.
type MockClient struct {
    fixtures []Fixture
    // fieldLengths maps response field name → length for serializing mock response bytes.
    // Populated from spec at construction time.
    fieldLengths map[string]int
}

// LoadMockClient reads fixture YAML and returns a MockClient.
// fieldLengths: map of response field name → length, used to build fixed-length response bytes.
func LoadMockClient(fixturePath string, fieldLengths map[string]int) (*MockClient, error) {
    data, err := os.ReadFile(fixturePath)
    if err != nil {
        return nil, fmt.Errorf("mock client: read fixtures %s: %w", fixturePath, err)
    }
    var fixtures []Fixture
    if err := yaml.Unmarshal(data, &fixtures); err != nil {
        return nil, fmt.Errorf("mock client: parse fixtures %s: %w", fixturePath, err)
    }
    return &MockClient{fixtures: fixtures, fieldLengths: fieldLengths}, nil
}

func (m *MockClient) Send(port int, msg CBSMessage) (BackendResponse, error) {
    for _, f := range m.fixtures {
        if matchesAll(msg, f.Match) {
            return m.serialize(f.Response), nil
        }
    }
    return nil, fmt.Errorf("mock client: no fixture matched message: %v", msg)
}

// matchesAll returns true if all key-values in match are present and equal in msg.
func matchesAll(msg CBSMessage, match map[string]string) bool {
    for k, v := range match {
        if msg[k] != v {
            return false
        }
    }
    return true
}

// serialize builds fixed-length response bytes from the fixture response map.
// Fields not in fieldLengths are skipped. Fields are ordered by fieldLengths key order
// — caller must pass fieldLengths in the correct spec-defined order.
func (m *MockClient) serialize(response map[string]string) BackendResponse {
    var buf strings.Builder
    for name, length := range m.fieldLengths {
        value := response[name]
        if len(value) >= length {
            buf.WriteString(value[:length])
        } else {
            buf.WriteString(value + strings.Repeat(" ", length-len(value)))
        }
    }
    return BackendResponse(buf.String())
}
```

> **Note on field ordering:** Go maps have no stable iteration order. The `fieldLengths` passed to `MockClient` must be derived from the spec's ordered `response.fields` list. Use a slice-based approach in the actual implementation rather than a plain map — see implementation notes below.

### Revised approach for ordered fields

Pass `[]spec.ResponseField` instead of `map[string]int` to preserve spec-defined order:

```go
type MockClient struct {
    fixtures     []Fixture
    respFields   []spec.ResponseField  // ordered
}
```

Serialize by iterating `respFields` in order. This matches how the real `TCPClient` would read the wire.

---

## Fixture File Format (`mock/fixtures.yaml`)

```yaml
# First match wins — put more specific matches before general ones

- match:
    RESOURCE_ID: "ERROR_ACCOUNT"
  response:
    ResponseCode: "BACKEND_ERR_001"
    FIELD_A: "000000000000000"
    FIELD_B: "   "

- match:
    RESOURCE_ID: "TIMEOUT_ACCOUNT"
  response:
    ResponseCode: "BACKEND_ERR_003"
    FIELD_A: "000000000000000"
    FIELD_B: "   "

- match:
    MSG_TYPE: "0200"
  response:
    ResponseCode: "OK"
    FIELD_A: "000000012345.67"
    FIELD_B: "THB"
```

---

## Unit Tests (`internal/cbs/mock_test.go`)

| Test | Description |
|------|-------------|
| `Match_found` | Message matches first fixture → correct response bytes |
| `Match_notfound` | No fixture matches → error returned |
| `FirstMatchWins` | More specific fixture listed first wins over general one |
| `LoadMockClient_file_not_found` | Missing fixture file → error |
| `LoadMockClient_bad_yaml` | Invalid YAML → error |

---

## Verification

```bash
go test ./internal/cbs/...
go build ./...
```
