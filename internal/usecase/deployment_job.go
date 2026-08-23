package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"dockflow/internal/domain"
	"dockflow/internal/service"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/filesystem"
)

type DeploymentJob struct {
	ID            string     `json:"id"`
	Namespace     string     `json:"namespace"`
	App           string     `json:"app"`
	SourceType    string     `json:"sourceType,omitempty"`
	SourceRef     string     `json:"sourceRef,omitempty"`
	Commit        string     `json:"commit,omitempty"`
	Version       string     `json:"version,omitempty"`
	Status        string     `json:"status"`
	ContainerID   string     `json:"containerId,omitempty"`
	IP            []string   `json:"ip,omitempty"`
	Logs          string     `json:"logs"`
	Error         string     `json:"error,omitempty"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Deleted       bool       `json:"deleted,omitempty"`
	RestartPolicy string     `json:"restart_policy,omitempty"`
	LogDriver     string     `json:"log_driver,omitempty"`
	LogMaxSize    string     `json:"log_max_size,omitempty"`
	LogMaxFile    int        `json:"log_max_file,omitempty"`
	NeedsRecreate bool       `json:"needs_recreate,omitempty"`
	SLSProject    string     `json:"sls_project,omitempty"`
	SLSLogstore   string     `json:"sls_logstore,omitempty"`
	SLSEndpoint   string     `json:"sls_endpoint,omitempty"`
	SLSConfigName string     `json:"sls_config_name,omitempty"`
}

type deploymentJobState struct {
	DeploymentJob
	mu sync.RWMutex
}

func (j *deploymentJobState) Write(p []byte) (int, error) {
	j.mu.Lock()
	j.Logs += string(p)
	j.mu.Unlock()
	return len(p), nil
}

func (j *deploymentJobState) snapshot() DeploymentJob {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.DeploymentJob
}

var deploymentJobs = struct {
	sync.RWMutex
	jobs   map[string]*deploymentJobState
	active map[string]string
}{jobs: map[string]*deploymentJobState{}, active: map[string]string{}}

var (
	deploymentJobsLoadOnce sync.Once
	deploymentJobsLoadErr  error
	deploymentJobsStoreMu  sync.Mutex
)

func StartDeployApp(opt DeployAppOptions) (DeploymentJob, error) {
	if err := ensureDeploymentJobsLoaded(); err != nil {
		return DeploymentJob{}, err
	}
	namespace, err := domain.NewNamespace(opt.Namespace)
	if err != nil {
		return DeploymentJob{}, err
	}
	if namespace == nil {
		return DeploymentJob{}, ErrNamespaceNotFound
	}
	app, found := namespace.FindApp(opt.Name)
	if !found {
		return DeploymentJob{}, ErrAppNotFound
	}
	runtimeConfig, normalized, err := deploymentRuntimeConfig(opt.ContainerEditOptions)
	if err != nil {
		return DeploymentJob{}, err
	}

	key := opt.Namespace + "/" + opt.Name
	deploymentJobs.Lock()
	if activeID := deploymentJobs.active[key]; activeID != "" {
		deploymentJobs.Unlock()
		return DeploymentJob{}, fmt.Errorf("app deployment already running: %s", activeID)
	}
	idBytes := make([]byte, 8)
	if _, err := rand.Read(idBytes); err != nil {
		deploymentJobs.Unlock()
		return DeploymentJob{}, err
	}
	id := hex.EncodeToString(idBytes)
	sourceType, sourceRef := deploymentSource(opt)
	job := &deploymentJobState{DeploymentJob: DeploymentJob{
		ID: id, Namespace: opt.Namespace, App: opt.Name,
		SourceType: sourceType, SourceRef: sourceRef, Commit: opt.Commit,
		Status: "running", StartedAt: time.Now(),
		RestartPolicy: normalized.RestartPolicy, LogDriver: normalized.LogDriver,
		LogMaxSize: normalized.LogMaxSize, LogMaxFile: normalized.LogMaxFile,
		SLSProject: normalized.SLSProject, SLSLogstore: normalized.SLSLogstore,
		SLSEndpoint: normalized.SLSEndpoint, SLSConfigName: normalized.SLSConfigName,
	}}
	deploymentJobs.jobs[id] = job
	deploymentJobs.active[key] = id
	deploymentJobs.Unlock()
	_ = persistDeploymentJobs()

	go func() {
		_, _ = fmt.Fprintf(job, "[%s] deployment started\n", time.Now().Format(time.RFC3339))
		deployer, err := service.NewAppDeployerWithLog(&app, job)
		if err == nil {
			deployer.WithRuntimeConfig(runtimeConfig)
			err = deployer.Deploy(&opt.Branch, &opt.Commit, &opt.Tag)
		}
		finishedAt := time.Now()
		job.mu.Lock()
		job.FinishedAt = &finishedAt
		if err != nil {
			job.Status = "failed"
			job.Error = err.Error()
			job.Logs += "\n[failed] " + err.Error() + "\n"
		} else {
			job.Status = "success"
			job.Logs += "\n[success] deployment completed\n"
		}
		job.mu.Unlock()
		if err == nil {
			populateDeploymentContainer(job, opt.Namespace, opt.Name)
		}
		deploymentJobs.Lock()
		delete(deploymentJobs.active, key)
		deploymentJobs.Unlock()
		_ = persistDeploymentJobs()
	}()

	return job.snapshot(), nil
}

func deploymentRuntimeConfig(edit ContainerEditOptions) (service.DeploymentRuntimeConfig, ContainerEditOptions, error) {
	policy, err := normalizeRestartPolicy(edit.RestartPolicy)
	if err != nil {
		return service.DeploymentRuntimeConfig{}, edit, err
	}
	if edit.LogDriver == "" {
		edit.LogDriver = "local"
	}
	if edit.LogDriver == "aliyun-sls" && (edit.SLSProject == "" || edit.SLSLogstore == "" || edit.SLSEndpoint == "" || edit.SLSConfigName == "") {
		return service.DeploymentRuntimeConfig{}, edit, fmt.Errorf("aliyun sls endpoint, project, logstore and config name are required")
	}
	logConfig, err := normalizeLogConfig(edit.LogDriver, edit.LogMaxSize, edit.LogMaxFile)
	if err != nil {
		return service.DeploymentRuntimeConfig{}, edit, err
	}
	edit.RestartPolicy = string(policy)
	edit.LogMaxSize = logConfig.Config["max-size"]
	edit.LogMaxFile, _ = strconv.Atoi(logConfig.Config["max-file"])
	return service.DeploymentRuntimeConfig{
		RestartPolicy: policy, LogConfig: logConfig, LogDriver: edit.LogDriver,
		LogMaxSize: edit.LogMaxSize, LogMaxFile: edit.LogMaxFile,
		SLSProject: edit.SLSProject, SLSLogstore: edit.SLSLogstore,
		SLSEndpoint: edit.SLSEndpoint, SLSConfigName: edit.SLSConfigName,
		Labels: slsContainerLabels(edit),
	}, edit, nil
}

func deploymentSource(opt DeployAppOptions) (string, string) {
	if opt.Tag != "" {
		return "tag", opt.Tag
	}
	if opt.Branch != "" {
		return "branch", opt.Branch
	}
	if opt.Commit != "" {
		return "commit", opt.Commit
	}
	return "", ""
}

func GetDeploymentJob(nsName, appName, id string) (DeploymentJob, error) {
	if err := ensureDeploymentJobsLoaded(); err != nil {
		return DeploymentJob{}, err
	}
	deploymentJobs.RLock()
	job := deploymentJobs.jobs[id]
	deploymentJobs.RUnlock()
	if job == nil {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	snapshot := job.snapshot()
	if snapshot.Deleted || snapshot.Namespace != nsName || snapshot.App != appName {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	refreshDeploymentRuntime(&snapshot)
	return snapshot, nil
}

func ListDeploymentJobs(nsName, appName string) []DeploymentJob {
	if ensureDeploymentJobsLoaded() != nil {
		return []DeploymentJob{}
	}
	reconcileAppDeployments(nsName, appName)
	deploymentJobs.RLock()
	defer deploymentJobs.RUnlock()
	result := make([]DeploymentJob, 0)
	for _, job := range deploymentJobs.jobs {
		if !job.Deleted && job.Namespace == nsName && job.App == appName {
			snapshot := job.snapshot()
			refreshDeploymentRuntime(&snapshot)
			result = append(result, snapshot)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].StartedAt.After(result[j].StartedAt) })
	return result
}

func populateDeploymentContainer(job *deploymentJobState, nsName, appName string) {
	ns, err := domain.NewNamespace(nsName)
	if err != nil || ns == nil {
		return
	}
	app, found := ns.FindApp(appName)
	if !found || len(app.Deploy) == 0 {
		return
	}
	deploy := app.Deploy[len(app.Deploy)-1]
	job.mu.Lock()
	job.ContainerID = deploy.ContainerId
	job.Version = deploy.Version
	if job.Commit == "" {
		job.Commit = deploy.Version
	}
	job.mu.Unlock()

	snapshot := job.snapshot()
	refreshDeploymentRuntime(&snapshot)
	job.mu.Lock()
	job.IP = snapshot.IP
	job.Status = snapshot.Status
	job.mu.Unlock()
}

func refreshDeploymentRuntime(job *DeploymentJob) {
	if job.Status == "succeeded" { // Compatibility with jobs created by older versions.
		job.Status = "success"
	}
	if job.Status != "success" || job.ContainerID == "" {
		return
	}
	inspect, err := docker.InspectContainer(job.ContainerID)
	if err != nil {
		return
	}
	actualRestart, actualDriver, actualSize, actualFile, slsEnabled := "", "", "", 0, false
	if inspect.HostConfig != nil {
		actualRestart = string(inspect.HostConfig.RestartPolicy.Name)
		actualDriver = inspect.HostConfig.LogConfig.Type
		actualSize = inspect.HostConfig.LogConfig.Config["max-size"]
		actualFile, _ = strconv.Atoi(inspect.HostConfig.LogConfig.Config["max-file"])
	}
	if inspect.Config != nil {
		slsEnabled = inspect.Config.Labels["dockflow.sls.enabled"] == "true"
	}
	desiredDriver := job.LogDriver
	if desiredDriver == "aliyun-sls" {
		desiredDriver = "json-file"
	}
	job.NeedsRecreate = job.LogDriver != "" && (job.RestartPolicy != actualRestart || desiredDriver != actualDriver || job.LogMaxSize != actualSize || job.LogMaxFile != actualFile || (job.LogDriver == "aliyun-sls") != slsEnabled)
	if inspect.State == nil || !inspect.State.Running {
		job.Status = "stopped"
		return
	}
	job.IP = job.IP[:0]
	if inspect.NetworkSettings != nil {
		for _, network := range inspect.NetworkSettings.Networks {
			if network != nil && network.IPAddress != "" {
				job.IP = append(job.IP, network.IPAddress)
			}
		}
	}
}

func UpdateDeploymentJobConfig(nsName, appName, id string, edit ContainerEditOptions) (DeploymentJob, error) {
	if err := ensureDeploymentJobsLoaded(); err != nil {
		return DeploymentJob{}, err
	}
	runtimeConfig, normalized, err := deploymentRuntimeConfig(edit)
	if err != nil {
		return DeploymentJob{}, err
	}
	deploymentJobs.RLock()
	job := deploymentJobs.jobs[id]
	deploymentJobs.RUnlock()
	if job == nil {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	snapshot := job.snapshot()
	if snapshot.Namespace != nsName || snapshot.App != appName || snapshot.Deleted || snapshot.Status == "running" || snapshot.ContainerID == "" {
		return DeploymentJob{}, fmt.Errorf("deployment job cannot be edited")
	}
	newContainerID := snapshot.ContainerID
	if normalized.ApplyNow {
		newContainerID, err = docker.RecreateContainer(snapshot.ContainerID, runtimeConfig.RestartPolicy, runtimeConfig.LogConfig, runtimeConfig.Labels)
		if err != nil {
			return DeploymentJob{}, err
		}
		ns, loadErr := domain.NewNamespace(nsName)
		if loadErr != nil {
			return DeploymentJob{}, loadErr
		}
		if ns == nil {
			return DeploymentJob{}, ErrNamespaceNotFound
		}
		app, found := ns.FindApp(appName)
		if !found {
			return DeploymentJob{}, ErrAppNotFound
		}
		for index := range app.Deploy {
			if app.Deploy[index].ContainerId == snapshot.ContainerID {
				app.Deploy[index].ContainerId = newContainerID
				app.Deploy[index].RestartPolicy = normalized.RestartPolicy
				app.Deploy[index].LogDriver = normalized.LogDriver
				app.Deploy[index].LogMaxSize = normalized.LogMaxSize
				app.Deploy[index].LogMaxFile = normalized.LogMaxFile
				app.Deploy[index].SLSProject = normalized.SLSProject
				app.Deploy[index].SLSLogstore = normalized.SLSLogstore
				app.Deploy[index].SLSEndpoint = normalized.SLSEndpoint
				app.Deploy[index].SLSConfigName = normalized.SLSConfigName
				break
			}
		}
		if err := domain.SaveApp(app); err != nil {
			return DeploymentJob{}, err
		}
	}
	job.mu.Lock()
	job.ContainerID = newContainerID
	job.RestartPolicy = normalized.RestartPolicy
	job.LogDriver = normalized.LogDriver
	job.LogMaxSize = normalized.LogMaxSize
	job.LogMaxFile = normalized.LogMaxFile
	job.SLSProject = normalized.SLSProject
	job.SLSLogstore = normalized.SLSLogstore
	job.SLSEndpoint = normalized.SLSEndpoint
	job.SLSConfigName = normalized.SLSConfigName
	job.NeedsRecreate = !normalized.ApplyNow
	job.mu.Unlock()
	if err := persistDeploymentJobs(); err != nil {
		return DeploymentJob{}, err
	}
	result := job.snapshot()
	refreshDeploymentRuntime(&result)
	return result, nil
}

func DeleteDeploymentJob(nsName, appName, id string) error {
	if err := ensureDeploymentJobsLoaded(); err != nil {
		return err
	}
	deploymentJobs.Lock()
	job := deploymentJobs.jobs[id]
	if job == nil {
		deploymentJobs.Unlock()
		return fmt.Errorf("deployment job not found")
	}
	snapshot := job.snapshot()
	if snapshot.Namespace != nsName || snapshot.App != appName {
		deploymentJobs.Unlock()
		return fmt.Errorf("deployment job not found")
	}
	if snapshot.Status == "running" {
		deploymentJobs.Unlock()
		return fmt.Errorf("deployment job is running")
	}
	deploymentJobs.Unlock()
	if snapshot.ContainerID != "" {
		if err := RemoveAppDeployment(nsName, appName, snapshot.ContainerID); err != nil {
			return err
		}
	}
	deploymentJobs.Lock()
	job.mu.Lock()
	job.Deleted = true
	job.Logs = ""
	job.mu.Unlock()
	deploymentJobs.Unlock()
	return persistDeploymentJobs()
}

func RestartDeploymentJob(nsName, appName, id string) (DeploymentJob, error) {
	if err := ensureDeploymentJobsLoaded(); err != nil {
		return DeploymentJob{}, err
	}
	deploymentJobs.RLock()
	job := deploymentJobs.jobs[id]
	deploymentJobs.RUnlock()
	if job == nil {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	snapshot := job.snapshot()
	if snapshot.Deleted || snapshot.Namespace != nsName || snapshot.App != appName {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	if snapshot.ContainerID == "" {
		return DeploymentJob{}, fmt.Errorf("deployment has no container to restart")
	}
	if err := docker.RestartContainer(snapshot.ContainerID, nil); err != nil {
		return DeploymentJob{}, err
	}
	snapshot.Status = "success"
	refreshDeploymentRuntime(&snapshot)
	job.mu.Lock()
	job.Status = snapshot.Status
	job.IP = snapshot.IP
	job.mu.Unlock()
	_ = persistDeploymentJobs()
	return snapshot, nil
}

func ensureDeploymentJobsLoaded() error {
	deploymentJobsLoadOnce.Do(func() {
		data, err := os.ReadFile(filesystem.DeploymentJobsPath)
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			deploymentJobsLoadErr = err
			return
		}
		var stored []DeploymentJob
		if err := json.Unmarshal(data, &stored); err != nil {
			deploymentJobsLoadErr = err
			return
		}
		now := time.Now()
		deploymentJobs.Lock()
		defer deploymentJobs.Unlock()
		for _, item := range stored {
			if item.Status == "running" {
				item.Status = "failed"
				item.Error = "dockflow restarted before deployment completed"
				item.FinishedAt = &now
			}
			copy := item
			deploymentJobs.jobs[item.ID] = &deploymentJobState{DeploymentJob: copy}
		}
	})
	return deploymentJobsLoadErr
}

func persistDeploymentJobs() error {
	deploymentJobsStoreMu.Lock()
	defer deploymentJobsStoreMu.Unlock()
	deploymentJobs.RLock()
	stored := make([]DeploymentJob, 0, len(deploymentJobs.jobs))
	for _, job := range deploymentJobs.jobs {
		stored = append(stored, job.snapshot())
	}
	deploymentJobs.RUnlock()
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filesystem.BaseDirName, 0755); err != nil {
		return err
	}
	tmp := filesystem.DeploymentJobsPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filesystem.DeploymentJobsPath)
}

func reconcileAppDeployments(nsName, appName string) {
	ns, err := domain.NewNamespace(nsName)
	if err != nil || ns == nil {
		return
	}
	app, found := ns.FindApp(appName)
	if !found {
		return
	}
	changed := false
	for _, deploy := range app.Deploy {
		deploymentJobs.Lock()
		var existing *deploymentJobState
		for _, job := range deploymentJobs.jobs {
			if job.ContainerID == deploy.ContainerId {
				existing = job
				break
			}
		}
		if existing != nil {
			existing.mu.Lock()
			if existing.Version == "" {
				existing.Version = deploy.Version
				existing.Commit = deploy.Version
				if existing.SourceType == "" {
					existing.SourceType = "existing"
					existing.SourceRef = "历史部署"
				}
				changed = true
			}
			existing.mu.Unlock()
		} else {
			id := "existing-" + deploy.ContainerId
			if len(id) > 25 {
				id = id[:25]
			}
			startedAt := time.Time{}
			if inspect, inspectErr := docker.InspectContainer(deploy.ContainerId); inspectErr == nil {
				startedAt, _ = time.Parse(time.RFC3339Nano, inspect.Created)
			}
			deploymentJobs.jobs[id] = &deploymentJobState{DeploymentJob: DeploymentJob{
				ID: id, Namespace: nsName, App: appName, Status: "success",
				SourceType: "existing", SourceRef: "历史部署", Commit: deploy.Version, Version: deploy.Version,
				ContainerID: deploy.ContainerId, StartedAt: startedAt,
			}}
			changed = true
		}
		deploymentJobs.Unlock()
	}
	if changed {
		_ = persistDeploymentJobs()
	}
}
