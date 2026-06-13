package codemap

import (
	"net/http"
	"testing"

	"engine-poc/internal/spec"
)

func TestMap_OK(t *testing.T) {
	m := New()
	got := m.Map("OK")
	if got.Code != 10000 {
		t.Errorf("expected 10000, got %d", got.Code)
	}
	if got.HTTPStatus != http.StatusOK {
		t.Errorf("expected HTTP 200, got %d", got.HTTPStatus)
	}
}

func TestMap_BackendErr001(t *testing.T) {
	m := New()
	got := m.Map("BACKEND_ERR_001")
	if got.Code != 21001 {
		t.Errorf("expected 21001, got %d", got.Code)
	}
}

func TestMap_BackendErr013(t *testing.T) {
	m := New()
	got := m.Map("BACKEND_ERR_013")
	if got.Code != 21013 {
		t.Errorf("expected 21013, got %d", got.Code)
	}
}

func TestMap_SvcErr001(t *testing.T) {
	m := New()
	got := m.Map("SVC_ERR_001")
	if got.Code != 22001 {
		t.Errorf("expected 22001, got %d", got.Code)
	}
}

func TestMap_Unknown(t *testing.T) {
	m := New()
	got := m.Map("GARBAGE_CODE")
	if got.Code != 20000 {
		t.Errorf("expected 20000 for unknown code, got %d", got.Code)
	}
	if got.HTTPStatus != http.StatusOK {
		t.Errorf("expected HTTP 200, got %d", got.HTTPStatus)
	}
}

func TestMap_AllBackendErr(t *testing.T) {
	m := New()
	for i := 1; i <= 13; i++ {
		code := "BACKEND_ERR_" + padInt(i)
		got := m.Map(code)
		expected := 21000 + i
		if got.Code != expected {
			t.Errorf("%s: expected %d, got %d", code, expected, got.Code)
		}
	}
}

func padInt(n int) string {
	if n < 10 {
		return "00" + string(rune('0'+n))
	}
	return "0" + string([]byte{byte('0' + n/10), byte('0' + n%10)})
}

func TestOverride_WinsOverStandard(t *testing.T) {
	overrides := map[string]spec.ResponseCodeDef{
		"19999": {EngineCode: 19999, HTTPStatus: 200, Description: "custom override", CBSCode: "OK"},
	}
	m := NewWithOverrides(overrides)
	got := m.Map("OK")
	if got.Code != 19999 {
		t.Errorf("expected override code 19999, got %d", got.Code)
	}
	if got.Message != "custom override" {
		t.Errorf("expected override message, got %q", got.Message)
	}
}

func TestOverride_NonOverriddenStillStandard(t *testing.T) {
	overrides := map[string]spec.ResponseCodeDef{
		"19999": {EngineCode: 19999, HTTPStatus: 200, Description: "custom", CBSCode: "OK"},
	}
	m := NewWithOverrides(overrides)
	got := m.Map("BACKEND_ERR_001")
	if got.Code != 21001 {
		t.Errorf("non-overridden code should still be 21001, got %d", got.Code)
	}
}
