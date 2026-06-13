package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"engine-poc/internal/cbs"
	"engine-poc/internal/codemap"
	"engine-poc/internal/spec"
)

const testSpecYAML = `
version: "1.0.0"
name: "Test Spec"
systems:
  SERVICE_A:
    port: 9001
    description: "test service"
endpoints:
  - path: /example/query
    method: POST
    summary: "test"
    request:
      fields:
        systemCode:
          type: string
          required: true
        resourceId:
          type: string
          required: true
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
            length: 16
          - name: FIELD_A
            length: 15
          - name: FIELD_B
            length: 3
        mapping:
          balance: "{{cbs.FIELD_A}}"
          currency: "{{cbs.FIELD_B}}"
`

const testFixturesYAML = `
- match:
    RESOURCE_ID: "ERROR_ACCOUNT"
  response:
    ResponseCode: "BACKEND_ERR_001"
    FIELD_A: "000000000000000"
    FIELD_B: "   "
- match:
    MSG_TYPE: "0200"
  response:
    ResponseCode: "OK"
    FIELD_A: "000000012345.67"
    FIELD_B: "THB"
`

func buildTestHandler(t *testing.T) (*Handler, spec.Endpoint) {
	t.Helper()

	dir := t.TempDir()
	specPath := filepath.Join(dir, "spec.yaml")
	fixturePath := filepath.Join(dir, "fixtures.yaml")

	if err := os.WriteFile(specPath, []byte(testSpecYAML), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fixturePath, []byte(testFixturesYAML), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := spec.Load(specPath)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}

	endpoint := s.Endpoints[0]
	mockClient, err := cbs.LoadMockClient(fixturePath, endpoint.Transform.Response.Fields)
	if err != nil {
		t.Fatalf("load mock: %v", err)
	}

	mapper := codemap.New()
	h := New(s, mockClient, mapper)
	return h, endpoint
}

func callHandler(t *testing.T, h *Handler, endpoint spec.Endpoint, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", "/v1/example/query", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handle(w, r, endpoint)
	return w
}

type testResp struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Data    map[string]any `json:"data"`
}

func TestHandler_HappyPath(t *testing.T) {
	h, ep := buildTestHandler(t)
	w := callHandler(t, h, ep, `{"systemCode":"SERVICE_A","resourceId":"1234567890"}`)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp testResp
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 10000 {
		t.Errorf("expected code 10000, got %d", resp.Code)
	}
}

func TestHandler_InvalidJSON(t *testing.T) {
	h, ep := buildTestHandler(t)
	w := callHandler(t, h, ep, "not-json")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp testResp
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 31000 {
		t.Errorf("expected code 31000, got %d", resp.Code)
	}
}

func TestHandler_MissingResourceId(t *testing.T) {
	h, ep := buildTestHandler(t)
	w := callHandler(t, h, ep, `{"systemCode":"SERVICE_A"}`)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	var resp testResp
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 32000 {
		t.Errorf("expected code 32000, got %d", resp.Code)
	}
}

func TestHandler_UnknownSystemCode(t *testing.T) {
	h, ep := buildTestHandler(t)
	w := callHandler(t, h, ep, `{"systemCode":"NOPE","resourceId":"123"}`)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp testResp
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 21002 {
		t.Errorf("expected code 21002, got %d", resp.Code)
	}
}

func TestHandler_BackendBusinessError(t *testing.T) {
	h, ep := buildTestHandler(t)
	w := callHandler(t, h, ep, `{"systemCode":"SERVICE_A","resourceId":"ERROR_ACCOUNT"}`)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp testResp
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 21001 {
		t.Errorf("expected code 21001, got %d", resp.Code)
	}
}
