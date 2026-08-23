package webapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"dockflow/internal/domain"
	"dockflow/internal/usecase"
)

const maxJSONBody = 1 << 20

// NewHandler exposes every implemented CLI operation as a versioned JSON API.
func NewHandler() http.Handler {
	return newHandler(authenticateSystemUser)
}

func newHandler(check credentialChecker) http.Handler {
	mux := http.NewServeMux()
	sessions := newSessionStore(check)
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("POST /api/v1/auth/login", sessions.login)
	mux.HandleFunc("POST /api/v1/auth/logout", sessions.logout)
	mux.HandleFunc("GET /api/v1/auth/session", func(w http.ResponseWriter, r *http.Request) {
		if username, ok := sessions.current(r); ok {
			writeJSON(w, http.StatusOK, map[string]string{"username": username})
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
	})
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/init", initDockflow)
	protected.HandleFunc("/api/v1/namespaces", namespaces)
	protected.HandleFunc("/api/v1/namespaces/", namespaceResource)
	protected.HandleFunc("/api/v1/repositories", repositories)
	protected.HandleFunc("/api/v1/repositories/", repositoryResource)
	protected.HandleFunc("GET /api/v1/openapi.json", serveOpenAPI)
	mux.Handle("/api/", sessions.require(protected))
	mux.Handle("/", spaHandler())
	return recoverer(mux)
}

func initDockflow(w http.ResponseWriter, _ *http.Request) {
	if err := usecase.Init(); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func namespaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 200, usecase.ListNamespace())
	case http.MethodPost:
		var in struct {
			Name string `json:"name"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
		if strings.TrimSpace(in.Name) == "" {
			writeBadRequest(w, "name is required")
			return
		}
		ns, err := usecase.CreateNamespace(in.Name)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, ns)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func namespaceResource(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/v1/namespaces/"))
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}
	ns := parts[0]
	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			result, err := usecase.InspectNamespace(ns)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 200, result)
		case http.MethodDelete:
			if err := usecase.RemoveNamespace(ns); err != nil {
				writeError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			methodNotAllowed(w, "GET, DELETE")
		}
		return
	}
	switch parts[1] {
	case "apps":
		handleApps(w, r, ns, parts[2:])
	case "databases":
		handleDatabases(w, r, ns, parts[2:])
	case "redis":
		handleRedis(w, r, ns, parts[2:])
	default:
		http.NotFound(w, r)
	}
}

func handleApps(w http.ResponseWriter, r *http.Request, ns string, rest []string) {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			v, err := usecase.ListApp(ns)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 200, v)
		case http.MethodPost:
			var in domain.AppSpec
			if err := decodeJSON(w, r, &in); err != nil {
				return
			}
			in.Namespace = ns
			if in.CPU == 0 {
				in.CPU = 1
			}
			if in.Memory == 0 {
				in.Memory = 1
			}
			if in.Trigger.Type == "" {
				in.Trigger.Type = "branch"
			}
			if in.Trigger.Rule == "" {
				in.Trigger.Rule = "main"
			}
			if err := usecase.CreateApp(in); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 201, in)
		default:
			methodNotAllowed(w, "GET, POST")
		}
		return
	}
	name := rest[0]
	if len(rest) == 3 && rest[1] == "deploy" && rest[2] == "jobs" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, usecase.ListDeploymentJobs(ns, name))
		return
	}
	if len(rest) == 4 && rest[1] == "deploy" && rest[2] == "jobs" && r.Method == http.MethodGet {
		job, err := usecase.GetDeploymentJob(ns, name, rest[3])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	if len(rest) == 5 && rest[1] == "deploy" && rest[2] == "jobs" && rest[4] == "logs" && r.Method == http.MethodGet {
		streamDeploymentLogs(w, r, ns, name, rest[3])
		return
	}
	if len(rest) == 4 && rest[1] == "deploy" && rest[2] == "jobs" && r.Method == http.MethodDelete {
		if err := usecase.DeleteDeploymentJob(ns, name, rest[3]); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if len(rest) == 4 && rest[1] == "deploy" && rest[2] == "jobs" && r.Method == http.MethodPut {
		var in struct {
			RestartPolicy string `json:"restart_policy"`
			LogDriver     string `json:"log_driver"`
			LogMaxSize    string `json:"log_max_size"`
			LogMaxFile    int    `json:"log_max_file"`
			ApplyNow      bool   `json:"apply_now"`
			SLSProject    string `json:"sls_project"`
			SLSLogstore   string `json:"sls_logstore"`
			SLSEndpoint   string `json:"sls_endpoint"`
			SLSConfigName string `json:"sls_config_name"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
		job, err := usecase.UpdateDeploymentJobConfig(ns, name, rest[3], usecase.ContainerEditOptions{RestartPolicy: in.RestartPolicy, LogDriver: in.LogDriver, LogMaxSize: in.LogMaxSize, LogMaxFile: in.LogMaxFile, ApplyNow: in.ApplyNow, SLSProject: in.SLSProject, SLSLogstore: in.SLSLogstore, SLSEndpoint: in.SLSEndpoint, SLSConfigName: in.SLSConfigName})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	if len(rest) == 5 && rest[1] == "deploy" && rest[2] == "jobs" && rest[4] == "restart" && r.Method == http.MethodPost {
		job, err := usecase.RestartDeploymentJob(ns, name, rest[3])
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, job)
		return
	}
	if len(rest) == 4 && rest[1] == "deploy" && rest[3] == "logs" && r.Method == http.MethodGet {
		tail := r.URL.Query().Get("tail")
		if tail != "" {
			value, err := strconv.Atoi(tail)
			if err != nil || value < 1 || value > 5000 {
				writeError(w, fmt.Errorf("tail must be between 1 and 5000"))
				return
			}
		}
		logs, err := usecase.GetAppDeployLogs(ns, name, rest[2], tail)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"logs": logs})
		return
	}
	if len(rest) == 5 && rest[1] == "deploy" && rest[3] == "logs" && rest[4] == "stream" && r.Method == http.MethodGet {
		streamContainerLogs(w, r, func(tail string) (io.ReadCloser, error) { return usecase.OpenAppRuntimeLogs(ns, name, rest[2], tail) })
		return
	}
	if len(rest) == 2 && rest[1] == "deploy" && r.Method == http.MethodPost {
		var in struct {
			Branch        string `json:"branch"`
			Commit        string `json:"commit"`
			Tag           string `json:"tag"`
			RestartPolicy string `json:"restart_policy"`
			LogDriver     string `json:"log_driver"`
			LogMaxSize    string `json:"log_max_size"`
			LogMaxFile    int    `json:"log_max_file"`
			SLSProject    string `json:"sls_project"`
			SLSLogstore   string `json:"sls_logstore"`
			SLSEndpoint   string `json:"sls_endpoint"`
			SLSConfigName string `json:"sls_config_name"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
		job, err := usecase.StartDeployApp(usecase.DeployAppOptions{Namespace: ns, Name: name, Branch: in.Branch, Commit: in.Commit, Tag: in.Tag, ContainerEditOptions: usecase.ContainerEditOptions{RestartPolicy: in.RestartPolicy, LogDriver: in.LogDriver, LogMaxSize: in.LogMaxSize, LogMaxFile: in.LogMaxFile, SLSProject: in.SLSProject, SLSLogstore: in.SLSLogstore, SLSEndpoint: in.SLSEndpoint, SLSConfigName: in.SLSConfigName}})
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
		return
	}
	if len(rest) == 1 && r.Method == http.MethodDelete {
		if err := usecase.RemoveApp(ns, name); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	if len(rest) == 1 && r.Method == http.MethodPut {
		var in domain.AppSpec
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
		if in.CPU == 0 {
			in.CPU = 1
		}
		if in.Memory == 0 {
			in.Memory = 1
		}
		if err := usecase.UpdateApp(ns, name, in); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, in)
		return
	}
	methodNotAllowed(w, "DELETE, PUT, POST")
}

func handleDatabases(w http.ResponseWriter, r *http.Request, ns string, rest []string) {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			v, err := usecase.Listdatabase(ns)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 200, v)
		case http.MethodPost:
			var in domain.DatabaseSpec
			if err := decodeJSON(w, r, &in); err != nil {
				return
			}
			in.Namespace = ns
			if in.CPU == 0 {
				in.CPU = 1
			}
			if in.Memory == 0 {
				in.Memory = 2
			}
			if in.DbType == "" {
				in.DbType = "mysql:5.7"
			}
			if in.RestartPolicy == "" {
				in.RestartPolicy = "unless-stopped"
			}
			if in.Name == "" || in.Username == "" || in.Password == "" || in.DbName == "" {
				writeBadRequest(w, "name, username, password and dbname are required")
				return
			}
			if err := usecase.Createdatabase(in); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 201, in)
		default:
			methodNotAllowed(w, "GET, POST")
		}
		return
	}
	name := rest[0]
	if len(rest) == 2 && rest[1] == "logs" && r.Method == http.MethodGet {
		streamContainerLogs(w, r, func(tail string) (io.ReadCloser, error) { return usecase.OpenDatabaseLogs(ns, name, tail) })
		return
	}
	if len(rest) == 2 && (rest[1] == "start" || rest[1] == "stop") && r.Method == http.MethodPost {
		if err := usecase.SetDatabaseRunning(ns, name, rest[1] == "start"); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": rest[1] + "ed"})
		return
	}
	if len(rest) == 1 && r.Method == http.MethodPut {
		var in struct {
			RestartPolicy string `json:"restart_policy"`
			LogDriver     string `json:"log_driver"`
			LogMaxSize    string `json:"log_max_size"`
			LogMaxFile    int    `json:"log_max_file"`
			ApplyNow      bool   `json:"apply_now"`
			SLSProject    string `json:"sls_project"`
			SLSLogstore   string `json:"sls_logstore"`
			SLSEndpoint   string `json:"sls_endpoint"`
			SLSConfigName string `json:"sls_config_name"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
		if err := usecase.UpdateDatabaseConfig(ns, name, usecase.ContainerEditOptions{RestartPolicy: in.RestartPolicy, LogDriver: in.LogDriver, LogMaxSize: in.LogMaxSize, LogMaxFile: in.LogMaxFile, ApplyNow: in.ApplyNow, SLSProject: in.SLSProject, SLSLogstore: in.SLSLogstore, SLSEndpoint: in.SLSEndpoint, SLSConfigName: in.SLSConfigName}); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"restart_policy": in.RestartPolicy})
		return
	}
	if len(rest) == 1 && r.Method == http.MethodDelete {
		if err := usecase.Removedatabase(ns, name); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	if len(rest) == 2 && rest[1] == "export" && r.Method == http.MethodGet {
		data, err := usecase.ExportSQLData(ns, name)
		if err != nil {
			writeError(w, err)
			return
		}
		w.Header().Set("Content-Type", "application/sql; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.sql"`, name))
		w.Write(data)
		return
	}
	if len(rest) == 2 && rest[1] == "import" && r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
		if err := usecase.StartDatabaseImport(ns, name, r.Body); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "importing"})
		return
	}
	methodNotAllowed(w, "DELETE, GET, POST")
}

func handleRedis(w http.ResponseWriter, r *http.Request, ns string, rest []string) {
	if len(rest) == 0 {
		switch r.Method {
		case http.MethodGet:
			v, err := usecase.ListRedis(ns)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 200, v)
		case http.MethodPost:
			var body struct {
				Name          string  `json:"name"`
				Version       string  `json:"version"`
				CPU           float64 `json:"cpu"`
				Memory        float64 `json:"memory"`
				Password      string  `json:"password"`
				AOF           *bool   `json:"appendonly"`
				Eviction      string  `json:"maxmemory_policy"`
				RestartPolicy string  `json:"restart_policy"`
			}
			if err := decodeJSON(w, r, &body); err != nil {
				return
			}
			in := domain.RedisSpec{Name: body.Name, Namespace: ns, Version: body.Version, CPU: body.CPU, Memory: body.Memory, Password: body.Password, Eviction: body.Eviction, RestartPolicy: body.RestartPolicy, AOF: true}
			if body.AOF != nil {
				in.AOF = *body.AOF
			}
			if in.Name == "" {
				writeBadRequest(w, "name is required")
				return
			}
			if in.CPU == 0 {
				in.CPU = .5
			}
			if in.Memory == 0 {
				in.Memory = .5
			}
			if in.Version == "" {
				in.Version = "7"
			}
			if in.Eviction == "" {
				in.Eviction = "allkeys-lru"
			}
			if in.RestartPolicy == "" {
				in.RestartPolicy = "unless-stopped"
			}
			if err := usecase.CreateRedis(in); err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, 201, in)
		default:
			methodNotAllowed(w, "GET, POST")
		}
		return
	}
	if len(rest) == 2 && rest[1] == "logs" && r.Method == http.MethodGet {
		streamContainerLogs(w, r, func(tail string) (io.ReadCloser, error) { return usecase.OpenRedisLogs(ns, rest[0], tail) })
		return
	}
	if len(rest) == 2 && (rest[1] == "start" || rest[1] == "stop") && r.Method == http.MethodPost {
		if err := usecase.SetRedisRunning(ns, rest[0], rest[1] == "start"); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": rest[1] + "ed"})
		return
	}
	if len(rest) == 1 && r.Method == http.MethodPut {
		var in struct {
			RestartPolicy string `json:"restart_policy"`
			LogDriver     string `json:"log_driver"`
			LogMaxSize    string `json:"log_max_size"`
			LogMaxFile    int    `json:"log_max_file"`
			ApplyNow      bool   `json:"apply_now"`
			SLSProject    string `json:"sls_project"`
			SLSLogstore   string `json:"sls_logstore"`
			SLSEndpoint   string `json:"sls_endpoint"`
			SLSConfigName string `json:"sls_config_name"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			return
		}
		if err := usecase.UpdateRedisConfig(ns, rest[0], usecase.ContainerEditOptions{RestartPolicy: in.RestartPolicy, LogDriver: in.LogDriver, LogMaxSize: in.LogMaxSize, LogMaxFile: in.LogMaxFile, ApplyNow: in.ApplyNow, SLSProject: in.SLSProject, SLSLogstore: in.SLSLogstore, SLSEndpoint: in.SLSEndpoint, SLSConfigName: in.SLSConfigName}); err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"restart_policy": in.RestartPolicy})
		return
	}
	if len(rest) == 1 && r.Method == http.MethodDelete {
		if err := usecase.RemoveRedis(ns, rest[0]); err != nil {
			writeError(w, err)
			return
		}
		w.WriteHeader(204)
		return
	}
	methodNotAllowed(w, "DELETE")
}

