package usecase

import "dockflow/internal/service/docker"

func Init() error {
	if err := docker.CheckDocker(); err != nil {
		return err
	}
	// next: PrepareWorkspace, EnsureTraefik ...
	return nil
}
