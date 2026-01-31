package filesystem

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	BaseDirName         = "/home/vscode/.dockflow"
	NamespaceDirName    = BaseDirName + "/namespace"
	BuildDockerfilePath = BaseDirName + "/build-templates/Dockerfile."
)

func PrepareWorkspace() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	baseDir := filepath.Join(home, BaseDirName)

	// 1. create base dir
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	// 2. sub dirs
	dirs := []string{
		"state",
		"cache",
		"logs",
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(baseDir, d), 0755); err != nil {
			return err
		}
	}

	// 3. dockflow.yaml
	cfgPath := filepath.Join(baseDir, "dockflow.yaml")
	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(cfgPath, defaultConfig(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func defaultConfig() []byte {
	return []byte(`
version: v0.1

platform:
  traefik:
    acmeEmail: ""

currentNamespace: default

namespaces:
  default:
    apps: {}
`)
}

func DirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err // 其他错误（如权限）
}
