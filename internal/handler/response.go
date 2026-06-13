package handler

import (
	"encoding/json"
	"net/http"

	"engine-poc/internal/codemap"
)

type apiResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func writeSuccess(w http.ResponseWriter, code codemap.EngineCode, data map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code.HTTPStatus)
	json.NewEncoder(w).Encode(apiResponse{
		Code:    code.Code,
		Message: code.Message,
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, err *EngineError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)
	json.NewEncoder(w).Encode(apiResponse{
		Code:    err.Code,
		Message: err.Message,
		Data:    nil,
	})
}
