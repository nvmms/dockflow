package service

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"dockflow/internal/service/git"
	"dockflow/internal/service/traefik"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
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
	app     *domain.AppSpec
	ns      *domain.Namespace
	log     io.Writer
	runtime DeploymentRuntimeConfig
}

type DeploymentRuntimeConfig struct {
	Domain        string
	RestartPolicy container.RestartPolicyMode
	LogConfig     container.LogConfig
	LogDriver     string
	LogMaxSize    string
	LogMaxFile    int
	SLSProject    string
	SLSLogstore   string
	SLSEndpoint   string
	SLSConfigName string
	Labels        map[string]string
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
		log: io.Discard,
		runtime: DeploymentRuntimeConfig{
			RestartPolicy: container.RestartPolicyUnlessStopped,
			LogConfig:     container.LogConfig{Type: "local", Config: map[string]string{"max-size": "10m", "max-file": "3"}},
			LogDriver:     "local", LogMaxSize: "10m", LogMaxFile: 3,
		},
	}, nil
}

func (d *AppDeployer) WithRuntimeConfig(config DeploymentRuntimeConfig) {
	d.runtime = config
}

func NewAppDeployerWithLog(app *domain.AppSpec, output io.Writer) (*AppDeployer, error) {
	deployer, err := NewAppDeployer(app)
	if err != nil {
		return nil, err
	}
	if output != nil {
		deployer.log = output
	}
	return deployer, nil
}

func (d *AppDeployer) logf(format string, args ...interface{}) {
	fmt.Fprintf(d.log, format+"\n", args...)
}

//
// ==========================
// Deploy Entry
// ==========================
//

func (d *AppDeployer) Deploy(branch, commit, tag *string) error {

	// ---------- git ----------
	d.logf("[source] resolving and fetching %s", d.app.Repo)
	version, err := d.fetchAppCode(branch, commit, tag)
	if err != nil {
		return fmt.Errorf("source checkout failed: %w", err)
	}
	d.logf("[source] checked out commit %s", version)

	containerName := docker.ResourceContainerName(d.app.Namespace, "app", d.app.Name, version)
	containerId, err := docker.HasContainer(containerName)
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
	d.logf("[build] building image %s:%s", d.app.Name, version)
	image, err := d.buildApp(version)
	if err != nil {
		return fmt.Errorf("image build failed: %w", err)
	}
	d.logf("[build] image ready: %s", image)

	// ---------- run version ----------
	d.logf("[runtime] starting version container")
	if err := d.deployVersion(image, version); err != nil {
		return err
	}

	// ---------- run latest ----------
	d.logf("[runtime] starting latest container")
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
		RepoURL:  d.app.Repo,
		DestDir:  repoPath,
		Token:    d.app.Token,
		Branch:   branch,
		Commit:   commit,
		Tag:      tag,
		Progress: d.log,
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

	// Docker build arguments are the portable way to expose application
	// configuration during image construction. Dockerfiles can consume them
	// with `ARG KEY` (and optionally promote them with `ENV KEY=$KEY`).
	buildArgs := appEnvBuildArgs(d.app.Envs)
	if err := docker.Build(repoPath, image, d.log, buildArgs); err != nil {
		return "", err
	}
	return image, nil
}

func appEnvBuildArgs(envs []domain.Env) map[string]*string {
	buildArgs := make(map[string]*string, len(envs))
	for _, env := range envs {
		value := env.Value
		buildArgs[env.Key] = &value
	}
	return buildArgs
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
		ContainerId: containerId, Version: version, Url: "/" + version, Domain: d.runtime.Domain,
		RestartPolicy: string(d.runtime.RestartPolicy), LogDriver: d.runtime.LogDriver,
		LogMaxSize: d.runtime.LogMaxSize, LogMaxFile: d.runtime.LogMaxFile,
		SLSProject: d.runtime.SLSProject, SLSLogstore: d.runtime.SLSLogstore,
		SLSEndpoint: d.runtime.SLSEndpoint, SLSConfigName: d.runtime.SLSConfigName,
	})

	return domain.SaveApp(*d.app)
}

//
// ==========================
// Run App
// ==========================
//

func (d *AppDeployer) runApp(image, version string) (string, error) {

	containerName := docker.ResourceContainerName(d.app.Namespace, "app", d.app.Name, version)

	// ---------- cleanup ----------
	if err := d.cleanupOldContainer(version); err != nil {
		return "", err
	}

	// ---------- run options ----------
	opts := docker.NewRunOptions(containerName, image)
	opts.WithRestart(d.runtime.RestartPolicy)
	opts.WithLogging(d.runtime.LogConfig.Type, d.runtime.LogConfig.Config["max-size"], d.runtime.LogMaxFile)

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
	for key, value := range d.runtime.Labels {
		opts.WithLabel(key, value)
	}

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
