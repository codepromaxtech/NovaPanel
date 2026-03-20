package main

import (
"context"
"fmt"
"os"

"github.com/jackc/pgx/v5/pgxpool"
"github.com/redis/go-redis/v9"
"github.com/novapanel/novapanel/internal/queue"
)

func main() {
// Connect to DB
dbURL := "postgres://novapanel:novapanel_secret@localhost:5432/novapanel?sslmode=disable"
pool, err := pgxpool.New(context.Background(), dbURL)
if err != nil {
fmt.Printf("DB error: %v\n", err)
os.Exit(1)
}
defer pool.Close()

// Connect to Redis
rdb := redis.NewClient(&redis.Options{
Addr: "localhost:6379",
})

// Create Queue
taskQueue := queue.NewTaskQueue(rdb, pool)

// Fetch Admin ID
var adminID string
err = pool.QueryRow(context.Background(), "SELECT id FROM users WHERE role = 'admin' LIMIT 1").Scan(&adminID)
if err != nil {
fmt.Printf("Admin User query error: %v\n", err)
os.Exit(1)
}

// Insert Dummy Server into DB
sshKey := `-----BEGIN OPENSSH PRIVATE KEY-----
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
TxjI+a31ansxAAAAB2VycEBlcnABAgM=
-----END OPENSSH PRIVATE KEY-----`

var serverID string
err = pool.QueryRow(context.Background(),
`INSERT INTO servers (name, hostname, ip_address, port, os, role, status, agent_status, ssh_key)
 VALUES ('Test Provision Node', 'test-node', '172.18.0.5', 22, 'ubuntu 22.04', 'web', 'pending', 'disconnected', $1)
 RETURNING id`, sshKey).Scan(&serverID)
if err != nil {
fmt.Printf("Server insert error: %v\n", err)
os.Exit(1)
}
fmt.Printf("Created test server with ID: %s\n", serverID)

// Enqueue Task
taskID, err := taskQueue.Enqueue(context.Background(), "server:setup", map[string]interface{}{
"server_id": serverID,
"ip":        "172.18.0.5",
"role":      "web",
}, 1, serverID, "")
if err != nil {
fmt.Printf("Queue error: %v\n", err)
os.Exit(1)
}

fmt.Printf("Successfully queued server:setup task ID: %s\n", taskID)
}
