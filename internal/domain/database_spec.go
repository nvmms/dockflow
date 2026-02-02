package domain

type DatabaseSpec struct {
	Namespace   string
	Name        string
	CPU         float64
	Memory      float64
	Username    string
	Password    string
	DbName      string
	DbType      string
	ContainerId string
	Ip          []string
	Remote      bool
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
