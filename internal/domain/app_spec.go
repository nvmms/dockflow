package domain

type AppURL struct {
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

type AppDeploy struct {
	ContainerId string `json:"containerId"`
	Version     string `json:"version"`
	Url         string `json:"url"`
}

type AppSpec struct {
	Namespace string             `json:"namespace"`
	Name      string             `json:"name"`
	CPU       float64            `json:"cpu"`     // CPU cores
	Memory    int                `json:"memory"`  // Memory in GB
	Repo      string             `json:"repo"`    // Git repository
	Token     string             `json:"token"`   // Git access token (optional)
	Trigger   Trigger            `json:"trigger"` // Deploy trigger
	Envs      []Env              `json:"env"`     // Environment variables
	URLs      []AppURL           `json:"url"`     // Access rules
	Deploy    []AppDeploy        `json:"deploy"`
	BuildArg  map[string]*string `json:"buildArg"`
	Secret    string             `json:"secret"`
}
