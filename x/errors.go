package x

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ErrorResponse struct {
	Error *ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int                    `json:"code"`
	Status  string                 `json:"status"`
	Reason  string                 `json:"reason,omitempty"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
	ID      string                 `json:"id,omitempty"`
}

func (e *ErrorDetail) Error() string {
	return fmt.Sprintf("[%d %s] %s: %s", e.Code, e.Status, e.Reason, e.Message)
}

func ErrInternalServerError(reason string, message string, details map[string]interface{}) *ErrorDetail {
	return &ErrorDetail{
		Code:    http.StatusInternalServerError,
		Status:  http.StatusText(http.StatusInternalServerError),
		Reason:  reason,
		Message: message,
		Details: details,
		ID:      "session_token_signing_error",
	}
}

func WriteJSONError(w http.ResponseWriter, r *http.Request, err *ErrorDetail) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Code)
	_ = json.NewEncoder(w).Encode(&ErrorResponse{Error: err})
}
