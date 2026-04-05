package main

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"assurity/assignment/internal/application"
	"assurity/assignment/internal/domain"
	"assurity/assignment/internal/domain/ports"
)

// start long-polls the queue and processes messages with up to workerConcurrency() goroutines.
func start(ctx context.Context, queue ports.JobQueue, worker *application.WorkerService) {
	concurrency := workerConcurrency()
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for run(ctx, queue, worker, sem, &wg) {
	}
	wg.Wait()
	log.Println("worker stopping")
}

// run receives one message and starts processing; returns false when the consume loop should exit.
func run(ctx context.Context, queue ports.JobQueue, worker *application.WorkerService, sem chan struct{}, wg *sync.WaitGroup) bool {
	msg, err := queue.Receive(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return false
		}
		log.Printf("receive: %v", err)
		time.Sleep(time.Second)
		return true
	}

	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		return false
	}

	wg.Add(1)
	go func(m domain.ReceivedProbeJob) {
		defer wg.Done()
		defer func() { <-sem }()

		// Use an uncanceled context so in-flight checks finish after SIGTERM; Receive already stopped.
		workerCtx := context.Background()
		if err := worker.Process(workerCtx, m); err != nil {
			log.Printf("process service_id=%d: %v (message will retry after visibility timeout)", m.Job.ServiceID, err)
			return
		}
		log.Printf("service_id=%d check completed", m.Job.ServiceID)

		if err := queue.Delete(workerCtx, m.ReceiptHandle); err != nil {
			log.Printf("delete message: %v", err)
		}
	}(msg)

	return true
}

func workerConcurrency() int {
	v := os.Getenv("WORKER_CONCURRENCY")
	if v == "" {
		return 4
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 4
	}
	if n > 256 {
		return 256
	}
	return n
}
