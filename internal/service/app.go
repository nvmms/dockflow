package service

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"dockflow/internal/service/git"
	"dockflow/internal/service/traefik"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNamespaceNotFound = errors.New("namespace not found")
)

//
// ==========================
// Deployer Struct
// ==========================
//

type AppDeployer struct {
	app *domain.AppSpec
	ns  *domain.Namespace
}

//
// ==========================
// Constructor
// ==========================
//

func NewAppDeployer(app *domain.AppSpec) (*AppDeployer, error) {
	ns, err := loadNamespace(app.Namespace)
	if err != nil {
		return nil, err
	}

	return &AppDeployer{
		app: app,
		ns:  ns,
	}, nil
}

//
// ==========================
// Deploy Entry
// ==========================
//

func (d *AppDeployer) Deploy(branch, commit, tag *string) error {

	// ---------- git ----------
	version, err := d.fetchAppCode(branch, commit, tag)
	if err != nil {
		return err
	}

	containerId, err := docker.HasContainer(d.app.Name + "_" + version)
	if err != nil {
		return err
	}

	if containerId != "" {
		err := docker.StopContainer(containerId, nil)
		if err != nil {
			return err
		}
		err = docker.RemoveContainer(containerId, true)
		if err != nil {
			return err
		}
	}

	// ---------- build ----------
	image, err := d.buildApp(version)
	if err != nil {
		return err
	}

	// ---------- run version ----------
	if err := d.deployVersion(image, version); err != nil {
		return err
	}

	// ---------- run latest ----------
	if err := d.deployVersion(image, "latest"); err != nil {
		return err
	}

	return nil
}

//
// ==========================
// Namespace
// ==========================
//

func loadNamespace(namespace string) (*domain.Namespace, error) {
	ns, err := domain.NewNamespace(namespace)
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, ErrNamespaceNotFound
	}
	return ns, nil
}

//
// ==========================
// Git
// ==========================
//

func (d *AppDeployer) fetchAppCode(
	branch, commit, tag *string,
) (string, error) {

	repoPath := filesystem.NamespaceDirName + "/" +
		d.app.Namespace + "/repo/" + d.app.Name

	opts := git.GitCloneOptions{
		RepoURL: d.app.Repo,
		DestDir: repoPath,
		Token:   d.app.Token,
		Branch:  branch,
		Commit:  commit,
		Tag:     tag,
	}

	return git.GetLatestCode(opts)
}

//
// ==========================
// Build
// ==========================
//

func (d *AppDeployer) buildApp(version string) (string, error) {

	repoPath := filesystem.NamespaceDirName + "/" +
		d.app.Namespace + "/repo/" + d.app.Name

	// var args map[string]*string
	// var err error
	// switch d.app.Platform {
	// case "java":
	// 	args, err = manifest.ParseJavaMaven(repoPath)
	// case "node-page":
	// 	args = d.app.BuildArg
	// case "go":
	// 	args = map[string]*string{}
	// case "python":
	// 	args = map[string]*string{}
	// default:
	// 	err = fmt.Errorf("build type [%s] not support", d.app.Platform)
	// }
	// if err != nil {
	// 	return "", err
	// }

	// ports := collectPorts(d.app.URLs)
	// args["APP_PORT"] = &ports

	image := fmt.Sprintf("%s:%s", d.app.Name, version)

	if err := docker.Build(repoPath, image); err != nil {
		return "", err
	}
	return image, nil
}

func collectPorts(urls []domain.AppURL) string {
	var ports []string
	for _, u := range urls {
		ports = append(ports, u.Port)
	}
	return strings.Join(ports, " ")
}

//
// ==========================
// Deploy Version
// ==========================
//

func (d *AppDeployer) deployVersion(
	image string,
	version string,
) error {

	containerId, err := d.runApp(image, version)
	if err != nil {
		return err
	}

	d.app.Deploy = append(d.app.Deploy, domain.AppDeploy{
		ContainerId: containerId,
		Version:     version,
		Url:         "/" + version,
	})

	return domain.SaveApp(*d.app)
}

//
// ==========================
// Run App
// ==========================
//

func (d *AppDeployer) runApp(image, version string) (string, error) {

	containerName := fmt.Sprintf("%s_%s", d.app.Name, version)

	// ---------- cleanup ----------
	if err := d.cleanupOldContainer(version); err != nil {
		return "", err
	}

	// ---------- run options ----------
	opts := docker.NewRunOptions(containerName, image)

	opts.WithCpu(d.app.CPU)
	opts.WithMemory(float64(d.app.Memory))

	for _, env := range d.app.Envs {
		opts.WithEnv(env.Key, env.Value)
	}

	opts.WithNetwork(traefik.TraefikNetwork)
	opts.WithNetwork(d.ns.Network)
	opts.WithLabel("dockflow.namespace", d.ns.Name)
	opts.WithLabel("dockflow.name", d.app.Name)
	opts.WithLabel("dockflow.version", version)

	// ---------- traefik ----------
	// opts.WithLabel("traefik.enable", "true")
	// opts.WithLabel("traefik.docker.network", traefik.TraefikNetwork)
	// opts.WithLabel("traefik.http.routers."+service+".rule", rule)

	// for i, url := range d.app.URLs {
	// 	service := fmt.Sprintf("%s_%s_%d", d.app.Name, version, i)
	// 	rule := buildTraefikRule(url.Host, version)

	// 	opts.WithLabel("traefik.http.routers."+service+".rule", rule)
	// 	opts.WithLabel("traefik.http.routers."+service+".entrypoints", "https")
	// 	opts.WithLabel("traefik.http.routers."+service+".tls", "true")
	// 	opts.WithLabel("traefik.http.routers."+service+".tls.certresolver", "le")
	// 	opts.WithLabel("traefik.http.routers."+service+".service", service)
	// 	opts.WithLabel(
	// 		"traefik.http.services."+service+".loadbalancer.server.port",
	// 		url.Port,
	// 	)
	// }

	return docker.RunContainer(opts)
}

//
// ==========================
// Cleanup
// ==========================
//

func (d *AppDeployer) cleanupOldContainer(version string) error {

	for i := len(d.app.Deploy) - 1; i >= 0; i-- {
		deploy := d.app.Deploy[i]

		if deploy.Version != version {
			continue
		}

		containerId, err := docker.HasContainer(deploy.ContainerId)
		if err != nil {
			return err
		}

		if containerId != "" {
			if err := docker.StopContainer(containerId, nil); err != nil {
				return err
			}
			if err := docker.RemoveContainer(containerId, true); err != nil {
				return err
			}
		}

		d.app.Deploy = append(d.app.Deploy[:i], d.app.Deploy[i+1:]...)
	}

	return domain.SaveApp(*d.app)
}

//
// ==========================
// Traefik Rule
// ==========================
//

func buildTraefikRule(host, version string) string {
	parts := strings.Split(host, "/")

	rule := "Host(`" + parts[0] + "`)"
	for _, p := range parts[1:] {
		rule += " && Path(`/" + p + "`)"
	}
	if version != "latest" {
		rule += " && Path(`/" + version + "`)"
	}
	return rule
}
