//////////////////////////////////////////////////////////////
//
// Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License")
//
// Storage Tier Service implementation.
// Handles health check and list-folders API requests.
//

package storage

import (
	"errors"
	"log"
	"net/http"
	"slices"

	"tfidentitypoc/internal/auth"
	"tfidentitypoc/internal/globalclient"
	"tfidentitypoc/internal/httputil"
	"tfidentitypoc/internal/storagedb"
)

// Server is the HTTP server for a storage tier instance.
type Server struct {
	tierID    int
	mux       *http.ServeMux
	jwtSecret string
	global    *globalclient.Client
	store     *storagedb.Store
}

// NewServer builds a storage tier server.
func NewServer(tierID int, jwtSecret string, global *globalclient.Client, store *storagedb.Store) *Server {
	s := &Server{
		tierID:    tierID,
		mux:       http.NewServeMux(),
		jwtSecret: jwtSecret,
		global:    global,
		store:     store,
	}
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /v1/teams/{teamId}/folders", s.handleListFolders)
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"tierId": s.tierID,
	})
}

// Implement the list folders API.
// This API is used to list the folders in a team.
// It requires a valid JWT token.
// It calls the global tier's discover API to get the list of teams that the user is a member of.
// It then checks if the team is provisioned on this storage tier.
// If the team is not provisioned, it returns a 404 Not Found error.
// If the team is provisioned, it returns the list of folders in the team.
func (s *Server) handleListFolders(w http.ResponseWriter, r *http.Request) {

	// Validate arguments.
	teamID := r.PathValue("teamId")
	if teamID == "" {
		httputil.NotFound(w)
		return
	}

	// Validate the auth token.
	authHeader := r.Header.Get("Authorization")
	token, err := auth.BearerToken(authHeader)
	if err != nil {
		httputil.Unauthorized(w)
		return
	}
	if _, err := auth.ParseUserJWT(s.jwtSecret, token); err != nil {
		httputil.Unauthorized(w)
		return
	}

	// Call the global tier's discover API to get the list of teams that the user is a member of.
	// Note that we pass the client token to the global tier's discover API. This is ok as
	// global tier will only return info pertaining to this user.
	discover, status, err := s.global.Discover(r.Context(), token)
	if err != nil {
		log.Printf("list folders: team=%s tier=%d: global discover: %v", teamID, s.tierID, err)
		httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{
			"error": "global tier unavailable",
		})
		return
	}
	if status == http.StatusUnauthorized {
		httputil.Unauthorized(w)
		return
	}
	if status != http.StatusOK {
		log.Printf("list folders: team=%s tier=%d: global discover status=%d", teamID, s.tierID, status)
		httputil.WriteJSON(w, http.StatusBadGateway, map[string]string{
			"error": "global tier error",
		})
		return
	}

	// The team user asked for is not in the list of teams that the user is a member of.
	// Return a 403 Forbidden error.
	if !slices.Contains(discover.TeamIDs, teamID) {
		httputil.Forbidden(w)
		return
	}

	result, err := s.store.ListFolders(r.Context(), teamID)
	if err != nil {
		// User has permissions to the team. But the team is not provisioned on this storage tier.
		// It is most likely a wrong routing.
		// TODO:sudhakar - Se if we need to do some type of redirect here.
		if errors.Is(err, storagedb.ErrTeamNotFound) {
			httputil.NotFound(w)
			return
		}
		log.Printf("list folders: team=%s tier=%d: %v", teamID, s.tierID, err)
		httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "internal error",
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, result)
}
