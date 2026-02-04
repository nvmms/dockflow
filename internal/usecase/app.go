package usecase

import (
	"dockflow/internal/domain"
	"dockflow/internal/service"
	"errors"
	"fmt"

	"github.com/samber/lo"
)

var (
	ErrAppNotFound = errors.New("app not found")
	// ErrdatabaseNotExist   = errors.New("database not exist")
	// ErrdatabaseExist      = errors.New("database name is exist")
	// ErrdatabaseNotSuppert = errors.New("database not suppert")
)

func CreateApp(app domain.AppSpec) error {
	// ---------- load namespace ----------
	ns, err := domain.NewNamespace(app.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	// ---------- duplicate check ----------
	_, found := lo.Find(ns.App, func(s domain.AppSpec) bool {
		return s.Name == app.Name
	})
	if found {
		return fmt.Errorf("app name [%s] is exist", app.Name)
	}

	// ---------- basic validate ----------
	if app.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if app.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	// if app.Token == "" {
	// 	return fmt.Errorf("token is required")
	// }

	// ---------- trigger validate ----------
	switch app.Trigger.Type {
	case "branch", "tag":
	default:
		return fmt.Errorf("invalid trigger type: %s", app.Trigger.Type)
	}

	if app.Trigger.Rule == "" {
		return fmt.Errorf("trigger rule is required")
	}

	// ---------- env validate ----------
	for _, env := range app.Envs {
		if env.Key == "" {
			return fmt.Errorf("env key is empty")
		}
	}

	// ---------- url validate ----------
	if len(app.URLs) == 0 {
		return fmt.Errorf("service must have at least one url")
	}

	for _, u := range app.URLs {
		if u.Host == "" {
			return fmt.Errorf("service url host is empty")
		}
		if u.Port == "" {
			return fmt.Errorf("service url port is empty")
		}
	}

	// ---------- append & save ----------
	ns.App = append(ns.App, app)

	if err := ns.Save(); err != nil {
		return err
	}

	return nil
}

func ListApp(ns string) ([]domain.AppSpec, error) {
	namespace, err := domain.NewNamespace(ns)
	if err != nil {
		return nil, err
	}
	if namespace == nil {
		return nil, ErrNamespaceNotFound
	}

	return namespace.App, nil
}

type DeployAppOptions struct {
	Namespace string
	Name      string
	Branch    string
	Commit    string
	Tag       string
}

func DeployApp(opt DeployAppOptions) error {
	namespace, err := domain.NewNamespace(opt.Namespace)
	if err != nil {
		return err
	}
	if namespace == nil {
		return ErrNamespaceNotFound
	}

	for _, app := range namespace.App {

		if app.Name == opt.Name {
			deploy, err := service.NewAppDeployer(&app)
			if err != nil {
				return err
			}
			err = deploy.Deploy(&opt.Branch, &opt.Commit, &opt.Tag)
			if err != nil {
				return err
			}
			return err
		}
	}

	return fmt.Errorf("app name [%s] not found", opt.Name)
}
