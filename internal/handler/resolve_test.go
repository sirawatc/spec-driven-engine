package handler

import (
	"testing"

	"engine-poc/internal/spec"
)

func TestResolvePort_Known(t *testing.T) {
	systems := map[string]spec.SystemDef{
		"SERVICE_A": {Port: 9001},
	}
	port, err := ResolvePort("SERVICE_A", systems)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 9001 {
		t.Errorf("expected 9001, got %d", port)
	}
}

func TestResolvePort_Unknown(t *testing.T) {
	systems := map[string]spec.SystemDef{
		"SERVICE_A": {Port: 9001},
	}
	_, err := ResolvePort("UNKNOWN", systems)
	if err == nil || err.Code != 21002 {
		t.Errorf("expected ErrSystemCodeNotFound (21002), got %v", err)
	}
}

func TestResolvePort_EmptyMap(t *testing.T) {
	_, err := ResolvePort("SERVICE_A", map[string]spec.SystemDef{})
	if err == nil || err.Code != 21002 {
		t.Errorf("expected ErrSystemCodeNotFound (21002), got %v", err)
	}
}
