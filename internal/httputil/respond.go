package httputil

import (
	"encoding/json"
	"net/http"
)

// WriteJSON encodes v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Unauthorized writes a 401 JSON error body.
func Unauthorized(w http.ResponseWriter) {
	WriteJSON(w, http.StatusUnauthorized, map[string]string{
		"error": "unauthorized",
	})
}

// Forbidden writes a 403 JSON error body.
func Forbidden(w http.ResponseWriter) {
	WriteJSON(w, http.StatusForbidden, map[string]string{
		"error": "forbidden",
	})
}

// NotFound writes a 404 JSON error body.
func NotFound(w http.ResponseWriter) {
	WriteJSON(w, http.StatusNotFound, map[string]string{
		"error": "not found",
	})
}
