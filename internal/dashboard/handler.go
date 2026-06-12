package dashboard

import (
	_ "embed"
	"html/template"
	"net/http"
	"sort"
	"strings"
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

//go:embed templates/endpoints-list.html
var endpointsListHTML string

//go:embed templates/endpoint-detail.html
var endpointDetailHTML string

//go:embed templates/systems.html
var systemsHTML string

//go:embed templates/codes.html
var codesHTML string

type Handler struct {
	registry spec.SpecRegistry
	loadedAt time.Time
}

func New(registry spec.SpecRegistry) *Handler {
	return &Handler{registry: registry, loadedAt: time.Now().UTC()}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.home)
	r.Get("/{version}", h.overview)
	r.Get("/{version}/endpoints", h.endpointsList)
	r.Get("/{version}/systems", h.systems)
	r.Get("/{version}/codes", h.codes)
}

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

func (h *Handler) endpointsList(w http.ResponseWriter, r *http.Request) {
	// Dispatch to detail if path query param present
	if r.URL.Query().Get("path") != "" {
		h.endpointDetail(w, r)
		return
	}

	version := chi.URLParam(r, "version")
	s, ok := h.registry[version]
	if !ok {
		http.Error(w, "version not found", http.StatusNotFound)
		return
	}

	tmpl := template.Must(template.New("layout").Parse(layoutHTML + endpointsListHTML))
	w.Header().Set("Content-Type", "text/html")
	tmpl.ExecuteTemplate(w, "layout", map[string]any{
		"Title":     s.Name + " — Endpoints",
		"Version":   version,
		"Endpoints": s.Endpoints,
	})
}

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

	tmpl := template.Must(template.New("layout").Parse(layoutHTML + endpointDetailHTML))
	w.Header().Set("Content-Type", "text/html")
	tmpl.ExecuteTemplate(w, "layout", map[string]any{
		"Title":    method + " " + path,
		"Version":  version,
		"Endpoint": found,
	})
}

func (h *Handler) systems(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	s, ok := h.registry[version]
	if !ok {
		http.Error(w, "version not found", http.StatusNotFound)
		return
	}

	tmpl := template.Must(template.New("layout").Parse(layoutHTML + systemsHTML))
	w.Header().Set("Content-Type", "text/html")
	tmpl.ExecuteTemplate(w, "layout", map[string]any{
		"Title":   s.Name + " — Systems",
		"Version": version,
		"Systems": s.Systems,
	})
}

func (h *Handler) codes(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	s, ok := h.registry[version]
	if !ok {
		http.Error(w, "version not found", http.StatusNotFound)
		return
	}

	// Build sorted code list
	type codeRow struct {
		EngineCode  int
		Type        string
		HTTPStatus  int
		Description string
		CBSCode     string
	}
	var rows []codeRow
	for _, def := range s.ResponseCodes {
		rows = append(rows, codeRow{
			EngineCode:  def.EngineCode,
			Type:        def.Type,
			HTTPStatus:  def.HTTPStatus,
			Description: def.Description,
			CBSCode:     def.CBSCode,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].EngineCode < rows[j].EngineCode })

	tmpl := template.Must(template.New("layout").Parse(layoutHTML + codesHTML))
	w.Header().Set("Content-Type", "text/html")
	tmpl.ExecuteTemplate(w, "layout", map[string]any{
		"Title":   s.Name + " — Response Codes",
		"Version": version,
		"Codes":   rows,
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
