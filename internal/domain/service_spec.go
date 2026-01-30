package domain

type ServiceURL struct {
	Host string `json:"host"` // external access domain
	Port string `json:"port"` // container port
}

type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Trigger struct {
	Type string `json:"type"` // branch | tag
	Rule string `json:"rule"` // main | v* | v1.*
}

type ServiceSpec struct {
	Namespace string       `json:"namespace"`
	Name      string       `json:"name"`
	CPU       float64      `json:"cpu"`     // CPU cores
	Memory    int          `json:"memory"`  // Memory in GB
	Repo      string       `json:"repo"`    // Git repository
	Token     string       `json:"token"`   // Git access token (optional)
	Trigger   Trigger      `json:"trigger"` // Deploy trigger
	Envs      []Env        `json:"env"`     // Environment variables
	URLs      []ServiceURL `json:"url"`     // Access rules
}
