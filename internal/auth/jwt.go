// ////////////////////////////////////////////////////////////
//
// Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License")
//
// JWT bearer extraction and HMAC validation.
package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// UserClaims holds validated JWT identity fields used by the POC tiers.
type UserClaims struct {
	UserID string
	Email  string
	OrgID  string
}

type pocClaims struct {
	Email string `json:"email"`
	OrgID string `json:"org_id"`
	jwt.RegisteredClaims
}

// BearerToken extracts the token from an Authorization Bearer header.
func BearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("authorization header must use Bearer scheme")
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}

// ParseUserJWT validates an HS256 JWT and returns canonical user claims.
// TODO:sudhakar - We are currently doing a simple HMAC validation.
// This should witch to validating with Auth provider's public key.
func ParseUserJWT(secret, tokenString string) (UserClaims, error) {
	if secret == "" {
		return UserClaims{}, errors.New("jwt secret not configured")
	}
	claims := &pocClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return UserClaims{}, errors.New("invalid jwt")
	}
	sub, err := claims.GetSubject()
	if err != nil || sub == "" {
		return UserClaims{}, errors.New("jwt missing sub claim")
	}
	return UserClaims{
		UserID: sub,
		Email:  claims.Email,
		OrgID:  claims.OrgID,
	}, nil
}
