package domain

import (
	"fmt"
	"os"
	"strings"

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
	if c.HTTP.Routers == nil {
		c.HTTP.Routers = make(map[string]Router)
	}
	if c.HTTP.Services == nil {
		c.HTTP.Services = make(map[string]Service)
	}

	routerName := "app-" + opt.Name
	serviceName := "svc-" + opt.Name

	c.HTTP.Routers[routerName] = Router{
		Rule:    normalizeRule(opt.Rule),
		Service: serviceName,
	}

	c.HTTP.Services[serviceName] = Service{
		LoadBalancer: &LoadBalancer{
			Servers: []Servers{
				{
					URL: normalizeURL(opt.Url),
				},
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

func (c *TraefikConfig) DebugPrint() {}

func normalizeRule(rule string) string {
	rule = strings.TrimSpace(rule)

	// 已经是 Traefik Rule，直接放行（兜底）
	if strings.Contains(rule, "Host(") {
		return rule
	}

	host := rule
	path := ""

	if idx := strings.Index(rule, "/"); idx != -1 {
		host = rule[:idx]
		path = rule[idx:]
	}

	if path == "" {
		return fmt.Sprintf(`Host("%s")`, host)
	}

	return fmt.Sprintf(
		`Host("%s") && PathPrefix("%s")`,
		host,
		path,
	)
}

func normalizeURL(raw string) string {
	if raw == "" {
		return raw
	}

	// 已经有协议，直接返回
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}

	// 否则默认补 http://
	return "http://" + raw
}
