package usecase

import (
	"bytes"
	"dockflow/internal/config"
	"dockflow/internal/domain"
	"dockflow/internal/service"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/git"
	"dockflow/internal/service/monitor"
	"dockflow/internal/util"
	"errors"
	"fmt"
	"strings"

	"github.com/docker/docker/pkg/stdcopy"
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

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.WebHookUrl != "" {
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
			CallbackURL: fmt.Sprintf("%s/%s/%s", strings.TrimSuffix(cfg.WebHookUrl, "/"), app.Namespace, app.Name),
		}
		err = git.NormalizeWebhookOption(opt)
		if err != nil {
			return err
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

func UpdateApp(nsName, appName string, updated domain.AppSpec) error {
	ns, err := domain.NewNamespace(nsName)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	current, found := ns.FindApp(appName)
	if !found {
		return ErrAppNotFound
	}
	if updated.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	if updated.Trigger.Type != "branch" && updated.Trigger.Type != "tag" {
		return fmt.Errorf("invalid trigger type: %s", updated.Trigger.Type)
	}
	if updated.Trigger.Rule == "" {
		return fmt.Errorf("trigger rule is required")
	}
	for _, env := range updated.Envs {
		if env.Key == "" {
			return fmt.Errorf("env key is empty")
		}
	}
	if len(updated.URLs) == 0 {
		return fmt.Errorf("service must have at least one url")
	}
	for _, u := range updated.URLs {
		if u.Host == "" || u.Port == "" {
			return fmt.Errorf("service url host and port are required")
		}
	}

	updated.Namespace = nsName
	updated.Name = appName
	// Deployments can become stale when their containers are removed outside
	// DockFlow. Do not let one stale entry prevent an otherwise valid app edit
	// from saving and regenerating the remaining Traefik configurations.
	updated.Deploy = make([]domain.AppDeploy, 0, len(current.Deploy))
	for _, deploy := range current.Deploy {
		containerID, err := docker.HasContainer(deploy.ContainerId)
		if err != nil {
			return err
		}
		if containerID == "" {
			if err := monitor.RemoveAppTraefikConfig(nsName, appName, deploy.Version); err != nil {
				return err
			}
			continue
		}
		deploy.ContainerId = containerID
		updated.Deploy = append(updated.Deploy, deploy)
	}
	updated.Secret = current.Secret
	if updated.Token == "" {
		updated.Token = current.Token
	}
	if err := domain.SaveApp(updated); err != nil {
		return err
	}
	return monitor.RefreshAppTraefik(updated)
}

func GetAppDeployLogs(nsName, appName, containerID, tail string) (string, error) {
	ns, err := domain.NewNamespace(nsName)
	if err != nil {
		return "", err
	}
	if ns == nil {
		return "", ErrNamespaceNotFound
	}
	app, found := ns.FindApp(appName)
	if !found {
		return "", ErrAppNotFound
	}
	_, found = lo.Find(app.Deploy, func(deploy domain.AppDeploy) bool {
		return deploy.ContainerId == containerID
	})
	if !found {
		return "", fmt.Errorf("deployment not found")
	}
	if tail == "" {
		tail = "200"
	}
	stream, err := docker.ContainerLogs(containerID, docker.ContainerLogOptions{Tail: tail})
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var output bytes.Buffer
	if _, err := stdcopy.StdCopy(&output, &output, stream); err != nil {
		return "", err
	}
	return output.String(), nil
}

type DeployAppOptions struct {
	Namespace string
	Name      string
	Branch    string
	Commit    string
	Tag       string
	ContainerEditOptions
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

func RemoveAppDeployment(nsName, appName, containerID string) error {
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

	deployIndex := -1
	version := ""
	for i, deploy := range app.Deploy {
		if deploy.ContainerId == containerID {
			deployIndex = i
			version = deploy.Version
			break
		}
	}
	if deployIndex == -1 {
		// The deployment may already have been removed from the app when a stale
		// container reference was reconciled. DELETE remains successful so its
		// historical deployment job can still be cleaned up.
		return nil
	}

	if existingID, err := docker.HasContainer(containerID); err != nil {
		return err
	} else if existingID != "" {
		if err := docker.StopContainer(existingID, nil); err != nil {
			return err
		}
		if err := docker.RemoveContainer(existingID, true); err != nil {
			return err
		}
	}
	if err := monitor.RemoveAppTraefikConfig(nsName, appName, version); err != nil {
		return err
	}

	app.Deploy = append(app.Deploy[:deployIndex], app.Deploy[deployIndex+1:]...)
	return domain.SaveApp(app)
}
