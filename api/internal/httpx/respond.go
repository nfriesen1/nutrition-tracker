package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// errorEnvelope is the single shape every non-2xx response takes.
// See docs/03-api-contract.md.
type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// writeJSON serialises v as the response body.
//
// It marshals before writing the header, so a marshalling failure can still become a 500
// rather than appending garbage to an already-committed 200.
func writeJSON(w http.ResponseWriter, r *http.Request, status int, v any) {
	body, err := json.Marshal(v)
	if err != nil {
		slog.ErrorContext(r.Context(), "marshalling response failed", "error", err)
		http.Error(w, `{"error":{"code":"internal_error","message":"Something went wrong."}}`,
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		// The client hung up mid-write. Nothing to do but note it — the status is sent.
		slog.DebugContext(r.Context(), "writing response body failed", "error", err)
	}
}

// writeNoContent sends a 204.
func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// writeError turns any error into the error envelope.
//
// An unrecognised error is a 500 whose real message is logged, never returned: an
// internal error string can carry a query fragment or a hostname, and the client has no
// use for it either way.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := statusFor(err)

	var apiErr *APIError
	message := "Something went wrong."
	var details map[string]string

	if errors.As(err, &apiErr) {
		message = apiErr.Message
		details = apiErr.Details
	}

	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed",
			"error", err, "method", r.Method, "path", r.URL.Path)
		message = "Something went wrong."
		details = nil
	}

	writeJSON(w, r, status, errorEnvelope{Error: errorBody{
		Code:    code,
		Message: message,
		Details: details,
	}})
}
