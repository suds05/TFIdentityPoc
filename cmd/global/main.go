package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"tfidentitypoc/internal/config"
	"tfidentitypoc/internal/global"
	"tfidentitypoc/internal/globaldb"
)

func main() {
	addr := config.FromEnv("LISTEN_ADDR", ":8080")
	mongoURI := config.FromEnv("MONGODB_URI", "mongodb://mongo:27017")
	dbName := config.FromEnv("MONGODB_DATABASE", "global")
	jwtSecret := config.FromEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("mongodb connect: %v", err)
	}
	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer dcancel()
		_ = client.Disconnect(dctx)
	}()

	store := globaldb.NewStore(client.Database(dbName))
	srv := global.NewServer(jwtSecret, store)

	httpSrv := &http.Server{Addr: addr, Handler: srv.Handler()}
	go func() {
		log.Printf("global tier listening on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}
