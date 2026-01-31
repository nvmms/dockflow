package service

import (
	"dockflow/internal/domain"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
	"dockflow/internal/service/git"
	"dockflow/internal/service/manifest"
	"dockflow/internal/service/traefik"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNamespaceNotFound = errors.New("namespace not found")
	ErrDeployType        = errors.New("deploy error, only branch、tag support")
	ErrDeployBranchBlank = errors.New("deploy with branch mode, branch can't be blank")
	ErrDeployTagBlank    = errors.New("deploy with tag mode, tag can't be blank")
)

func DeployApp(app domain.AppSpec, branch *string, commit *string, tag *string) error {

	// ---------- namespace ----------
	ns, err := filesystem.LoadNamespace(app.Namespace)
	if err != nil {
		return err
	}
	if ns == nil {
		return ErrNamespaceNotFound
	}

	// ---------- git ----------
	appRepoPath := filesystem.NamespaceDirName + "/" + app.Namespace + "/repo/" + app.Name

	gitOpt := git.GitCloneOptions{
		RepoURL: app.Repo,
		DestDir: appRepoPath,
		Token:   app.Token,

		// 这里只是“版本选择输入”
		Branch: branch,
		Commit: commit,
		Tag:    tag,
	}

	// GetLatestCode 内部已经保证：
	// - branch/tag → commit
	// - checkout 到确定 commit
	version, err := git.GetLatestCode(gitOpt)
	if err != nil {
		return err
	}

	// ---------- build ----------

	ports := []string{}
	for _, url := range app.URLs {
		ports = append(ports, url.Port)
	}
	portsSrt := strings.Join(ports, " ")

	args, err := manifest.ParseJavaMaven(appRepoPath)
	if err != nil {
		return err
	}
	args["APP_PORT"] = &portsSrt

	deployImage := fmt.Sprintf("%s:%s", app.Name, version)
	if err := docker.Build(appRepoPath, deployImage, app.Platform, args); err != nil {
		print("build image error: \n")
		return err
	}

	// 检测 指定version是否已经存在
	containerName := fmt.Sprintf("%s_%s", app.Name, version)
	for _, deploy := range app.Deploy {
		if deploy.Version == version {
			isExist, err := docker.HasContainer(deploy.ContainerId)
			if err != nil {
				return err
			}
			if isExist {
				err = docker.StopContainer(deploy.ContainerId, nil)
				if err != nil {
					return err
				}
				err = docker.RemoveContainer(deploy.ContainerId, true)
				if err != nil {
					return err
				}
			}
		}
	}

	// ---------- run ----------
	opts := docker.NewRunOptions(containerName, deployImage)
	opts.WithCpu(app.CPU)
	opts.WithMemory(float64(app.Memory))

	for _, env := range app.Envs {
		opts.WithEnv(env.Key, env.Value)
	}

	// network
	opts.WithNetwork(traefik.TraefikNetwork)
	opts.WithNetwork(ns.Network)

	// ---------- traefik ----------
	opts.WithLabel("traefik.enable", "true")
	opts.WithLabel("traefik.docker.network", traefik.TraefikNetwork)

	for index, url := range app.URLs {
		serviceName := fmt.Sprintf("%s_%d", app.Name, index)

		// Host + Path 规则
		urlSlice := strings.Split(url.Host, "/")
		rule := "Host(`" + urlSlice[0] + "`)"
		if len(urlSlice) > 1 {
			for _, path := range urlSlice[1:] {
				rule += " && Path(`/" + path + "`)"
			}
		}

		rule += " && Path(`/" + version + "`)"

		opts.WithLabel("traefik.http.routers."+serviceName+".rule", rule)
		opts.WithLabel("traefik.http.routers."+serviceName+".entrypoints", "https")
		opts.WithLabel("traefik.http.routers."+serviceName+".tls", "true")
		opts.WithLabel("traefik.http.routers."+serviceName+".tls.certresolver", "le")
		opts.WithLabel("traefik.http.routers."+serviceName+".service", serviceName)
		opts.WithLabel(
			"traefik.http.services."+serviceName+".loadbalancer.server.port",
			url.Port,
		)
	}

	containerId, err := docker.RunContainer(opts)
	if err != nil {
		return err
	}

	deploy := domain.AppDeploy{
		ContainerId: containerId,
		Version:     version,
		Url:         "/" + version,
	}

	app.Deploy = append(app.Deploy, deploy)
	filesystem.SaveApp(app)

	return nil
}
