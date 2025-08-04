package private

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetClientIP_FromSSHConnection(t *testing.T) {
	origSSHConn := os.Getenv("SSH_CONNECTION")
	origSSHClient := os.Getenv("SSH_CLIENT")
	defer func() {
		_ = os.Setenv("SSH_CONNECTION", origSSHConn)
		_ = os.Setenv("SSH_CLIENT", origSSHClient)
	}()

	_ = os.Setenv("SSH_CONNECTION", "192.168.1.55 12345 10.0.0.1 22")
	_ = os.Unsetenv("SSH_CLIENT")

	assert.Equal(t, "192.168.1.55", getClientIP())
}

func TestGetClientIP_FromSSHClient(t *testing.T) {
	origSSHConn := os.Getenv("SSH_CONNECTION")
	origSSHClient := os.Getenv("SSH_CLIENT")
	defer func() {
		_ = os.Setenv("SSH_CONNECTION", origSSHConn)
		_ = os.Setenv("SSH_CLIENT", origSSHClient)
	}()

	_ = os.Unsetenv("SSH_CONNECTION")
	_ = os.Setenv("SSH_CLIENT", "10.1.2.3 34567 22")

	assert.Equal(t, "10.1.2.3", getClientIP())
}

func TestGetClientIP_Default(t *testing.T) {
	origSSHConn := os.Getenv("SSH_CONNECTION")
	origSSHClient := os.Getenv("SSH_CLIENT")
	defer func() {
		_ = os.Setenv("SSH_CONNECTION", origSSHConn)
		_ = os.Setenv("SSH_CLIENT", origSSHClient)
	}()

	_ = os.Unsetenv("SSH_CONNECTION")
	_ = os.Unsetenv("SSH_CLIENT")

	assert.Equal(t, "127.0.0.1", getClientIP())
}
