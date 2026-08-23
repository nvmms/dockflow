package domain

type RedisSpec struct {
	Name          string   `json:"name"`
	Namespace     string   `json:"namespace"`
	Version       string   `json:"version"`
	CPU           float64  `json:"cpu"`
	Memory        float64  `json:"memory"`
	Password      string   `json:"password"`
	AOF           bool     `json:"appendonly"`
	Eviction      string   `json:"maxmemory_policy"`
	ContainerId   string   `json:"container_id"`
	Ip            []string `json:"ip"`
	RestartPolicy string   `json:"restart_policy,omitempty"`
	LogDriver     string   `json:"log_driver,omitempty"`
	LogMaxSize    string   `json:"log_max_size,omitempty"`
	LogMaxFile    int      `json:"log_max_file,omitempty"`
	SLSProject    string   `json:"sls_project,omitempty"`
	SLSLogstore   string   `json:"sls_logstore,omitempty"`
	SLSEndpoint   string   `json:"sls_endpoint,omitempty"`
	SLSConfigName string   `json:"sls_config_name,omitempty"`
	NeedsRecreate bool     `json:"needs_recreate,omitempty"`
	Status        string   `json:"status,omitempty"`
}

func NewRedisSpace(
	name string,
	namespace string,
	password string,
	cpu float64,
	memory float64,
	version string,
	aof bool,
	eviction string,
) RedisSpec {

	return RedisSpec{
		Name:      name,
		Namespace: namespace,
		Password:  password,
		CPU:       cpu,
		Memory:    memory,
		Version:   version,
		AOF:       aof,
		Eviction:  eviction,
	}
}
