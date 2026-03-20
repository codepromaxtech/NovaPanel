package services

// ServerForSSH is a lightweight struct for SSH connection details.
type ServerForSSH struct {
	IPAddress string
	Port      int
	Role      string
	SSHKey    string
}
