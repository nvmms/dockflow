package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/filesystem"
	"errors"
	"fmt"

	"github.com/samber/lo"
)

var (
	ErrServiceExist = errors.New("database not found")
	// ErrdatabaseNotExist   = errors.New("database not exist")
	// ErrdatabaseExist      = errors.New("database name is exist")
	// ErrdatabaseNotSuppert = errors.New("database not suppert")
)

func CreateService(service domain.ServiceSpec) error {
	// ---------- load namespace ----------
	ns, err := filesystem.LoadNamespace(service.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	// ---------- duplicate check ----------
	_, found := lo.Find(ns.Service, func(s domain.ServiceSpec) bool {
		return s.Name == service.Name
	})
	if found {
		return ErrServiceExist
	}

	// ---------- basic validate ----------
	if service.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if service.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	if service.Token == "" {
		return fmt.Errorf("token is required")
	}

	// ---------- trigger validate ----------
	switch service.Trigger.Type {
	case "branch", "tag":
	default:
		return fmt.Errorf("invalid trigger type: %s", service.Trigger.Type)
	}

	if service.Trigger.Rule == "" {
		return fmt.Errorf("trigger rule is required")
	}

	// ---------- env validate ----------
	for _, env := range service.Envs {
		if env.Key == "" {
			return fmt.Errorf("env key is empty")
		}
	}

	// ---------- url validate ----------
	if len(service.URLs) == 0 {
		return fmt.Errorf("service must have at least one url")
	}

	for _, u := range service.URLs {
		if u.Host == "" {
			return fmt.Errorf("service url host is empty")
		}
		if u.Port == "" {
			return fmt.Errorf("service url port is empty")
		}
	}

	// ---------- append & save ----------
	ns.Service = append(ns.Service, service)

	if err := filesystem.SaveNamespace(*ns); err != nil {
		return err
	}

	return nil
}
