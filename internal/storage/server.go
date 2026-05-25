package storage

import (
	"encoding/json"
	"net/http"
)

// Server is the HTTP server skeleton for a storage tier instance.
type Server struct {
	tierID int
	mux    *http.ServeMux
}

// NewServer builds a storage tier server with route stubs only.
func NewServer(tierID int) *Server {
	s := &Server{tierID: tierID, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	// TODO: GET /v1/teams/{teamId}/folders — discover via global tier + local DB
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ok",
		"tierId": s.tierID,
	})
}
