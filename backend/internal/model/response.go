package model

import "net/http"

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func SuccessResponse(data interface{}) (int, APIResponse) {
	return http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	}
}

func ErrorResponse(code, message, detail string) (int, APIResponse) {
	return http.StatusOK, APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Detail:  detail,
		},
	}
}

type ExtractResultResponse struct {
	SQL         string        `json:"sql"`
	Bindings    []interface{} `json:"bindings"`
	RestoredSQL string        `json:"restoredSql"`
	LogType     string        `json:"logType"`
}

type FormatResultResponse struct {
	FormattedSQL string `json:"formattedSql"`
}

type MarkdownReportResponse struct {
	Markdown string `json:"markdown"`
}
