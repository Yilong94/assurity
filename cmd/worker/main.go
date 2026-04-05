package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"assurity/assignment/internal/adapters/persistence/postgres"
	"assurity/assignment/internal/adapters/probe/httpprobe"
	sqsadapter "assurity/assignment/internal/adapters/queue/sqs"
	"assurity/assignment/internal/adapters/webhook"
	"assurity/assignment/internal/application"
	"assurity/assignment/internal/domain/ports"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
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

	queue, err := sqsadapter.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer queue.Close()

	var notifier ports.DownNotifier = &webhook.Noop{}
	if u := os.Getenv("ALERT_WEBHOOK_URL"); u != "" {
		notifier = webhook.New(u)
	}

	worker := &application.WorkerService{
		Repo:  repo,
		Probe: httpprobe.New(),
		Alert: notifier,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	start(ctx, queue, worker)
}
