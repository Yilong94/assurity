package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"assurity/assignment/internal/adapters/persistence/postgres"
	"assurity/assignment/internal/application"
	"assurity/assignment/internal/presentation/httpapi"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	sql, err := postgres.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer sql.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := postgres.NewRepository(sql)
	if err := repo.Migrate(ctx); err != nil {
		log.Fatal(err)
	}

	statusApp := &application.StatusService{Repo: repo}

	mux := http.NewServeMux()
	api := &httpapi.API{Status: statusApp}
	api.Register(mux)

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpapi.WithCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shCancel()
	_ = srv.Shutdown(shCtx)
}
