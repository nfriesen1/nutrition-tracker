package httpx

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nfriesen1/nutrition-tracker/api/internal/config"
)

func TestMain(m *testing.M) {
	// The recovery test logs a stack trace on purpose. Discard it so a passing run is
	// quiet and a real failure stands out.
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	os.Exit(m.Run())
}

func newTestServer(env string) http.Handler {
	return NewServer(config.Config{Env: env, Port: "8080"}, "test-version").Routes()
}

// decodeError reads the error envelope, failing the test if the body is not one.
func decodeError(t *testing.T, body io.Reader) errorEnvelope {
	t.Helper()
	var env errorEnvelope
	if err := json.NewDecoder(body).Decode(&env); err != nil {
		t.Fatalf("response is not the error envelope: %v", err)
	}
	return env
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	newTestServer(config.EnvDevelopment).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var body healthResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decoding body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	// Proves the -ldflags version reaches the response, so it is possible to tell what
	// is deployed.
	if body.Version != "test-version" {
		t.Errorf("version = %q, want test-version", body.Version)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
}

func TestHealthzSetsRequestID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	newTestServer(config.EnvDevelopment).ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-Id") == "" {
		t.Error("X-Request-Id is empty; correlating a log line to a report needs it")
	}
}

func TestUnknownRouteReturnsEnvelope(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	newTestServer(config.EnvDevelopment).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	// net/http's default 404 is plain text. A client that has to parse two error shapes
	// will eventually parse the wrong one.
	env := decodeError(t, rec.Body)
	if env.Error.Code != "not_found" {
		t.Errorf("code = %q, want not_found", env.Error.Code)
	}
}

func TestWrongMethodIsRejected(t *testing.T) {
	// Go 1.22 method-aware patterns mean the mux answers this, not the handler.
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()
	newTestServer(config.EnvDevelopment).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestRecoveryTurnsPanicInto500(t *testing.T) {
	boom := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})
	handler := chain(boom, Recovery(), RequestLogger())

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()

	// The point of the middleware: this call returns instead of taking the process down.
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	env := decodeError(t, rec.Body)
	if env.Error.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", env.Error.Code)
	}
	// The panic value must not reach the client — it can carry internal detail.
	if env.Error.Message != "Something went wrong." {
		t.Errorf("message = %q leaks internal detail", env.Error.Message)
	}
}

func TestRecoveryServesLaterRequests(t *testing.T) {
	var shouldPanic bool
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if shouldPanic {
			panic("boom")
		}
		writeJSON(w, r, http.StatusOK, map[string]string{"ok": "yes"})
	})
	handler := chain(h, Recovery(), RequestLogger())

	shouldPanic = true
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/x", nil))

	// The whole reason recovery exists: one bad request must not end the process.
	shouldPanic = false
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status after a panic = %d, want 200", rec.Code)
	}
}

func TestErrorStatusMapping(t *testing.T) {
	// The table in docs/03-api-contract.md.
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{Errorf(ErrBadRequest, "x"), http.StatusBadRequest, "bad_request"},
		{Errorf(ErrUnauthenticated, "x"), http.StatusUnauthorized, "unauthenticated"},
		{Errorf(ErrNotFound, "x"), http.StatusNotFound, "not_found"},
		{Errorf(ErrConflict, "x"), http.StatusConflict, "conflict"},
		{ValidationError(map[string]string{"quantity": "must be > 0"}), http.StatusUnprocessableEntity, "validation_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.wantCode, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			env := decodeError(t, rec.Body)
			if env.Error.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", env.Error.Code, tt.wantCode)
			}
		})
	}
}

func TestValidationErrorCarriesFieldDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil),
		ValidationError(map[string]string{"quantity": "must be greater than 0"}))

	env := decodeError(t, rec.Body)
	if env.Error.Details["quantity"] != "must be greater than 0" {
		t.Errorf("details = %v; a form cannot highlight the right input without them", env.Error.Details)
	}
}

func TestInternalErrorDetailIsNotLeaked(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil),
		Errorf(nil, "connection to db-prod-7.internal refused"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	env := decodeError(t, rec.Body)
	// An internal message can carry a hostname or a query fragment, and the client has
	// no use for it either way.
	if env.Error.Message != "Something went wrong." {
		t.Errorf("message = %q leaks internal detail", env.Error.Message)
	}
}

func TestDevCORSOnlyInDevelopment(t *testing.T) {
	const origin = "http://localhost:8081"

	t.Run("development allows the Expo dev server", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		newTestServer(config.EnvDevelopment).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, origin)
		}
	})

	t.Run("production sends no CORS header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		newTestServer(config.EnvProduction).ServeHTTP(rec, req)

		// Production web is same-origin behind Netlify's proxy and native sends no
		// Origin at all, so there is nothing to permit. See docs/06-deployment.md.
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("Access-Control-Allow-Origin = %q in production, want empty", got)
		}
	})

	t.Run("unlisted origin is not reflected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		rec := httptest.NewRecorder()
		newTestServer(config.EnvDevelopment).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("reflected an unlisted origin: %q", got)
		}
	})
}
