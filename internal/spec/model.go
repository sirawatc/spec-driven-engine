package spec

// Spec is the root in-memory representation of one parsed spec file.
type Spec struct {
	Version       string                     `yaml:"version"`
	Name          string                     `yaml:"name"`
	Description   string                     `yaml:"description"`
	Systems       map[string]SystemDef       `yaml:"systems"`
	ResponseCodes map[string]ResponseCodeDef `yaml:"response_codes"`
	Endpoints     []Endpoint                 `yaml:"endpoints"`
}

type SystemDef struct {
	Port        int    `yaml:"port"`
	Description string `yaml:"description"`
}

type ResponseCodeDef struct {
	EngineCode  int    `yaml:"-"` // set from the map key during Load
	Type        string `yaml:"type"`
	HTTPStatus  int    `yaml:"http_status"`
	Description string `yaml:"description"`
	CBSCode     string `yaml:"backend_code"`
}

type Endpoint struct {
	Path          string         `yaml:"path"`
	Method        string         `yaml:"method"`
	Summary       string         `yaml:"summary"`
	Description   string         `yaml:"description"`
	Request       RequestSchema  `yaml:"request"`
	Transform     TransformSpec  `yaml:"transform"`
	ResponseCodes []EndpointCode `yaml:"response_codes"`
}

type RequestSchema struct {
	Fields map[string]FieldDef `yaml:"fields"`
}

type FieldDef struct {
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
}

type TransformSpec struct {
	Request  []FieldRule       `yaml:"request"`
	Response ResponseTransform `yaml:"response"`
}

// FieldRule defines one fixed-length field in the TCP request message.
type FieldRule struct {
	Name   string `yaml:"name"`
	Length int    `yaml:"length"`
	Value  string `yaml:"value"` // literal or {{request.body.<field>}} template
	Align  string `yaml:"align"` // "left" (default) or "right"
	Pad    string `yaml:"pad"`   // default " " (space)
}

type ResponseTransform struct {
	Fields  []ResponseField   `yaml:"fields"`
	Mapping map[string]string `yaml:"mapping"` // HTTP key → {{cbs.<field>}} template
}

// ResponseField defines one fixed-length field in the TCP response message.
type ResponseField struct {
	Name   string `yaml:"name"`
	Length int    `yaml:"length"`
}

type EndpointCode struct {
	Code        int    `yaml:"code"`
	Description string `yaml:"description"`
}
