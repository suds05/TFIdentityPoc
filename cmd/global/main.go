package main

import (
	"log"
	"net/http"

	"tfidentitypoc/internal/config"
	"tfidentitypoc/internal/global"
)

func main() {
	addr := config.FromEnv("LISTEN_ADDR", ":8080")
	_ = config.FromEnv("MONGODB_URI", "mongodb://mongo:27017")
	_ = config.FromEnv("MONGODB_DATABASE", "identity")
	_ = config.FromEnv("JWT_SECRET", "")

	srv := global.NewServer()
	log.Printf("global tier listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
