package httpx

import "net/http"

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// handleHealth answers GET /healthz.
//
// No auth and, deliberately, no database round trip. Fly restarts a machine whose health
// check fails, and restarting the API does not fix Postgres — a check that failed on a
// database blip would turn a brief database problem into a restart loop. Database
// reachability is a monitoring concern, not a liveness one. See docs/03-api-contract.md.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, r, http.StatusOK, healthResponse{Status: "ok", Version: s.version})
}
