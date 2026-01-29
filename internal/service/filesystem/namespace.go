package filesystem

import "dockflow/internal/domain"

func ListNamespaces() []domain.Namespace {
	return []domain.Namespace{}
}

func LoadNamespace(name string) (*domain.Namespace, error) {
	return nil, nil
}

func SaveNamespace(name *domain.Namespace) error {
	return nil
}

func RemoveNamespace(name string) error {
	return nil
}
