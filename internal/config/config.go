//////////////////////////////////////////////////////////////
//
// Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
// Licensed under the Apache License, Version 2.0 (the "License")
//
// Environment variable helpers with defaults for service configuration.
//
package config

import "os"

// FromEnv reads a string environment variable or returns the default.
func FromEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
