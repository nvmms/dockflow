package filesystem

import (
	"dockflow/internal/domain"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var (
	ErrNamespaceNotFound       = errors.New("namespace not found")
	ErrNamespaceNotInitialized = errors.New("namespace not initialized")
)

func ListNamespaces() []domain.Namespace {
	var result []domain.Namespace

	entries, err := os.ReadDir(NamespaceDirName)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ns, err := LoadNamespace(entry.Name())
		if err == nil {
			result = append(result, *ns)
		}
	}

	return result
}

func LoadNamespace(name string) (*domain.Namespace, error) {
	dir := namespaceDir(name)

	// 1️⃣ 判断 namespace 目录是否存在
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNamespaceNotFound
		}
		return nil, err
	}

	file := namespaceFile(name)

	// 2️⃣ 判断 namespace.json 是否存在
	if _, err := os.Stat(file); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNamespaceNotInitialized
		}
		return nil, err
	}

	// 3️⃣ 读取并解析
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var ns domain.Namespace
	if err := json.Unmarshal(data, &ns); err != nil {
		return nil, err
	}

	// 4️⃣ 容错：name 不一致时自动修正
	if ns.Name == "" {
		ns.Name = name
	}

	return &ns, nil
}
func SaveNamespace(ns domain.Namespace) error {
	if ns.Name == "" {
		return errors.New("namespace name required")
	}

	dir := namespaceDir(ns.Name)
	print(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(
		namespaceFile(ns.Name),
		data,
		0644,
	)
}

func RemoveNamespace(name string) error {
	if name == "" {
		return errors.New("namespace name required")
	}

	return os.RemoveAll(namespaceDir(name))
}

func namespaceDir(name string) string {
	return filepath.Join(NamespaceDirName, name)
}

func namespaceFile(name string) string {
	return filepath.Join(namespaceDir(name), "namespace.json")
}
