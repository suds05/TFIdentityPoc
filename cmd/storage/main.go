package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"strconv"

	"tfidentitypoc/internal/config"
	"tfidentitypoc/internal/storage"
)

func main() {
	tierIDFlag := flag.Int("tier-id", 0, "storage tier ID (1 or 2)")
	flag.Parse()

	tierID := *tierIDFlag
	if tierID == 0 {
		if v := os.Getenv("TIER_ID"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				tierID = n
			}
		}
	}
	if tierID == 0 {
		log.Fatal("tier-id is required (flag --tier-id or env TIER_ID)")
	}

	addr := config.FromEnv("LISTEN_ADDR", ":8080")
	_ = config.FromEnv("MONGODB_URI", "mongodb://mongo:27017")
	_ = config.FromEnv("MONGODB_DATABASE", "")
	_ = config.FromEnv("GLOBAL_TIER_URL", "http://global:8080")
	_ = config.FromEnv("JWT_SECRET", "")

	srv := storage.NewServer(tierID)
	log.Printf("storage tier %d listening on %s", tierID, addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
