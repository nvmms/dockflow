package usecase

import (
	"dockflow/internal/config"
	"dockflow/internal/domain"
	"dockflow/internal/service"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/git"
	"dockflow/internal/util"
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

	app.Secret = util.GenerateRandomString(32)
	gitinfo, err := domain.NewGitUrl(app.Repo)
	if err != nil {
		return err
	}

	_token := app.Token
	if _token == "" {
		_token, err = config.FindGit(gitinfo.Host, gitinfo.Username)
		if err != nil {
			return err
		}
	}

	opt := git.WebhookOption{
		Repo:        app.Repo,
		Secret:      app.Secret,
		Token:       _token,
		CallbackURL: fmt.Sprintf("http://117.50.200.150:8090/webhook/git/%s/%s", gitinfo.Username, gitinfo.Repo),
	}
	err = git.NormalizeWebhookOption(opt)
	if err != nil {
		return err
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

func RemoveApp(nsName, appName string) error {
	ns, err := domain.NewNamespace(nsName)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	app, found := ns.FindApp(appName)
	if !found {
		return ErrAppNotFound
	}

	for _, deploy := range app.Deploy {
		err := docker.StopContainer(deploy.ContainerId, nil)
		if err != nil {
			return err
		}
		err = docker.RemoveContainer(deploy.ContainerId, true)
		if err != nil {
			return err
		}
	}

	ns.RemoveApp(appName)
	return nil
}
