package docker

import (
	"os"
	"os/exec"
)

func EnsureImage(image string) error {
	if imageExists(image) {
		return nil
	}

	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func imageExists(image string) bool {
	cmd := exec.Command("docker", "image", "inspect", image)
	return cmd.Run() == nil
}
