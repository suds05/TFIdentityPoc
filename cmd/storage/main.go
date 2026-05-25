package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"tfidentitypoc/internal/config"
	"tfidentitypoc/internal/globalclient"
	"tfidentitypoc/internal/storage"
	"tfidentitypoc/internal/storagedb"
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
	mongoURI := config.FromEnv("MONGODB_URI", "mongodb://mongo:27017")
	dbName := config.FromEnv("MONGODB_DATABASE", "")
	if dbName == "" {
		dbName = fmt.Sprintf("storage_tier_%d", tierID)
	}
	globalURL := config.FromEnv("GLOBAL_TIER_URL", "http://global:8080")
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

	store := storagedb.NewStore(client.Database(dbName))
	global := globalclient.NewClient(globalURL)
	srv := storage.NewServer(tierID, jwtSecret, global, store)

	httpSrv := &http.Server{Addr: addr, Handler: srv.Handler()}
	go func() {
		log.Printf("storage tier %d listening on %s (db=%s)", tierID, addr, dbName)
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
