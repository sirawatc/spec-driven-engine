package transformer

import (
	"testing"

	"engine-poc/internal/cbs"
	"engine-poc/internal/spec"
)

func TestToTCP_Literal(t *testing.T) {
	rules := []spec.FieldRule{{Name: "MSG_TYPE", Length: 4, Value: "0200"}}
	msg, raw, err := ToTCP(map[string]any{}, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != "0200" {
		t.Errorf("expected '0200', got %q", string(raw))
	}
	if msg["MSG_TYPE"] != "0200" {
		t.Errorf("expected msg[MSG_TYPE]='0200', got %q", msg["MSG_TYPE"])
	}
}

func TestToTCP_Template(t *testing.T) {
	rules := []spec.FieldRule{{Name: "RESOURCE_ID", Length: 16, Value: "{{request.body.resourceId}}"}}
	body := map[string]any{"resourceId": "ABC123"}
	_, raw, err := ToTCP(body, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(raw) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(raw))
	}
	if string(raw[:6]) != "ABC123" {
		t.Errorf("expected 'ABC123' at start, got %q", string(raw[:6]))
	}
}

func TestToTCP_LeftAlign(t *testing.T) {
	rules := []spec.FieldRule{{Name: "F", Length: 8, Value: "abc", Align: "left"}}
	_, raw, err := ToTCP(map[string]any{}, rules)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "abc     " {
		t.Errorf("expected left-padded 'abc     ', got %q", string(raw))
	}
}

func TestToTCP_RightAlign(t *testing.T) {
	rules := []spec.FieldRule{{Name: "F", Length: 8, Value: "abc", Align: "right"}}
	_, raw, err := ToTCP(map[string]any{}, rules)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "     abc" {
		t.Errorf("expected right-padded '     abc', got %q", string(raw))
	}
}

func TestToTCP_CustomPad(t *testing.T) {
	rules := []spec.FieldRule{{Name: "F", Length: 8, Value: "42", Pad: "0", Align: "right"}}
	_, raw, err := ToTCP(map[string]any{}, rules)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "00000042" {
		t.Errorf("expected '00000042', got %q", string(raw))
	}
}

func TestToTCP_Truncate(t *testing.T) {
	rules := []spec.FieldRule{{Name: "F", Length: 4, Value: "toolongvalue"}}
	_, raw, err := ToTCP(map[string]any{}, rules)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "tool" {
		t.Errorf("expected truncated 'tool', got %q", string(raw))
	}
}

func TestToTCP_MissingField(t *testing.T) {
	rules := []spec.FieldRule{{Name: "F", Length: 8, Value: "{{request.body.missing}}"}}
	_, _, err := ToTCP(map[string]any{}, rules)
	if err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestToHTTP_ParseOffsets(t *testing.T) {
	// 8+15+3 = 26 bytes
	raw := cbs.BackendResponse("OK      000000012345.67THB")
	fields := []spec.ResponseField{
		{Name: "ResponseCode", Length: 8},
		{Name: "FIELD_A", Length: 15},
		{Name: "FIELD_B", Length: 3},
	}
	mapping := map[string]string{
		"balance":  "{{cbs.FIELD_A}}",
		"currency": "{{cbs.FIELD_B}}",
	}
	rt := spec.ResponseTransform{Fields: fields, Mapping: mapping}

	result, err := ToHTTP(raw, rt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["balance"] != "000000012345.67" {
		t.Errorf("expected balance '000000012345.67', got %q", result["balance"])
	}
	if result["currency"] != "THB" {
		t.Errorf("expected currency 'THB', got %q", result["currency"])
	}
}

func TestToHTTP_TrimPadding(t *testing.T) {
	raw := cbs.BackendResponse("OK      ")
	fields := []spec.ResponseField{{Name: "ResponseCode", Length: 8}}
	rt := spec.ResponseTransform{Fields: fields, Mapping: map[string]string{"rc": "{{cbs.ResponseCode}}"}}

	result, err := ToHTTP(raw, rt)
	if err != nil {
		t.Fatal(err)
	}
	if result["rc"] != "OK" {
		t.Errorf("expected trimmed 'OK', got %q", result["rc"])
	}
}

func TestToHTTP_ShortResponse(t *testing.T) {
	raw := cbs.BackendResponse("OK") // too short
	fields := []spec.ResponseField{{Name: "ResponseCode", Length: 8}}
	rt := spec.ResponseTransform{Fields: fields}

	_, err := ToHTTP(raw, rt)
	if err == nil {
		t.Fatal("expected error for short response")
	}
}

func TestParseResponseFields(t *testing.T) {
	raw := cbs.BackendResponse("OK      ")
	fields := []spec.ResponseField{{Name: "ResponseCode", Length: 8}}
	parsed, err := ParseResponseFields(raw, fields)
	if err != nil {
		t.Fatal(err)
	}
	if parsed["ResponseCode"] != "OK" {
		t.Errorf("expected 'OK', got %q", parsed["ResponseCode"])
	}
}
