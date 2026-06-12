# Story: E6-S2 — Dashboard Detail Pages

**Epic:** E6 — Dashboard  
**Status:** Ready for Development  
**Depends on:** E6-S1  
**Blocks:** E7-S2

---

## Summary

Add the four detail pages under each spec version in the dashboard: endpoints list, endpoint detail (with full transform rules), systems table, and response codes table. All data comes from the in-memory `SpecRegistry` — no file reads.

---

## Acceptance Criteria

- [ ] `GET /dashboard/<version>/endpoints` → list of all endpoints (path, method, summary)
- [ ] `GET /dashboard/<version>/endpoints?path=/example/query&method=POST` → full endpoint detail
  - Request fields (name, type, required, description)
  - Request transform rules (backend field, length, value/template, align, pad)
  - Response transform fields (backend field, length) + mapping
  - Response codes for this endpoint (global merged with endpoint-specific)
- [ ] `GET /dashboard/<version>/systems` → systemCode → port → description table
- [ ] `GET /dashboard/<version>/codes` → full response code table (code, type, HTTP status, description, backend code)
- [ ] Non-existent version → `404` HTML page
- [ ] All data from `SpecRegistry` — no file I/O at request time

---

## Routes to Add (`internal/dashboard/handler.go`)

Extend `Routes()`:

```go
func (h *Handler) Routes(r chi.Router) {
    r.Get("/", h.home)
    r.Get("/{version}", h.overview)
    r.Get("/{version}/endpoints", h.endpointsList)
    r.Get("/{version}/endpoints", h.endpointDetail) // use query params path+method
    r.Get("/{version}/systems", h.systems)
    r.Get("/{version}/codes", h.codes)
}
```

> **Note:** `endpointsList` and `endpointDetail` share the same URL — distinguish by presence of `path` query param:

```go
func (h *Handler) endpointsList(w http.ResponseWriter, r *http.Request) {
    if r.URL.Query().Get("path") != "" {
        h.endpointDetail(w, r)
        return
    }
    // render list
}
```

---

## Handler Functions

### Endpoints List

```go
func (h *Handler) endpointsList(w http.ResponseWriter, r *http.Request) {
    version := chi.URLParam(r, "version")
    s, ok := h.registry[version]
    if !ok {
        http.Error(w, "version not found", http.StatusNotFound)
        return
    }
    // render endpoints list template with s.Endpoints
}
```

### Endpoint Detail

```go
func (h *Handler) endpointDetail(w http.ResponseWriter, r *http.Request) {
    version := chi.URLParam(r, "version")
    path := r.URL.Query().Get("path")
    method := r.URL.Query().Get("method")

    s, ok := h.registry[version]
    if !ok {
        http.Error(w, "version not found", http.StatusNotFound)
        return
    }

    var found *spec.Endpoint
    for i := range s.Endpoints {
        ep := &s.Endpoints[i]
        if ep.Path == path && strings.EqualFold(ep.Method, method) {
            found = ep
            break
        }
    }
    if found == nil {
        http.Error(w, "endpoint not found", http.StatusNotFound)
        return
    }
    // render endpoint detail template
}
```

---

## Templates

### `internal/dashboard/templates/endpoints-list.html`
```html
{{define "content"}}
<a href="/dashboard/{{.Version}}">← Overview</a>
<h2>Endpoints</h2>
<table>
  <thead><tr><th>Method</th><th>Path</th><th>Summary</th></tr></thead>
  <tbody>
  {{range .Endpoints}}
    <tr>
      <td>{{.Method}}</td>
      <td><a href="/dashboard/{{$.Version}}/endpoints?path={{.Path}}&method={{.Method}}">{{.Path}}</a></td>
      <td>{{.Summary}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{end}}
```

### `internal/dashboard/templates/endpoint-detail.html`
```html
{{define "content"}}
<a href="/dashboard/{{.Version}}/endpoints">← Endpoints</a>
<h2>{{.Endpoint.Method}} {{.Endpoint.Path}}</h2>
<p>{{.Endpoint.Description}}</p>

<h3>Request Fields</h3>
<table>
  <thead><tr><th>Field</th><th>Type</th><th>Required</th><th>Description</th></tr></thead>
  <tbody>
  {{range $name, $def := .Endpoint.Request.Fields}}
    <tr><td>{{$name}}</td><td>{{$def.Type}}</td><td>{{$def.Required}}</td><td>{{$def.Description}}</td></tr>
  {{end}}
  </tbody>
</table>

<h3>Request Transform (HTTP → TCP)</h3>
<table>
  <thead><tr><th>Backend Field</th><th>Length</th><th>Value / Template</th><th>Align</th><th>Pad</th></tr></thead>
  <tbody>
  {{range .Endpoint.Transform.Request}}
    <tr><td>{{.Name}}</td><td>{{.Length}}</td><td><code>{{.Value}}</code></td><td>{{.Align}}</td><td>"{{.Pad}}"</td></tr>
  {{end}}
  </tbody>
</table>

<h3>Response Transform (TCP → HTTP)</h3>
<h4>Backend Response Fields (by offset)</h4>
<table>
  <thead><tr><th>Backend Field</th><th>Length</th></tr></thead>
  <tbody>
  {{range .Endpoint.Transform.Response.Fields}}
    <tr><td>{{.Name}}</td><td>{{.Length}}</td></tr>
  {{end}}
  </tbody>
</table>
<h4>Mapping</h4>
<table>
  <thead><tr><th>HTTP Response Key</th><th>Source Template</th></tr></thead>
  <tbody>
  {{range $key, $tmpl := .Endpoint.Transform.Response.Mapping}}
    <tr><td>{{$key}}</td><td><code>{{$tmpl}}</code></td></tr>
  {{end}}
  </tbody>
</table>

<h3>Response Codes</h3>
<table>
  <thead><tr><th>Code</th><th>Description</th></tr></thead>
  <tbody>
  {{range .Endpoint.ResponseCodes}}
    <tr><td>{{.Code}}</td><td>{{.Description}}</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}
```

### `internal/dashboard/templates/systems.html`
```html
{{define "content"}}
<a href="/dashboard/{{.Version}}">← Overview</a>
<h2>Systems Registry</h2>
<table>
  <thead><tr><th>systemCode</th><th>Port</th><th>Description</th></tr></thead>
  <tbody>
  {{range $code, $def := .Systems}}
    <tr><td>{{$code}}</td><td>{{$def.Port}}</td><td>{{$def.Description}}</td></tr>
  {{end}}
  </tbody>
</table>
{{end}}
```

### `internal/dashboard/templates/codes.html`
```html
{{define "content"}}
<a href="/dashboard/{{.Version}}">← Overview</a>
<h2>Response Code Table</h2>
<table>
  <thead><tr><th>Engine Code</th><th>Type</th><th>HTTP Status</th><th>Description</th><th>Backend Code</th></tr></thead>
  <tbody>
  {{range .Codes}}
    <tr>
      <td>{{.EngineCode}}</td>
      <td>{{.Type}}</td>
      <td>{{.HTTPStatus}}</td>
      <td>{{.Description}}</td>
      <td>{{.CBSCode}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{end}}
```

---

## Verification

```bash
go build ./...
go run ./cmd/engine

# open in browser:
# http://localhost:8080/dashboard/v1/endpoints
# http://localhost:8080/dashboard/v1/endpoints?path=/example/query&method=POST
# http://localhost:8080/dashboard/v1/systems
# http://localhost:8080/dashboard/v1/codes
# http://localhost:8080/dashboard/doesnotexist/endpoints  → 404
```
