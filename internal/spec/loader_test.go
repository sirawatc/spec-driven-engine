package spec

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validMinimalSpec = `
version: "1.0.0"
name: "Test Spec"
description: "A test spec"
systems:
  SVC_A:
    port: 9001
    description: "service a"
endpoints:
  - path: /test
    method: POST
    summary: "test endpoint"
    transform:
      request:
        - name: FIELD1
          length: 4
          value: "0200"
      response:
        fields:
          - name: ResponseCode
            length: 8
        mapping:
          result: "{{cbs.ResponseCode}}"
`

func TestLoad_ValidSpec(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", validMinimalSpec)

	s, err := Load(path)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if s.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", s.Version)
	}
	if s.Name != "Test Spec" {
		t.Errorf("expected name 'Test Spec', got %s", s.Name)
	}
	if len(s.Endpoints) != 1 {
		t.Errorf("expected 1 endpoint, got %d", len(s.Endpoints))
	}
}

func TestLoad_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", `name: "Test"\nendpoints: []`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestLoad_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", `version: "1.0"`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "spec.yaml", `:::invalid yaml:::`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid yaml")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/spec.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_UnknownTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	content := validMinimalSpec + "\nunknown_key: ignored\n"
	path := writeFile(t, dir, "spec.yaml", content)

	_, err := Load(path)
	if err != nil {
		t.Fatalf("unknown top-level key should be ignored, got: %v", err)
	}
}

func TestLoad_EndpointMissingPath(t *testing.T) {
	dir := t.TempDir()
	content := `
version: "1.0"
name: "Test"
endpoints:
  - method: POST
    transform:
      request:
        - name: F1
          length: 4
          value: "1234"
      response:
        fields:
          - name: RC
            length: 8
`
	path := writeFile(t, dir, "spec.yaml", content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for endpoint missing path")
	}
}

func TestLoad_TransformFieldZeroLength(t *testing.T) {
	dir := t.TempDir()
	content := `
version: "1.0"
name: "Test"
endpoints:
  - path: /test
    method: POST
    transform:
      request:
        - name: FIELD1
          length: 0
          value: "test"
      response:
        fields:
          - name: RC
            length: 8
`
	path := writeFile(t, dir, "spec.yaml", content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for transform field with zero length")
	}
}
