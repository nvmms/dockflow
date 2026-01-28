package usecase

import (
	"dockflow/internal/config"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"dockflow/internal/service/traefik"
)

func Init() error {
	if err := docker.CheckDocker(); err != nil {
		return err
	}

	if err := filesystem.PrepareWorkspace(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	_ = cfg // 现在不用，先保证能读

	if err := traefik.Init(); err != nil {
		return err
	}

	// next: EnsureNetwork / EnsureTraefik
	return nil
}
