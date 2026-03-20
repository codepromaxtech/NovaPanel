package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const taskQueueKey = "novapanel:tasks"

type TaskQueue struct {
	rdb *redis.Client
	db  *pgxpool.Pool
}

type TaskPayload struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Payload  map[string]interface{} `json:"payload"`
	Priority int                    `json:"priority"`
	ServerID string                 `json:"server_id,omitempty"`
	UserID   string                 `json:"user_id,omitempty"`
}

func NewTaskQueue(rdb *redis.Client, db *pgxpool.Pool) *TaskQueue {
	return &TaskQueue{rdb: rdb, db: db}
}

// Enqueue adds a task to the Redis queue and persists it in PostgreSQL
func (q *TaskQueue) Enqueue(ctx context.Context, taskType string, payload map[string]interface{}, priority int, serverID, userID string) (string, error) {
	taskID := uuid.New().String()

	task := TaskPayload{
		ID:       taskID,
		Type:     taskType,
		Payload:  payload,
		Priority: priority,
		ServerID: serverID,
		UserID:   userID,
	}

	data, err := json.Marshal(task)
	if err != nil {
		return "", fmt.Errorf("failed to marshal task: %w", err)
	}

	// Push to Redis queue
	if err := q.rdb.LPush(ctx, taskQueueKey, data).Err(); err != nil {
		return "", fmt.Errorf("failed to enqueue task: %w", err)
	}

	// Persist in PostgreSQL
	payloadJSON, _ := json.Marshal(payload)
	_, err = q.db.Exec(ctx,
		`INSERT INTO tasks (id, type, payload, status, priority, server_id, user_id)
		 VALUES ($1, $2, $3, 'queued', $4, NULLIF($5, '')::uuid, NULLIF($6, '')::uuid)`,
		taskID, taskType, payloadJSON, priority, serverID, userID)
	if err != nil {
		log.Printf("Warning: failed to persist task %s: %v", taskID, err)
	}

	return taskID, nil
}

// Dequeue retrieves the next task from the queue
func (q *TaskQueue) Dequeue(ctx context.Context, timeout time.Duration) (*TaskPayload, error) {
	result, err := q.rdb.BRPop(ctx, timeout, taskQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var task TaskPayload
	if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Update status in DB
	q.db.Exec(ctx, "UPDATE tasks SET status = 'running', started_at = NOW(), attempts = attempts + 1 WHERE id = $1", task.ID)

	return &task, nil
}

// Complete marks a task as completed
func (q *TaskQueue) Complete(ctx context.Context, taskID string, result map[string]interface{}) error {
	resultJSON, _ := json.Marshal(result)
	_, err := q.db.Exec(ctx,
		"UPDATE tasks SET status = 'completed', completed_at = NOW(), result = $1 WHERE id = $2",
		resultJSON, taskID)
	return err
}

// Fail marks a task as failed
func (q *TaskQueue) Fail(ctx context.Context, taskID string, errMsg string) error {
	_, err := q.db.Exec(ctx,
		"UPDATE tasks SET status = 'failed', completed_at = NOW(), error = $1 WHERE id = $2",
		errMsg, taskID)
	return err
}

// QueueLength returns the number of pending tasks
func (q *TaskQueue) QueueLength(ctx context.Context) (int64, error) {
	return q.rdb.LLen(ctx, taskQueueKey).Result()
}
