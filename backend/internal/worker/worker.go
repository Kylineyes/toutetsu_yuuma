package worker

import (
	"context"
	"log"
	"time"

	"github.com/Kylineyes/toutetsu_yuuma/backend/internal/pipeline"
)

// Worker is a simple in-process pool backed by a buffered channel (SQLite is the durable queue).
type Worker struct {
	runner *pipeline.Runner
	ch     chan string
}

// New starts n goroutines consuming task ids.
func New(r *pipeline.Runner, n int) *Worker {
	w := &Worker{runner: r, ch: make(chan string, 1024)}
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		go w.loop()
	}
	return w
}

func (w *Worker) loop() {
	for id := range w.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Hour)
		func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("[worker] task %s panic: %v", id, rec)
				}
			}()
			w.runner.Run(ctx, id)
		}()
		cancel()
	}
}

// Enqueue submits a task id for processing.
func (w *Worker) Enqueue(id string) {
	w.ch <- id
}
