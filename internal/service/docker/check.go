package docker

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

var (
	ErrDockerNotFound   = errors.New("docker command not found")
	ErrDockerNotRunning = errors.New("docker daemon not running")
	ErrDockerNoPerm     = errors.New("no permission to access docker daemon")
)

func isNotFound(err error) bool {
	return strings.Contains(err.Error(), "executable file not found") ||
		strings.Contains(err.Error(), "not found")
}

func isNoPermission(err error) bool {
	return strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "Got permission denied")
}

func isDaemonNotRunning(err error) bool {
	return strings.Contains(err.Error(), "Cannot connect to the Docker daemon") ||
		strings.Contains(err.Error(), "Is the docker daemon running")
}

// CheckDocker verifies docker availability and permission
func CheckDocker() error {
	// 1. check docker command exists
	if err := run("docker", "version"); err != nil {
		if isNotFound(err) {
			return ErrDockerNotFound
		}
		return err
	}

	// 2. check docker daemon
	if err := run("docker", "info"); err != nil {
		if isNoPermission(err) {
			return ErrDockerNoPerm
		}
		if isDaemonNotRunning(err) {
			return ErrDockerNotRunning
		}
		return err
	}

	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return errors.New(stderr.String())
		}
		return err
	}
	return nil
}
