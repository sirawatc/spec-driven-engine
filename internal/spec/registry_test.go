package spec

import (
	"os"
	"path/filepath"
	"testing"
)

const simpleSpec = `
version: "1.0.0"
name: "Simple Spec"
endpoints:
  - path: /test
    method: POST
    transform:
      request:
        - name: F1
          length: 4
          value: "0001"
      response:
        fields:
          - name: ResponseCode
            length: 8
`

const anotherSpec = `
version: "2.0.0"
name: "Another Spec"
endpoints:
  - path: /test
    method: POST
    transform:
      request:
        - name: F1
          length: 4
          value: "0002"
      response:
        fields:
          - name: ResponseCode
            length: 8
`

func makeVersionDir(t *testing.T, root, version, content string) {
	t.Helper()
	dir := filepath.Join(root, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRegistry_SingleVersion(t *testing.T) {
	root := t.TempDir()
	makeVersionDir(t, root, "v1", simpleSpec)

	reg, err := LoadRegistry(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := reg["v1"]; !ok {
		t.Error("expected v1 in registry")
	}
}

func TestLoadRegistry_TwoVersions(t *testing.T) {
	root := t.TempDir()
	makeVersionDir(t, root, "v1", simpleSpec)
	makeVersionDir(t, root, "v2", anotherSpec)

	reg, err := LoadRegistry(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg) != 2 {
		t.Errorf("expected 2 versions, got %d", len(reg))
	}
	if reg["v1"].Version != "1.0.0" {
		t.Errorf("v1 version mismatch")
	}
	if reg["v2"].Version != "2.0.0" {
		t.Errorf("v2 version mismatch")
	}
}

func TestLoadRegistry_BadFile(t *testing.T) {
	root := t.TempDir()
	makeVersionDir(t, root, "v1", simpleSpec)

	badDir := filepath.Join(root, "v2")
	if err := os.MkdirAll(badDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "spec.yaml"), []byte(":::bad yaml:::"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadRegistry(root)
	if err == nil {
		t.Fatal("expected error for bad yaml in v2")
	}
}

func TestLoadRegistry_EmptyDirectory(t *testing.T) {
	root := t.TempDir()

	_, err := LoadRegistry(root)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
}

func TestLoadRegistry_SubdirWithNoYAML(t *testing.T) {
	root := t.TempDir()
	makeVersionDir(t, root, "v1", simpleSpec)
	// Create a subdir with no yaml
	if err := os.MkdirAll(filepath.Join(root, "empty"), 0755); err != nil {
		t.Fatal(err)
	}

	reg, err := LoadRegistry(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg) != 1 {
		t.Errorf("expected only 1 version (empty dir skipped), got %d", len(reg))
	}
}
