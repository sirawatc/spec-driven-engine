package integration_test

import (
	"encoding/json"
	"io"
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

	registry, err := spec.LoadRegistry("../../spec")
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

	bodyBytes := new(strings.Builder)
	io.Copy(bodyBytes, resp.Body)
	if !strings.Contains(bodyBytes.String(), "Service Engine") {
		t.Error("dashboard body should contain 'Service Engine'")
	}
}
