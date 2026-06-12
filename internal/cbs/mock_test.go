package cbs

import (
	"os"
	"path/filepath"
	"testing"

	"engine-poc/internal/spec"
)

var testRespFields = []spec.ResponseField{
	{Name: "ResponseCode", Length: 8},
	{Name: "FIELD_A", Length: 15},
	{Name: "FIELD_B", Length: 3},
}

const fixtureYAML = `
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

func loadTestMock(t *testing.T, yaml string) *MockClient {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fixtures.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	mc, err := LoadMockClient(path, testRespFields)
	if err != nil {
		t.Fatalf("load mock client: %v", err)
	}
	return mc
}

func TestMatch_Found(t *testing.T) {
	mc := loadTestMock(t, fixtureYAML)
	resp, err := mc.Send(9001, CBSMessage{"MSG_TYPE": "0200", "RESOURCE_ID": "12345"})
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	// total length: 8+15+3 = 26
	if len(resp) != 26 {
		t.Errorf("expected 26 bytes, got %d", len(resp))
	}
}

func TestMatch_NotFound(t *testing.T) {
	mc := loadTestMock(t, fixtureYAML)
	_, err := mc.Send(9001, CBSMessage{"MSG_TYPE": "9999", "RESOURCE_ID": "NOMATCH"})
	if err == nil {
		t.Fatal("expected error when no fixture matches")
	}
}

func TestFirstMatchWins(t *testing.T) {
	mc := loadTestMock(t, fixtureYAML)
	// ERROR_ACCOUNT should match specific fixture first, not the general MSG_TYPE one
	resp, err := mc.Send(9001, CBSMessage{"MSG_TYPE": "0200", "RESOURCE_ID": "ERROR_ACCOUNT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ResponseCode field is first 8 bytes
	rc := string(resp[:8])
	if rc != "BACKEND_" {
		// "BACKEND_ERR_001" truncated to 8 = "BACKEND_"
		t.Errorf("expected BACKEND_ERR_001 response (first match), got ResponseCode=%q", rc)
	}
}

func TestLoadMockClient_FileNotFound(t *testing.T) {
	_, err := LoadMockClient("/nonexistent/fixtures.yaml", testRespFields)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadMockClient_BadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixtures.yaml")
	if err := os.WriteFile(path, []byte(":::bad:::"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadMockClient(path, testRespFields)
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}
