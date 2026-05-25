package config

import "os"

// FromEnv reads a string environment variable or returns the default.
func FromEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