type repoInput struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	URL   string `json:"url"`
}

func repositories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	v, err := usecase.RepoList()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func repositoryResource(w http.ResponseWriter, r *http.Request) {
	provider := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/repositories/"), "/")
	if provider != "github" && provider != "gitee" && provider != "gitlab" {
		writeBadRequest(w, "provider must be github, gitee or gitlab")
		return
	}
	var in repoInput
	if r.Method == http.MethodDelete {
		in.Name = r.URL.Query().Get("name")
		in.URL = r.URL.Query().Get("url")
	} else if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	if in.Name == "" {
		writeBadRequest(w, "name is required")
		return
	}
	if provider == "gitlab" && in.URL == "" {
		writeBadRequest(w, "url is required for gitlab")
		return
	}
	values := map[string]string{"repo": provider, "name": in.Name, "token": in.Token, "url": in.URL}
	var err error
	switch r.Method {
	case http.MethodPost:
		if in.Token == "" {
			writeBadRequest(w, "token is required")
			return
		}
		err = usecase.RepoAdd(values)
	case http.MethodPut:
		if in.Token == "" {
			writeBadRequest(w, "token is required")
			return
		}
		err = usecase.RepoUpdate(values)
	case http.MethodDelete:
		err = usecase.RepoRemove(values)
	default:
		methodNotAllowed(w, "POST, PUT, DELETE")
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	if r.Method == http.MethodPost {
		writeJSON(w, 201, in)
	} else {
		w.WriteHeader(204)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeBadRequest(w, "invalid JSON: "+err.Error())
		return err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeBadRequest(w, "request body must contain one JSON object")
		return errors.New("multiple JSON values")
	}
	return nil
}

func splitPath(p string) []string {
	raw := strings.Split(strings.Trim(p, "/"), "/")
	out := raw[:0]
	for _, v := range raw {
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeBadRequest(w http.ResponseWriter, message string) {
	writeJSON(w, 400, map[string]string{"error": message})
}
func writeError(w http.ResponseWriter, err error) {
	status := 500
	s := strings.ToLower(err.Error())
	if strings.Contains(s, "not found") || strings.Contains(s, "not exist") {
		status = 404
	} else if strings.Contains(s, "exist") || strings.Contains(s, "already running") || strings.Contains(s, "is running") {
		status = 409
	} else if strings.Contains(s, "required") || strings.Contains(s, "invalid") || strings.Contains(s, "support") || strings.Contains(s, "empty") {
		status = 400
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeJSON(w, 405, map[string]string{"error": "method not allowed"})
}
func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeJSON(w, 500, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
