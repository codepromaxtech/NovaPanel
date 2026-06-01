package provisioner

import (
	"fmt"
	"log"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	poolIdleTTL     = 5 * time.Minute
	poolCleanPeriod = 2 * time.Minute
)

type poolEntry struct {
	client   *ssh.Client
	lastUsed time.Time
	mu       sync.Mutex
}

// sshPool is the process-wide SSH connection pool.
// Key format: "<user>@<host>:<port>" (derived from ServerInfo).
var sshPool sync.Map // map[string]*poolEntry

func init() {
	// Background goroutine evicts idle or broken connections.
	go func() {
		ticker := time.NewTicker(poolCleanPeriod)
		defer ticker.Stop()
		for range ticker.C {
			cleanPool()
		}
	}()
}

func poolKey(server ServerInfo) string {
	port := server.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s@%s:%d", server.SSHUser, server.IPAddress, port)
}

// getOrDial returns a pooled *ssh.Client, dialling a new one if needed.
// Caller must NOT close the client; the pool owns its lifetime.
func getOrDial(server ServerInfo) (*ssh.Client, error) {
	key := poolKey(server)

	if raw, ok := sshPool.Load(key); ok {
		entry := raw.(*poolEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()

		// Probe liveness with a cheap keepalive request.
		if _, _, err := entry.client.SendRequest("keepalive@openssh.com", true, nil); err == nil {
			entry.lastUsed = time.Now()
			return entry.client, nil
		}
		// Connection is broken — close silently and fall through to redial.
		entry.client.Close()
		sshPool.Delete(key)
	}

	cfg, err := buildSSHConfig(server)
	if err != nil {
		return nil, fmt.Errorf("ssh config: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server.IPAddress, server.Port)
	if server.Port == 0 {
		addr = server.IPAddress + ":22"
	}

	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}

	entry := &poolEntry{client: client, lastUsed: time.Now()}
	// If two goroutines race to create for the same key, keep only one.
	if actual, loaded := sshPool.LoadOrStore(key, entry); loaded {
		client.Close() // discard the one we just dialled
		return actual.(*poolEntry).client, nil
	}
	return client, nil
}

// runSSHPooled uses a pooled client to run a single command/script.
func runSSHPooled(server ServerInfo, script string, useSudo bool) (string, error) {
	client, err := getOrDial(server)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		// Pool entry went bad between probe and use — evict and retry once.
		key := poolKey(server)
		sshPool.Delete(key)
		client.Close()

		client2, err2 := getOrDial(server)
		if err2 != nil {
			return "", fmt.Errorf("ssh session (retry): %w", err2)
		}
		session, err = client2.NewSession()
		if err != nil {
			return "", fmt.Errorf("ssh session: %w", err)
		}
	}
	defer session.Close()

	return runSession(session, server, script, useSudo)
}

func cleanPool() {
	cutoff := time.Now().Add(-poolIdleTTL)
	sshPool.Range(func(key, value any) bool {
		entry := value.(*poolEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if entry.lastUsed.Before(cutoff) {
			entry.client.Close()
			sshPool.Delete(key)
			log.Printf("ssh pool: evicted idle connection %s", key.(string))
		}
		return true
	})
}

// EvictServer removes the pooled connection for a specific server (call on
// server delete / credential change).
func EvictServer(server ServerInfo) {
	key := poolKey(server)
	if raw, ok := sshPool.Load(key); ok {
		raw.(*poolEntry).client.Close()
		sshPool.Delete(key)
	}
}

// buildSSHClientConfig is an alias kept for pool internal use.
// The real implementation lives in provisioner.go as buildSSHConfig.

// runSession executes a script on an existing ssh.Session.
func runSession(session *ssh.Session, server ServerInfo, script string, useSudo bool) (string, error) {
	var outBuf, errBuf syncBuffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	innerScript := "set -e\nexport DEBIAN_FRONTEND=noninteractive\n" + script

	var wrappedScript string
	if useSudo {
		escaped := singleQuoteEscape(innerScript)
		if server.SSHPassword != "" {
			escapedPass := singleQuoteEscape(server.SSHPassword)
			wrappedScript = "echo '" + escapedPass + "' | sudo -S bash -c '" + escaped + "'"
		} else {
			wrappedScript = "sudo -n bash -c '" + escaped + "'"
		}
	} else {
		wrappedScript = innerScript
	}

	if err := session.Run(wrappedScript); err != nil {
		return errBuf.String() + "\n" + outBuf.String(),
			fmt.Errorf("script failed: %w\nstderr: %s", err, errBuf.String())
	}
	return outBuf.String(), nil
}

func singleQuoteEscape(s string) string {
	out := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); i++ {
		if s[i] == '\'' {
			out = append(out, '\'', '"', '\'', '"', '\'')
		} else {
			out = append(out, s[i])
		}
	}
	return string(out)
}

// syncBuffer is a bytes.Buffer safe for concurrent stdout/stderr writes.
type syncBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.buf = append(b.buf, p...)
	b.mu.Unlock()
	return len(p), nil
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

// PoolStats returns current pool size (for monitoring/debug).
func PoolStats() (total int) {
	sshPool.Range(func(_, _ any) bool { total++; return true })
	return
}
