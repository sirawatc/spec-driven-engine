package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"engine-poc/internal/spec"
)

// ParseAndValidate reads the request body, parses JSON, and validates
// required fields against schema. Returns parsed body map or EngineError.
func ParseAndValidate(r *http.Request, schema spec.RequestSchema) (map[string]any, *EngineError) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, ErrParseRequest
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, ErrParseRequest
	}

	if _, ok := data["systemCode"]; !ok {
		return nil, ErrValidation("systemCode")
	}

	for name, def := range schema.Fields {
		if def.Required {
			if _, ok := data[name]; !ok {
				return nil, ErrValidation(name)
			}
		}
	}

	return data, nil
}
