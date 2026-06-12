package handler

import (
	"net/http"

	"engine-poc/internal/cbs"
	"engine-poc/internal/codemap"
	"engine-poc/internal/spec"
	"engine-poc/internal/transformer"
)

type Handler struct {
	spec   *spec.Spec
	client cbs.Client
	mapper *codemap.Mapper
}

func New(s *spec.Spec, client cbs.Client, mapper *codemap.Mapper) *Handler {
	return &Handler{spec: s, client: client, mapper: mapper}
}

// Factory returns a HandlerFactory compatible with server.New().
func (h *Handler) Factory() func(version string, endpoint spec.Endpoint) http.HandlerFunc {
	return func(version string, endpoint spec.Endpoint) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			h.handle(w, r, endpoint)
		}
	}
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request, endpoint spec.Endpoint) {
	body, engineErr := ParseAndValidate(r, endpoint.Request)
	if engineErr != nil {
		writeError(w, engineErr)
		return
	}

	systemCode, _ := body["systemCode"].(string)
	port, engineErr := ResolvePort(systemCode, h.spec.Systems)
	if engineErr != nil {
		writeError(w, engineErr)
		return
	}

	cbsMsg, _, err := transformer.ToTCP(body, endpoint.Transform.Request)
	if err != nil {
		writeError(w, ErrInternal)
		return
	}

	rawResp, err := h.client.Send(port, cbsMsg)
	if err != nil {
		writeError(w, ErrInternal)
		return
	}

	parsed, err := transformer.ParseResponseFields(rawResp, endpoint.Transform.Response.Fields)
	if err != nil {
		writeError(w, ErrInternal)
		return
	}
	engineCode := h.mapper.Map(parsed["ResponseCode"])

	httpData, err := transformer.ToHTTP(rawResp, endpoint.Transform.Response)
	if err != nil {
		writeError(w, ErrInternal)
		return
	}

	writeSuccess(w, engineCode, httpData)
}
