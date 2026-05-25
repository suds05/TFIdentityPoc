//go:build ignore

//////////////////////////////////////////////////////////////
//
// Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License")
//
// CLI helper to mint signed POC JWTs for curl, Postman, and smoke test scripts.
// Usage: go run scripts/mint_jwt.go [secret] [sub]
//
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := "poc-dev-secret"
	sub := "usr_sudhakan"
	if len(os.Args) > 1 {
		secret = os.Args[1]
	}
	if len(os.Args) > 2 {
		sub = os.Args[2]
	}
	claims := jwt.MapClaims{
		"sub":    sub,
		"email":  "sudhakan@gmail.com",
		"org_id": "org_acme",
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign jwt: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(token)
}
