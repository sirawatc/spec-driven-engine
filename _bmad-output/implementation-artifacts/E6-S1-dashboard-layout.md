# Story: E6-S1 — Dashboard Server and Layout

**Epic:** E6 — Dashboard  
**Status:** Ready for Development  
**Depends on:** E1-S3  
**Blocks:** E6-S2

---

## Summary

Serve a read-only HTML dashboard from embedded Go templates. The root `/dashboard` page lists all loaded spec versions. Each version links to its own overview page. No external CSS/JS — self-contained HTML only.

---

## Acceptance Criteria

- [ ] `GET /dashboard` → HTML page listing all spec versions (version key, spec name, version field, description)
- [ ] Each version links to `/dashboard/<version>`
- [ ] `GET /dashboard/<version>` → spec overview: name, version, description, load timestamp, nav links to Endpoints / Systems / Codes
- [ ] Templates embedded via `//go:embed`
- [ ] Renders with one version and with two versions
- [ ] No external CSS/JS CDN dependencies

---

## Dashboard Handler (`internal/dashboard/handler.go`)

```go
package dashboard

import (
    _ "embed"
    "html/template"
    "net/http"
    "sort"
    "time"

    "github.com/go-chi/chi/v5"

    "engine-poc/internal/spec"
)

//go:embed templates/layout.html
var layoutHTML string

//go:embed templates/home.html
var homeHTML string

//go:embed templates/overview.html
var overviewHTML string

type Handler struct {
    registry  spec.SpecRegistry
    loadedAt  time.Time
}

func New(registry spec.SpecRegistry) *Handler {
    return &Handler{registry: registry, loadedAt: time.Now().UTC()}
}

func (h *Handler) Routes(r chi.Router) {
    r.Get("/", h.home)
    r.Get("/{version}", h.overview)
}

// home lists all spec versions.
func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
    type versionRow struct {
        Key         string
        Name        string
        Version     string
        Description string
    }

    var rows []versionRow
    for _, v := range sortedKeys(h.registry) {
        s := h.registry[v]
        rows = append(rows, versionRow{Key: v, Name: s.Name, Version: s.Version, Description: s.Description})
    }

    tmpl := template.Must(template.New("layout").Parse(layoutHTML + homeHTML))
    w.Header().Set("Content-Type", "text/html")
    tmpl.ExecuteTemplate(w, "layout", map[string]any{
        "Title": "Service Engine",
        "Rows":  rows,
    })
}

// overview shows a single spec version's details.
func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
    version := chi.URLParam(r, "version")
    s, ok := h.registry[version]
    if !ok {
        http.Error(w, "spec version not found: "+version, http.StatusNotFound)
        return
    }

    tmpl := template.Must(template.New("layout").Parse(layoutHTML + overviewHTML))
    w.Header().Set("Content-Type", "text/html")
    tmpl.ExecuteTemplate(w, "layout", map[string]any{
        "Title":    s.Name + " — " + version,
        "Spec":     s,
        "Version":  version,
        "LoadedAt": h.loadedAt.Format(time.RFC3339),
    })
}

func sortedKeys(r spec.SpecRegistry) []string {
    keys := make([]string, 0, len(r))
    for k := range r {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    return keys
}
```

---

## Templates

### `internal/dashboard/templates/layout.html`
```html
{{define "layout"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>{{.Title}}</title>
  <style>
    body { font-family: monospace; max-width: 960px; margin: 40px auto; padding: 0 20px; }
    h1 { border-bottom: 2px solid #333; padding-bottom: 8px; }
    table { width: 100%; border-collapse: collapse; margin-top: 16px; }
    th, td { text-align: left; padding: 8px 12px; border-bottom: 1px solid #ddd; }
    th { background: #f5f5f5; }
    a { color: #0055cc; }
    nav a { margin-right: 16px; }
  </style>
</head>
<body>
  <a href="/dashboard">← All Versions</a>
  <h1>{{.Title}}</h1>
  {{template "content" .}}
</body>
</html>
{{end}}
```

### `internal/dashboard/templates/home.html`
```html
{{define "content"}}
<p>Service Engine — spec versions loaded:</p>
<table>
  <thead><tr><th>Version</th><th>Name</th><th>Spec Version</th><th>Description</th></tr></thead>
  <tbody>
  {{range .Rows}}
    <tr>
      <td><a href="/dashboard/{{.Key}}">{{.Key}}</a></td>
      <td>{{.Name}}</td>
      <td>{{.Version}}</td>
      <td>{{.Description}}</td>
    </tr>
  {{end}}
  </tbody>
</table>
{{end}}
```

### `internal/dashboard/templates/overview.html`
```html
{{define "content"}}
<p><strong>Spec version:</strong> {{.Spec.Version}}</p>
<p><strong>Description:</strong> {{.Spec.Description}}</p>
<p><strong>Loaded at:</strong> {{.LoadedAt}}</p>
<nav>
  <a href="/dashboard/{{.Version}}/endpoints">Endpoints</a>
  <a href="/dashboard/{{.Version}}/systems">Systems</a>
  <a href="/dashboard/{{.Version}}/codes">Response Codes</a>
</nav>
{{end}}
```

---

## Register Dashboard Routes in Server

In `internal/server/server.go`, mount the dashboard (add after health check):

```go
dash := dashboard.New(registry)
r.Route("/dashboard", func(r chi.Router) {
    dash.Routes(r)
})
```

---

## Verification

```bash
go build ./...
go run ./cmd/engine
# open http://localhost:8080/dashboard in browser
# → version list page with v1 row linking to /dashboard/v1
# open http://localhost:8080/dashboard/v1
# → overview with name, version, description, nav links
# open http://localhost:8080/dashboard/doesnotexist
# → 404 plain text
```
