package provisioner

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerInfo holds the SSH connection details for a server
type ServerInfo struct {
	IPAddress  string
	Port       int
	SSHUser    string
	SSHKey     string
	SSHPassword string
	AuthMethod string // "password" or "key"
}

// Result holds the output of a provisioning step
type Result struct {
	Module    string `json:"module"`
	Status    string `json:"status"` // completed, failed
	Output    string `json:"output"`
	Duration  string `json:"duration"`
}

// RunScript SSHes into the server and runs a shell script, returning output.
// Automatically wraps the script with sudo when the SSH user is not root.
func RunScript(server ServerInfo, script string) (string, error) {
	needsSudo := server.SSHUser != "" && server.SSHUser != "root"
	return runSSH(server, script, needsSudo)
}

// RunScriptAsUser SSHes into the server and runs a script as the logged-in user
// (no sudo). Use this for user-level operations like ~/.ssh/authorized_keys setup.
func RunScriptAsUser(server ServerInfo, script string) (string, error) {
	return runSSH(server, script, false)
}

func runSSH(server ServerInfo, script string, useSudo bool) (string, error) {
	config, err := buildSSHConfig(server)
	if err != nil {
		return "", fmt.Errorf("ssh config error: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server.IPAddress, server.Port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("ssh dial error to %s: %w", addr, err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session error: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Build the inner script with error handling
	innerScript := fmt.Sprintf("set -e\nexport DEBIAN_FRONTEND=noninteractive\n%s", script)

	var wrappedScript string
	if useSudo {
		// Escape single quotes in the script to safely embed in bash -c '...'
		escapedScript := strings.ReplaceAll(innerScript, "'", "'\"'\"'")

		if server.SSHPassword != "" {
			// Use sudo -S to read password from stdin (pipe it via echo)
			// Escape single quotes in password too
			escapedPass := strings.ReplaceAll(server.SSHPassword, "'", "'\"'\"'")
			wrappedScript = fmt.Sprintf("echo '%s' | sudo -S bash -c '%s'", escapedPass, escapedScript)
		} else {
			// No password available — try passwordless sudo (NOPASSWD in sudoers)
			wrappedScript = fmt.Sprintf("sudo -n bash -c '%s'", escapedScript)
		}
	} else {
		wrappedScript = innerScript
	}

	if err := session.Run(wrappedScript); err != nil {
		return stderr.String() + "\n" + stdout.String(), fmt.Errorf("script failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// ProvisionServer runs all module install scripts for a server
func ProvisionServer(server ServerInfo, modules []string, onProgress func(module, status, output string)) []Result {
	var results []Result

	for _, mod := range modules {
		script, ok := InstallScripts[mod]
		if !ok {
			log.Printf("  No install script for module: %s", mod)
			results = append(results, Result{Module: mod, Status: "failed", Output: "no install script available"})
			if onProgress != nil {
				onProgress(mod, "failed", "no install script available")
			}
			continue
		}

		if onProgress != nil {
			onProgress(mod, "running", "")
		}

		start := time.Now()
		output, err := RunScript(server, script)
		duration := time.Since(start).Round(time.Second).String()

		if err != nil {
			log.Printf("  Module %s failed: %v", mod, err)
			results = append(results, Result{Module: mod, Status: "failed", Output: output, Duration: duration})
			if onProgress != nil {
				onProgress(mod, "failed", output)
			}
		} else {
			log.Printf("  Module %s installed in %s", mod, duration)
			results = append(results, Result{Module: mod, Status: "completed", Output: output, Duration: duration})
			if onProgress != nil {
				onProgress(mod, "completed", output)
			}
		}
	}

	return results
}

func buildSSHConfig(server ServerInfo) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            server.SSHUser,
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For panel use
	}

	if server.AuthMethod == "key" && server.SSHKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(server.SSHKey))
		if err != nil {
			// Try with empty passphrase first, then treat as unencrypted
			return nil, fmt.Errorf("failed to parse SSH key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if server.SSHPassword != "" {
		config.Auth = []ssh.AuthMethod{ssh.Password(server.SSHPassword)}
	} else {
		return nil, fmt.Errorf("no valid SSH auth method available")
	}

	// Add keyboard-interactive as fallback
	if server.SSHPassword != "" {
		config.Auth = append(config.Auth, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range answers {
					answers[i] = server.SSHPassword
				}
				return answers, nil
			},
		))
	}

	// Also handle hostkey verification for known hosts
	config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil // Trust all hosts in panel context
	}

	return config, nil
}
