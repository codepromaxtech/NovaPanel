package agent

import (
	"os/exec"
)

// ExecuteCommand runs a bash command on the host. In later phases, this will securely subscribe to Redis/WebSockets instead of direct HTTP calls, but we start with a simple execution primitive.
func ExecuteCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	return string(output), err
}
