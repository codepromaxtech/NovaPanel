package queue

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	queue    *TaskQueue
	handlers map[string]TaskHandler
	stop     chan struct{}
}

type TaskHandler func(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error)

func NewWorker(queue *TaskQueue) *Worker {
	return &Worker{
		queue:    queue,
		handlers: make(map[string]TaskHandler),
		stop:     make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a specific task type
func (w *Worker) RegisterHandler(taskType string, handler TaskHandler) {
	w.handlers[taskType] = handler
}

// Start begins processing tasks from the queue
func (w *Worker) Start(workerCount int) {
	log.Printf("✓ Starting %d task workers", workerCount)
	for i := 0; i < workerCount; i++ {
		go w.process(i)
	}
}

// Stop signals all workers to stop
func (w *Worker) Stop() {
	close(w.stop)
}

func (w *Worker) process(workerID int) {
	log.Printf("  Worker %d started", workerID)
	for {
		select {
		case <-w.stop:
			log.Printf("  Worker %d stopped", workerID)
			return
		default:
			ctx := context.Background()
			task, err := w.queue.Dequeue(ctx, 5*time.Second)
			if err != nil {
				log.Printf("  Worker %d: dequeue error: %v", workerID, err)
				time.Sleep(1 * time.Second)
				continue
			}
			if task == nil {
				continue
			}

			log.Printf("  Worker %d: processing task %s (type: %s)", workerID, task.ID, task.Type)

			handler, ok := w.handlers[task.Type]
			if !ok {
				log.Printf("  Worker %d: no handler for task type: %s", workerID, task.Type)
				w.queue.Fail(ctx, task.ID, "no handler registered for task type: "+task.Type)
				continue
			}

			result, err := handler(ctx, task.Payload)
			if err != nil {
				log.Printf("  Worker %d: task %s failed: %v", workerID, task.ID, err)
				w.queue.Fail(ctx, task.ID, err.Error())
				continue
			}

			w.queue.Complete(ctx, task.ID, result)
			log.Printf("  Worker %d: task %s completed", workerID, task.ID)
		}
	}
}
