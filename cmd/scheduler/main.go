package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"assurity/assignment/internal/adapters/config/yamlconfig"
	"assurity/assignment/internal/adapters/persistence/postgres"
	sqsadapter "assurity/assignment/internal/adapters/queue/sqs"
	"assurity/assignment/internal/application"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/config/services.yaml"
	}

	tick := 5 * time.Second
	if v := os.Getenv("SCHEDULER_TICK"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Fatalf("SCHEDULER_TICK: %v", err)
		}
		tick = d
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

	loader := yamlconfig.NewLoader(configPath)

	scheduler := &application.SchedulerService{
		Loader: loader,
		Repo:   repo,
		Queue:  queue,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		cancel()
	}()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	run := func() {
		n, err := scheduler.Run(ctx)
		if err != nil {
			log.Printf("scheduler tick: %v", err)
			return
		}
		if n > 0 {
			log.Printf("enqueued %d ping job(s)", n)
		}
	}

	run()
	for {
		select {
		case <-ctx.Done():
			log.Println("scheduler stopping")
			return
		case <-ticker.C:
			run()
		}
	}
}
