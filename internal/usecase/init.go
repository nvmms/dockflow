package usecase

import (
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
)

func Init() error {
	if err := docker.CheckDocker(); err != nil {
		return err
	}

	if err := filesystem.PrepareWorkspace(); err != nil {
		return err
	}

	// next: EnsureTraefik
	return nil
}
