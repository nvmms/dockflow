package domain

type RedisSpec struct {
	Name        string
	Namespace   string
	Version     string
	CPU         float64
	Memory      float64
	Password    string
	AOF         bool
	Eviction    string
	ContainerId string
	Ip          []string
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
