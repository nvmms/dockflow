package domain

type DatabaseSpec struct {
	Namespace     string   `json:"namespace"`
	Name          string   `json:"name"`
	CPU           float64  `json:"cpu"`
	Memory        float64  `json:"memory"`
	Username      string   `json:"username"`
	Password      string   `json:"password"`
	DbName        string   `json:"dbname"`
	DbType        string   `json:"dbtype"`
	ContainerId   string   `json:"container_id"`
	Ip            []string `json:"ip"`
	Remote        bool     `json:"remote"`
	RestartPolicy string   `json:"restart_policy,omitempty"`
	Status        string   `json:"status,omitempty"`
	ImportError   string   `json:"import_error,omitempty"`
}

// func NewDatabaseSpec(
// 	namespace string,
// 	name string,
// 	cpu float64,
// 	memory float64,
// 	username string,
// 	password string,
// 	dbname string,
// 	// version string,
// 	// aof bool,
// 	// eviction string,
// ) DatabaseSpec {

// 	return DatabaseSpec{
// 		Name:      name,
// 		Namespace: namespace,
// 		Password:  password,
// 		CPU:       cpu,
// 		Memory:    memory,
// 		Version:   version,
// 		AOF:       aof,
// 		Eviction:  eviction,
// 	}
// }
