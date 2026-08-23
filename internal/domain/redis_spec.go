package domain

type RedisSpec struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Version     string   `json:"version"`
	CPU         float64  `json:"cpu"`
	Memory      float64  `json:"memory"`
	Password    string   `json:"password"`
	AOF         bool     `json:"appendonly"`
	Eviction    string   `json:"maxmemory_policy"`
	ContainerId string   `json:"container_id"`
	Ip          []string `json:"ip"`
	Status      string   `json:"status,omitempty"`
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
