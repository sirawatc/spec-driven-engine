package codemap

import (
	"engine-poc/internal/spec"
)

// EngineCode is the resolved 5-digit response code with its HTTP status.
type EngineCode struct {
	Code       int
	HTTPStatus int
	Message    string
}

// Mapper translates backend ResponseCode → EngineCode.
type Mapper struct {
	table map[string]EngineCode
}

// New creates a Mapper seeded with the standard table.
func New() *Mapper {
	table := make(map[string]EngineCode, len(standardTable))
	for k, v := range standardTable {
		table[k] = v
	}
	return &Mapper{table: table}
}

// NewWithOverrides creates a Mapper seeded with the standard table and merged overrides.
func NewWithOverrides(overrides map[string]spec.ResponseCodeDef) *Mapper {
	table := make(map[string]EngineCode, len(standardTable)+len(overrides))
	for k, v := range standardTable {
		table[k] = v
	}
	for _, def := range overrides {
		if def.CBSCode == "" {
			continue
		}
		table[def.CBSCode] = EngineCode{
			Code:       def.EngineCode,
			HTTPStatus: def.HTTPStatus,
			Message:    def.Description,
		}
	}
	return &Mapper{table: table}
}

// Map returns the EngineCode for the given backend response code.
// Unknown codes return the default business error (20000).
func (m *Mapper) Map(cbsCode string) EngineCode {
	if code, ok := m.table[cbsCode]; ok {
		return code
	}
	return m.table["__default__"]
}
