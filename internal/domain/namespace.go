package domain

import (
	"dockflow/internal/service/filesystem"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/samber/lo"
)

type Namespace struct {
	Name      string         `json:"name"`
	Network   string         `json:"network"`
	NetworkId string         `json:"network_id"`
	Subnet    string         `json:"subnet"`
	Gateway   string         `json:"gateway"`
	CreatedAt time.Time      `json:"created_at"`
	Redis     []RedisSpec    `json:"redis"`
	Database  []DatabaseSpec `json:"database"`
	App       []AppSpec      `json:"app"`
}

var (
	ErrNamespaceNotFound       = errors.New("namespace not found")
	ErrNamespaceNotInitialized = errors.New("namespace not initialized")
)

func ListNamespaces() []Namespace {
	var result []Namespace

	entries, err := os.ReadDir(filesystem.NamespaceDirName)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ns, err := NewNamespace(entry.Name())
		if err == nil {
			result = append(result, *ns)
		}
	}

	return result
}

func NewNamespace(name string) (*Namespace, error) {
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

	var ns Namespace
	if err := json.Unmarshal(data, &ns); err != nil {
		return nil, err
	}

	// 4️⃣ 容错：name 不一致时自动修正
	if ns.Name == "" {
		ns.Name = name
	}

	return &ns, nil
}

func (n *Namespace) FindApp(appName string) (AppSpec, bool) {
	return lo.Find(n.App, func(app AppSpec) bool {
		return appName == app.Name
	})
}

func (n *Namespace) RemoveApp(appName string) {
	_, index, found := lo.FindIndexOf(n.App, func(app AppSpec) bool {
		return appName == app.Name
	})
	if found {
		n.App = append(n.App[:index], n.App[index+1:]...)
	}
	n.Save()
}

func (n *Namespace) Save() error {
	if n.Name == "" {
		return errors.New("namespace name required")
	}

	dir := namespaceDir(n.Name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(
		namespaceFile(n.Name),
		data,
		0644,
	)
}

func (n *Namespace) Remove() error {
	if n == nil {
		return errors.New("namespace name required")
	}

	return os.RemoveAll(namespaceDir(n.Name))
}

func namespaceDir(name string) string {
	return filepath.Join(filesystem.NamespaceDirName, name)
}

func namespaceFile(name string) string {
	return filepath.Join(namespaceDir(name), "namespace.json")
}
