package codemap

import "net/http"

var standardTable = map[string]EngineCode{
	// Success
	"OK": {Code: 10000, HTTPStatus: http.StatusOK, Message: "Success"},

	// Business errors — BACKEND_ERR series
	"BACKEND_ERR_001": {Code: 21001, HTTPStatus: http.StatusOK, Message: "Backend internal error"},
	"BACKEND_ERR_002": {Code: 21002, HTTPStatus: http.StatusOK, Message: "Route name not found"},
	"BACKEND_ERR_003": {Code: 21003, HTTPStatus: http.StatusOK, Message: "Transaction timeout"},
	"BACKEND_ERR_004": {Code: 21004, HTTPStatus: http.StatusOK, Message: "Program exception error"},
	"BACKEND_ERR_005": {Code: 21005, HTTPStatus: http.StatusOK, Message: "Operation reversal rejected"},
	"BACKEND_ERR_006": {Code: 21006, HTTPStatus: http.StatusOK, Message: "System is not ready to process"},
	"BACKEND_ERR_007": {Code: 21007, HTTPStatus: http.StatusOK, Message: "Duplicate application key"},
	"BACKEND_ERR_008": {Code: 21008, HTTPStatus: http.StatusOK, Message: "Unknown data format"},
	"BACKEND_ERR_009": {Code: 21009, HTTPStatus: http.StatusOK, Message: "Conversion not found"},
	"BACKEND_ERR_010": {Code: 21010, HTTPStatus: http.StatusOK, Message: "Host unavailable"},
	"BACKEND_ERR_011": {Code: 21011, HTTPStatus: http.StatusOK, Message: "System watchdog failed to start job"},
	"BACKEND_ERR_012": {Code: 21012, HTTPStatus: http.StatusOK, Message: "Invalid source id"},
	"BACKEND_ERR_013": {Code: 21013, HTTPStatus: http.StatusOK, Message: "Unknown error"},

	// Business errors — SVC_ERR series
	"SVC_ERR_001": {Code: 22001, HTTPStatus: http.StatusOK, Message: "No records found"},

	// Default: unmapped backend code
	"__default__": {Code: 20000, HTTPStatus: http.StatusOK, Message: "Business error"},
}
