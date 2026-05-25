package global

import (
	"encoding/json"
	"net/http"
)

// Server is the HTTP server skeleton for the global (identity) tier.
type Server struct {
	mux *http.ServeMux
}

// NewServer builds a global tier server with route stubs only.
func NewServer() *Server {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	// TODO: GET /v1/discover — JWT validation + identity DB lookup
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
