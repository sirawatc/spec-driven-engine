# Story: E7-S2 — End-to-End Integration Test

**Epic:** E7 — PoC End-to-End Validation  
**Status:** Ready for Development  
**Depends on:** E7-S1, E6-S2  
**Blocks:** nothing (final story)

---

## Summary

Write an integration test that starts the full engine in-process (using `httptest`) and exercises all PoC success criteria via HTTP. All 5 request test cases plus the dashboard smoke test must pass with `go test ./...`.

---

## Acceptance Criteria

- [ ] Test starts engine with `spec/v1/example.yaml` and `mock/fixtures.yaml` on a random port
- [ ] 5 request test cases, each asserting HTTP status + `code` field in JSON body:
  1. Happy path → code `10000`, HTTP 200, `data.balance` and `data.currency` present
  2. Missing `resourceId` → code `32000`, HTTP 400
  3. Unknown `systemCode` → code `21002`, HTTP 200
  4. backend business error (`ERROR_ACCOUNT`) → code `21001`, HTTP 200
  5. backend timeout (`TIMEOUT_ACCOUNT`) → code `21003`, HTTP 200
- [ ] Dashboard smoke test: `GET /dashboard` → HTTP 200, body contains spec name string
- [ ] All tests pass with `go test ./...`

---

## Test File (`e2e_test.go` at repo root, or `internal/integration/e2e_test.go`)

```go
package integration_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "engine-poc/internal/cbs"
    "engine-poc/internal/codemap"
    "engine-poc/internal/dashboard"
    "engine-poc/internal/handler"
    "engine-poc/internal/server"
    "engine-poc/internal/spec"
)

func buildTestServer(t *testing.T) *httptest.Server {
    t.Helper()

    registry, err := spec.LoadRegistry("../../spec") // adjust path from test file location
    if err != nil {
        t.Fatalf("load registry: %v", err)
    }

    handlerMap := make(map[string]*handler.Handler)
    for version, s := range registry {
        mapper := codemap.NewWithOverrides(s.ResponseCodes)
        var respFields []spec.ResponseField
        if len(s.Endpoints) > 0 {
            respFields = s.Endpoints[0].Transform.Response.Fields
        }
        mockClient, err := cbs.LoadMockClient("../../mock/fixtures.yaml", respFields)
        if err != nil {
            t.Fatalf("load mock client: %v", err)
        }
        handlerMap[version] = handler.New(s, mockClient, mapper)
    }

    factory := func(version string, endpoint spec.Endpoint) http.HandlerFunc {
        return handlerMap[version].Factory()(version, endpoint)
    }

    dash := dashboard.New(registry)
    srv := server.New(registry, factory, dash)
    return httptest.NewServer(srv.Router())
}

type apiResp struct {
    Code    int            `json:"code"`
    Message string         `json:"message"`
    Data    map[string]any `json:"data"`
}

func post(t *testing.T, url, body string) *http.Response {
    t.Helper()
    resp, err := http.Post(url, "application/json", strings.NewReader(body))
    if err != nil {
        t.Fatalf("POST %s: %v", url, err)
    }
    return resp
}

func decodeResp(t *testing.T, resp *http.Response) apiResp {
    t.Helper()
    var r apiResp
    if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
        t.Fatalf("decode response: %v", err)
    }
    return r
}

func TestE2E_HappyPath(t *testing.T) {
    srv := buildTestServer(t)
    defer srv.Close()

    resp := post(t, srv.URL+"/v1/example/query",
        `{"systemCode":"SERVICE_A","resourceId":"1234567890"}`)

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected HTTP 200, got %d", resp.StatusCode)
    }
    body := decodeResp(t, resp)
    if body.Code != 10000 {
        t.Errorf("expected code 10000, got %d", body.Code)
    }
    if body.Data["balance"] == "" {
        t.Error("expected balance in data")
    }
    if body.Data["currency"] == "" {
        t.Error("expected currency in data")
    }
}

func TestE2E_MissingField(t *testing.T) {
    srv := buildTestServer(t)
    defer srv.Close()

    resp := post(t, srv.URL+"/v1/example/query",
        `{"systemCode":"SERVICE_A"}`)

    if resp.StatusCode != http.StatusBadRequest {
        t.Errorf("expected HTTP 400, got %d", resp.StatusCode)
    }
    body := decodeResp(t, resp)
    if body.Code != 32000 {
        t.Errorf("expected code 32000, got %d", body.Code)
    }
}

func TestE2E_UnknownSystemCode(t *testing.T) {
    srv := buildTestServer(t)
    defer srv.Close()

    resp := post(t, srv.URL+"/v1/example/query",
        `{"systemCode":"UNKNOWN","resourceId":"1234567890"}`)

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected HTTP 200, got %d", resp.StatusCode)
    }
    body := decodeResp(t, resp)
    if body.Code != 21002 {
        t.Errorf("expected code 21002, got %d", body.Code)
    }
}

func TestE2E_CBSBusinessError(t *testing.T) {
    srv := buildTestServer(t)
    defer srv.Close()

    resp := post(t, srv.URL+"/v1/example/query",
        `{"systemCode":"SERVICE_A","resourceId":"ERROR_ACCOUNT"}`)

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected HTTP 200, got %d", resp.StatusCode)
    }
    body := decodeResp(t, resp)
    if body.Code != 21001 {
        t.Errorf("expected code 21001, got %d", body.Code)
    }
}

func TestE2E_CBSTimeout(t *testing.T) {
    srv := buildTestServer(t)
    defer srv.Close()

    resp := post(t, srv.URL+"/v1/example/query",
        `{"systemCode":"SERVICE_A","resourceId":"TIMEOUT_ACCOUNT"}`)

    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected HTTP 200, got %d", resp.StatusCode)
    }
    body := decodeResp(t, resp)
    if body.Code != 21003 {
        t.Errorf("expected code 21003, got %d", body.Code)
    }
}

func TestE2E_Dashboard_Smoke(t *testing.T) {
    srv := buildTestServer(t)
    defer srv.Close()

    resp, err := http.Get(srv.URL + "/dashboard")
    if err != nil {
        t.Fatalf("GET /dashboard: %v", err)
    }
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected HTTP 200, got %d", resp.StatusCode)
    }

    // Body should contain the spec name
    var buf strings.Builder
    _, _ = buf.ReadFrom(resp.Body)
    if !strings.Contains(buf.String(), "Service Engine") {
        t.Error("dashboard body should contain spec name 'Service Engine'")
    }
}
```

---

## Note on `server.Router()`

E7-S2 requires `server.New()` to expose a `Router()` method returning `http.Handler` so `httptest.NewServer` can use it directly without binding a port:

```go
// Add to Server in internal/server/server.go:
func (s *Server) Router() http.Handler {
    return s.router
}
```

---

## Verification

```bash
go test ./...   # all tests pass, including unit tests from all packages
```

If any test fails, the PoC success criteria are not met.
