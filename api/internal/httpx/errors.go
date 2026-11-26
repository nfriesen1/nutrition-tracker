package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors. Handlers return these; only writeError turns them into a status.
//
// A handler that writes a status code directly is a bug: the mapping below is the single
// place the API's error contract lives, and duplicating it is how a 404 starts coming
// back as a 403 from one endpoint. See docs/03-api-contract.md.
var (
	ErrBadRequest       = errors.New("bad request")
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrNotFound         = errors.New("not found")
	ErrMethodNotAllowed = errors.New("method not allowed")
	ErrConflict         = errors.New("conflict")
	ErrValidation       = errors.New("validation failed")
)

// APIError carries a sentinel plus the message and field details that reach the client.
type APIError struct {
	// Sentinel is one of the errors above; it selects the status code and the code string.
	Sentinel error
	// Message is one human-readable sentence, safe to show a user.
	Message string
	// Details maps field name to problem. Only meaningful for ErrValidation.
	Details map[string]string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%v: %s", e.Sentinel, e.Message)
}

// Unwrap lets errors.Is(err, ErrNotFound) work on a wrapped APIError.
func (e *APIError) Unwrap() error { return e.Sentinel }

// Errorf builds an APIError with a formatted message.
func Errorf(sentinel error, format string, args ...any) *APIError {
	return &APIError{Sentinel: sentinel, Message: fmt.Sprintf(format, args...)}
}

// ValidationError builds a 422 carrying per-field detail.
func ValidationError(details map[string]string) *APIError {
	return &APIError{
		Sentinel: ErrValidation,
		Message:  "The request body failed validation.",
		Details:  details,
	}
}

// statusFor maps a sentinel to its HTTP status and stable machine-readable code.
//
// The table in docs/03-api-contract.md is the authority; this is its implementation.
func statusFor(err error) (status int, code string) {
	switch {
	case errors.Is(err, ErrBadRequest):
		return http.StatusBadRequest, "bad_request"
	case errors.Is(err, ErrUnauthenticated):
		return http.StatusUnauthorized, "unauthenticated"
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, ErrMethodNotAllowed):
		return http.StatusMethodNotAllowed, "method_not_allowed"
	case errors.Is(err, ErrConflict):
		return http.StatusConflict, "conflict"
	case errors.Is(err, ErrValidation):
		return http.StatusUnprocessableEntity, "validation_failed"
	default:
		return http.StatusInternalServerError, "internal_error"
	}
}
