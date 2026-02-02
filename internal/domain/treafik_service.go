package domain

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type TraefikConfig struct {
	Path string     `yaml:"-"`
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Routers  map[string]Router  `yaml:"routers"`
	Services map[string]Service `yaml:"services"`
}

/* ---------- Router ---------- */

type Router struct {
	Rule        string   `yaml:"rule"`
	Service     string   `yaml:"service"`
	Middlewares []string `yaml:"middlewares,omitempty"`
}

/* ---------- Service ---------- */

type Service struct {
	LoadBalancer *LoadBalancer `yaml:"loadBalancer,omitempty"`
}

type LoadBalancer struct {
	Servers []Servers `yaml:"servers"`
}

type Servers struct {
	URL string `yaml:"url"`
}

type TraefikServiceOpt struct {
	Name string
	Rule string
	Url  string
}

/* ---------- func ---------- */

func NewTraefikConfig(path string) (*TraefikConfig, error) {
	c := &TraefikConfig{
		Path: path,
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			c.HTTP = HTTPConfig{
				Routers:  make(map[string]Router),
				Services: make(map[string]Service),
			}
			return c, nil
		}
		return nil, err
	}
	err := c.Load()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *TraefikConfig) Load() error {
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}

	// 防御性初始化
	if c.HTTP.Routers == nil {
		c.HTTP.Routers = make(map[string]Router)
	}
	if c.HTTP.Services == nil {
		c.HTTP.Services = make(map[string]Service)
	}

	return nil
}

func (c *TraefikConfig) AddService(opt TraefikServiceOpt) {
	// 初始化 map（非常重要）
	if c.HTTP.Routers == nil {
		c.HTTP.Routers = make(map[string]Router)
	}
	if c.HTTP.Services == nil {
		c.HTTP.Services = make(map[string]Service)
	}

	routerName := "app-" + opt.Name
	serviceName := "svc-" + opt.Name

	c.HTTP.Routers[routerName] = Router{
		Rule:    opt.Rule,
		Service: serviceName,
	}

	c.HTTP.Services[serviceName] = Service{
		LoadBalancer: &LoadBalancer{
			Servers: []Servers{
				{URL: opt.Url},
			},
		},
	}
}

func (c *TraefikConfig) Save() error {
	if c.Path == "" {
		return fmt.Errorf("traefik config path is empty")
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	tmp := c.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmp, c.Path)
}

func (c *TraefikConfig) RemoveService(name string) {
	if c.HTTP.Routers != nil {
		delete(c.HTTP.Routers, "app-"+name)
	}

	if c.HTTP.Services != nil {
		delete(c.HTTP.Services, "svc-"+name)
	}
}

func (c *TraefikConfig) DebugPrint()
