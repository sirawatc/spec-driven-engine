package handler

import "net/http"

// EngineError carries the 5-digit engine code and HTTP status for a pipeline failure.
type EngineError struct {
	Code       int
	HTTPStatus int
	Message    string
}

func (e *EngineError) Error() string { return e.Message }

var (
	ErrParseRequest      = &EngineError{Code: 31000, HTTPStatus: http.StatusBadRequest, Message: "Input parse error"}
	ErrSystemCodeNotFound = &EngineError{Code: 21002, HTTPStatus: http.StatusOK, Message: "Route name not found"}
	ErrInternal          = &EngineError{Code: 40000, HTTPStatus: http.StatusInternalServerError, Message: "Internal server error"}
)

func ErrValidation(field string) *EngineError {
	return &EngineError{
		Code:       32000,
		HTTPStatus: http.StatusBadRequest,
		Message:    "Input validation error: missing required field: " + field,
	}
}
