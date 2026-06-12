package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"engine-poc/internal/spec"
)

var testSchema = spec.RequestSchema{
	Fields: map[string]spec.FieldDef{
		"systemCode": {Type: "string", Required: true},
		"resourceId": {Type: "string", Required: true},
	},
}

func makeReq(body string) *httptest.ResponseRecorder {
	_ = body
	return httptest.NewRecorder()
}

func TestParseAndValidate_Valid(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"systemCode":"SVC_A","resourceId":"123"}`))
	data, err := ParseAndValidate(r, testSchema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data["systemCode"] != "SVC_A" {
		t.Errorf("expected systemCode SVC_A, got %v", data["systemCode"])
	}
}

func TestParseAndValidate_InvalidJSON(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader("not-json"))
	_, err := ParseAndValidate(r, testSchema)
	if err == nil || err.Code != 31000 {
		t.Errorf("expected code 31000, got %v", err)
	}
}

func TestParseAndValidate_MissingSystemCode(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"resourceId":"123"}`))
	_, err := ParseAndValidate(r, testSchema)
	if err == nil || err.Code != 32000 {
		t.Errorf("expected code 32000, got %v", err)
	}
}

func TestParseAndValidate_MissingRequiredField(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"systemCode":"SVC_A"}`))
	_, err := ParseAndValidate(r, testSchema)
	if err == nil || err.Code != 32000 {
		t.Errorf("expected code 32000, got %v", err)
	}
}

func TestParseAndValidate_ExtraFieldAccepted(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"systemCode":"SVC_A","resourceId":"1","extra":"y"}`))
	_, err := ParseAndValidate(r, testSchema)
	if err != nil {
		t.Errorf("extra field should be accepted, got: %v", err)
	}
}
