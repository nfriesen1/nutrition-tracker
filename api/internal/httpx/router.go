package httpx

import (
	"net/http"

	"github.com/nfriesen1/nutrition-tracker/api/internal/config"
)

// Server holds the dependencies every handler needs.
//
// It is constructed once in cmd/api/main.go. Handlers are methods on it rather than
// closures over package-level state, which is what makes them testable against a
// hand-built Server with no running process. The store lands here in T-07.
type Server struct {
	cfg     config.Config
	version string
}

func NewServer(cfg config.Config, version string) *Server {
	return &Server{cfg: cfg, version: version}
}

// Routes returns the fully-wrapped handler for the whole service.
//
// This is the one place every endpoint is listed. Go 1.22's method-aware patterns mean
// the method is part of the pattern, so the mux itself answers 405 for a known path with
// the wrong method — no per-handler method check.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Unauthenticated.
	mux.HandleFunc("GET /healthz", s.handleHealth)

	// Authenticated. T-08 wraps this group with the JWT middleware; until then there is
	// nothing here to protect.
	//
	//   mux.HandleFunc("GET /api/v1/me", ...)               T-10
	//   mux.HandleFunc("PATCH /api/v1/me", ...)             T-10
	//   mux.HandleFunc("GET /api/v1/foods", ...)            T-11
	//   mux.HandleFunc("POST /api/v1/foods", ...)           T-11
	//   mux.HandleFunc("GET /api/v1/foods/{id}", ...)       T-11
	//   mux.HandleFunc("GET /api/v1/diary", ...)            T-12
	//   mux.HandleFunc("POST /api/v1/diary/entries", ...)   T-12
	//   mux.HandleFunc("PATCH /api/v1/diary/entries/{id}",  T-12
	//   mux.HandleFunc("DELETE /api/v1/diary/entries/{id}", T-12

	middleware := []Middleware{
		// Outermost: a panic inside the logger is still caught.
		Recovery(),
		RequestLogger(),
		// Rewrites the mux's own 404 and 405 into the error envelope. Deliberately not a
		// catch-all `/` route — that matches `POST /healthz` and turns a 405 into a 404,
		// hiding that the path exists and only the method is wrong.
		JSONNotFound(),
	}

	if s.cfg.IsDevelopment() {
		middleware = append(middleware, DevCORS([]string{
			"http://localhost:8081",
			"http://localhost:19006",
		}))
	}

	return chain(mux, middleware...)
}
