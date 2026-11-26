package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// Middleware is the standard decorator shape. Composing these by hand, rather than
// reaching for a framework, is most of what docs/00-overview.md means by using the
// standard library to learn Go's interfaces.
type Middleware func(http.Handler) http.Handler

// chain applies middleware so that the first argument is the OUTERMOST layer.
//
// chain(h, recovery, logging) means a request passes through recovery, then logging,
// then h — and a panic anywhere inside, logging included, is still caught.
func chain(h http.Handler, middleware ...Middleware) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

type ctxKeyRequestID struct{}

// RequestIDFrom returns the request id assigned by the logging middleware.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(ctxKeyRequestID{}).(string)
	return id
}

// statusRecorder captures the status code, which http.ResponseWriter does not expose.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	// A handler that writes without calling WriteHeader has implicitly sent a 200.
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Recovery turns a panic into a 500 with a logged stack trace, instead of taking the
// process down and dropping every in-flight request with it.
//
// This must be the outermost middleware.
func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					// http.ErrAbortHandler is the documented way to abort a response on
					// purpose; re-panicking preserves that meaning instead of logging it
					// as a crash.
					if v == http.ErrAbortHandler {
						panic(v)
					}
					slog.ErrorContext(r.Context(), "panic recovered",
						"panic", v,
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(debug.Stack()),
					)
					writeError(w, r, Errorf(nil, "Something went wrong."))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger assigns a request id and logs method, path, status, and duration on the
// way out.
func RequestLogger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			id := newRequestID()

			ctx := context.WithValue(r.Context(), ctxKeyRequestID{}, id)
			r = r.WithContext(ctx)
			w.Header().Set("X-Request-Id", id)

			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)

			if rec.status == 0 {
				rec.status = http.StatusOK
			}

			slog.InfoContext(ctx, "request",
				"request_id", id,
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"bytes", rec.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

func newRequestID() string {
	var b [8]byte
	// crypto/rand.Read never returns an error as of Go 1.24; it panics on failure.
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// envelopeRewriter swallows a plain-text 404/405 so JSONNotFound can replace it.
type envelopeRewriter struct {
	http.ResponseWriter
	intercepted int
}

func (w *envelopeRewriter) WriteHeader(status int) {
	if status == http.StatusNotFound || status == http.StatusMethodNotAllowed {
		// Hold the status back. Headers already set on the map — net/http's Allow
		// header on a 405, in particular — survive, because they are only flushed when
		// WriteHeader is eventually called.
		w.intercepted = status
		return
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *envelopeRewriter) Write(b []byte) (int, error) {
	if w.intercepted != 0 {
		// Discard net/http's "404 page not found" body; report success so the handler
		// does not treat this as a broken connection.
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

// JSONNotFound rewrites net/http's plain-text 404 and 405 into the error envelope.
//
// This is a middleware rather than a catch-all `/` route because a catch-all silently
// defeats the mux's own 405: `POST /healthz` would match `/` and come back as 404,
// hiding the fact that the path exists and only the method is wrong.
//
// A client that has to parse two different error shapes will eventually parse the wrong
// one, so every response from this service uses the envelope in docs/03-api-contract.md.
func JSONNotFound() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rw := &envelopeRewriter{ResponseWriter: w}
			next.ServeHTTP(rw, r)

			switch rw.intercepted {
			case http.StatusNotFound:
				writeError(w, r, Errorf(ErrNotFound, "No such endpoint."))
			case http.StatusMethodNotAllowed:
				writeError(w, r, Errorf(ErrMethodNotAllowed,
					"%s is not allowed on this endpoint.", r.Method))
			}
		})
	}
}

// DevCORS allows the Expo web dev server to call a locally-running API.
//
// Development only, and deliberately so. In production web is same-origin behind
// Netlify's /api proxy and native sends no Origin header at all, so there is nothing to
// permit — shipping a permissive header there would widen the surface for no benefit.
// See docs/06-deployment.md.
func DevCORS(allowedOrigins []string) Middleware {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods",
					strings.Join([]string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}, ", "))
				w.Header().Set("Access-Control-Max-Age", "300")
				// Origin varies the response, so a cache must not serve one origin's
				// response to another.
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
