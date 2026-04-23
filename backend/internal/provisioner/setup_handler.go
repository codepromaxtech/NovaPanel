package provisioner

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	novacrypto "github.com/novapanel/novapanel/internal/crypto"
)

// SetupHandler processes "server_setup" tasks from the queue.
// It reads the server SSH details from DB, provisions each module,
// and writes progress to setup_logs.
type SetupHandler struct {
	db *pgxpool.Pool
}

func NewSetupHandler(db *pgxpool.Pool) *SetupHandler {
	return &SetupHandler{db: db}
}

// Handle is the TaskHandler function signature: func(ctx, payload) (result, error)
func (h *SetupHandler) Handle(ctx context.Context, payload map[string]interface{}) (map[string]interface{}, error) {
	serverID, _ := payload["server_id"].(string)
	modulesRaw, _ := payload["modules"].([]interface{})

	if serverID == "" || len(modulesRaw) == 0 {
		return nil, fmt.Errorf("server_id and modules are required")
	}

	var modules []string
	for _, m := range modulesRaw {
		if s, ok := m.(string); ok {
			modules = append(modules, s)
		}
	}

	log.Printf("🔧 Starting server setup for %s with modules: %v", serverID, modules)

	// Update server setup_status
	h.db.Exec(ctx, `UPDATE servers SET setup_status = 'running' WHERE id = $1`, serverID)

	// Get server SSH info
	var server ServerInfo
	var port int
	var encKey, encPassword string
	err := h.db.QueryRow(ctx,
		`SELECT host(ip_address), port, ssh_user, COALESCE(ssh_key, ''), COALESCE(ssh_password, ''), COALESCE(auth_method, 'password')
		 FROM servers WHERE id = $1`, serverID,
	).Scan(&server.IPAddress, &port, &server.SSHUser, &encKey, &encPassword, &server.AuthMethod)
	if err != nil {
		h.db.Exec(ctx, `UPDATE servers SET setup_status = 'failed' WHERE id = $1`, serverID)
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}
	server.Port = port

	// Decrypt SSH credentials stored encrypted in the database
	if cryptoKey, kerr := novacrypto.GetEncryptionKey(); kerr == nil {
		if encKey != "" {
			if dec, derr := novacrypto.Decrypt(encKey, cryptoKey); derr == nil {
				encKey = dec
			}
		}
		if encPassword != "" {
			if dec, derr := novacrypto.Decrypt(encPassword, cryptoKey); derr == nil {
				encPassword = dec
			}
		}
	}
	server.SSHKey = encKey
	server.SSHPassword = encPassword

	// Create initial setup_logs entries (all pending)
	for _, mod := range modules {
		h.db.Exec(ctx,
			`INSERT INTO setup_logs (server_id, module, status) VALUES ($1, $2, 'pending')
			 ON CONFLICT DO NOTHING`,
			serverID, mod)
	}

	// Run provisioning with progress tracking
	allCompleted := true
	results := ProvisionServer(server, modules, func(module, status, output string) {
		now := time.Now()
		switch status {
		case "running":
			h.db.Exec(ctx,
				`UPDATE setup_logs SET status = 'running', started_at = $3 WHERE server_id = $1 AND module = $2`,
				serverID, module, now)
		case "completed":
			h.db.Exec(ctx,
				`UPDATE setup_logs SET status = 'completed', output = $3, completed_at = $4 WHERE server_id = $1 AND module = $2`,
				serverID, module, output, now)
			// Mark module as installed in server_modules
			h.db.Exec(ctx,
				`INSERT INTO server_modules (server_id, module, enabled) VALUES ($1, $2, true)
				 ON CONFLICT (server_id, module) DO UPDATE SET enabled = true`,
				serverID, module)
		case "failed":
			allCompleted = false
			h.db.Exec(ctx,
				`UPDATE setup_logs SET status = 'failed', output = $3, completed_at = $4 WHERE server_id = $1 AND module = $2`,
				serverID, module, output, now)
		}
	})

	// Update server overall status
	if allCompleted {
		h.db.Exec(ctx, `UPDATE servers SET setup_status = 'completed' WHERE id = $1`, serverID)
	} else {
		h.db.Exec(ctx, `UPDATE servers SET setup_status = 'partial' WHERE id = $1`, serverID)
	}

	// Build result summary
	resultMap := map[string]interface{}{
		"server_id": serverID,
		"total":     len(modules),
	}
	completed := 0
	for _, r := range results {
		if r.Status == "completed" {
			completed++
		}
	}
	resultMap["completed"] = completed
	resultMap["failed"] = len(modules) - completed

	log.Printf("✅ Server setup done: %d/%d modules installed for %s", completed, len(modules), serverID)

	return resultMap, nil
}
