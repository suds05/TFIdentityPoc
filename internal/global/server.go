package global

import (
	"errors"
	"log"
	"net/http"

	"tfidentitypoc/internal/auth"
	"tfidentitypoc/internal/httputil"
	"tfidentitypoc/internal/identity"
)

// Server is the HTTP server for the global (identity) tier.
type Server struct {
	mux       *http.ServeMux
	jwtSecret string
	store     *identity.Store
}

// NewServer builds a global tier server.
func NewServer(jwtSecret string, store *identity.Store) *Server {
	s := &Server{
		mux:       http.NewServeMux(),
		jwtSecret: jwtSecret,
		store:     store,
	}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /v1/discover", s.handleDiscover)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Discovery API.
func (s *Server) handleDiscover(w http.ResponseWriter, r *http.Request) {
	// Validate the token.
	token, err := auth.BearerToken(r.Header.Get("Authorization"))
	if err != nil {
		httputil.Unauthorized(w)
		return
	}
	claims, err := auth.ParseUserJWT(s.jwtSecret, token)
	if err != nil {
		httputil.Unauthorized(w)
		return
	}

	// Read discovery data from DB for this userid.
	result, err := s.store.Discover(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			httputil.Unauthorized(w)
			return
		}
		log.Printf("discover: user=%s: %v", claims.UserID, err)
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal error",
		})
		return
	}
	httputil.WriteJSON(w, http.StatusOK, result)
}
