package usecase

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"dockflow/internal/domain"
	"dockflow/internal/service"
)

type DeploymentJob struct {
	ID         string     `json:"id"`
	Namespace  string     `json:"namespace"`
	App        string     `json:"app"`
	Status     string     `json:"status"`
	Logs       string     `json:"logs"`
	Error      string     `json:"error,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
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

func StartDeployApp(opt DeployAppOptions) (DeploymentJob, error) {
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
	job := &deploymentJobState{DeploymentJob: DeploymentJob{
		ID: id, Namespace: opt.Namespace, App: opt.Name,
		Status: "running", StartedAt: time.Now(),
	}}
	deploymentJobs.jobs[id] = job
	deploymentJobs.active[key] = id
	deploymentJobs.Unlock()

	go func() {
		_, _ = fmt.Fprintf(job, "[%s] deployment started\n", time.Now().Format(time.RFC3339))
		deployer, err := service.NewAppDeployerWithLog(&app, job)
		if err == nil {
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
			job.Status = "succeeded"
			job.Logs += "\n[success] deployment completed\n"
		}
		job.mu.Unlock()
		deploymentJobs.Lock()
		delete(deploymentJobs.active, key)
		deploymentJobs.Unlock()
	}()

	return job.snapshot(), nil
}

func GetDeploymentJob(nsName, appName, id string) (DeploymentJob, error) {
	deploymentJobs.RLock()
	job := deploymentJobs.jobs[id]
	deploymentJobs.RUnlock()
	if job == nil {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	snapshot := job.snapshot()
	if snapshot.Namespace != nsName || snapshot.App != appName {
		return DeploymentJob{}, fmt.Errorf("deployment job not found")
	}
	return snapshot, nil
}

func ListDeploymentJobs(nsName, appName string) []DeploymentJob {
	deploymentJobs.RLock()
	defer deploymentJobs.RUnlock()
	result := make([]DeploymentJob, 0)
	for _, job := range deploymentJobs.jobs {
		if job.Namespace == nsName && job.App == appName {
			result = append(result, job.snapshot())
		}
	}
	return result
}

func ListNamespaceDeploymentJobs(nsName string) []DeploymentJob {
	deploymentJobs.RLock()
	defer deploymentJobs.RUnlock()
	result := make([]DeploymentJob, 0)
	for _, job := range deploymentJobs.jobs {
		if job.Namespace == nsName {
			result = append(result, job.snapshot())
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})
	return result
}
