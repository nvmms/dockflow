package domain

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

/* ---------- Root ---------- */

type TraefikConfig struct {
	Path string     `yaml:"-"`
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Routers     map[string]Router     `yaml:"routers"`
	Services    map[string]Service    `yaml:"services"`
	Middlewares map[string]Middleware `yaml:"middlewares,omitempty"`
}

/* ---------- Router ---------- */

type Router struct {
	Rule        string     `yaml:"rule"`
	Service     string     `yaml:"service"`
	EntryPoints []string   `yaml:"entryPoints,omitempty"`
	Middlewares []string   `yaml:"middlewares,omitempty"`
	TLS         *TLSConfig `yaml:"tls,omitempty"`
}

type TLSConfig struct {
	CertResolver string `yaml:"certResolver,omitempty"`
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

/* ---------- Middleware ---------- */

type Middleware struct {
	StripPrefix      *StripPrefix      `yaml:"stripPrefix,omitempty"`
	StripPrefixRegex *StripPrefixRegex `yaml:"stripPrefixRegex,omitempty"`
}

type StripPrefix struct {
	Prefixes []string `yaml:"prefixes"`
}

type StripPrefixRegex struct {
	Regex []string `yaml:"regex"`
}

/* ---------- Options ---------- */

type TraefikServiceOpt struct {
	Name         string
	Rule         string
	Url          string
	EnableTLS    bool
	CertResolver string
}

/* ---------- Constructor ---------- */

func NewTraefikConfig(path string) (*TraefikConfig, error) {
	c := &TraefikConfig{
		Path: path,
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			c.HTTP = HTTPConfig{
				Routers:     make(map[string]Router),
				Services:    make(map[string]Service),
				Middlewares: make(map[string]Middleware),
			}
			return c, nil
		}
		return nil, err
	}

	if err := c.Load(); err != nil {
		return nil, err
	}

	return c, nil
}

/* ---------- Load / Save ---------- */

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
	if c.HTTP.Middlewares == nil {
		c.HTTP.Middlewares = make(map[string]Middleware)
	}

	return nil
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

/* ---------- Core Logic ---------- */

func (c *TraefikConfig) AddService(opt TraefikServiceOpt) {
	if c.HTTP.Routers == nil {
		c.HTTP.Routers = make(map[string]Router)
	}
	if c.HTTP.Services == nil {
		c.HTTP.Services = make(map[string]Service)
	}
	if c.HTTP.Middlewares == nil {
		c.HTTP.Middlewares = make(map[string]Middleware)
	}

	routerName := "app-" + opt.Name
	serviceName := "svc-" + opt.Name
	middlewareName := "mw-strip-" + opt.Name

	rule := normalizeRule(opt.Rule)

	router := Router{
		Rule:    rule,
		Service: serviceName,
	}

	// EntryPoints / TLS
	if opt.EnableTLS {
		router.EntryPoints = []string{"websecure"}
		if opt.CertResolver == "" {
			router.TLS = &TLSConfig{CertResolver: "letsencrypt"}
		} else {
			router.TLS = &TLSConfig{CertResolver: opt.CertResolver}
		}
	} else {
		router.EntryPoints = []string{"web"}
	}

	// === 自动 strip path ===
	if path := extractPathFromRule(opt.Rule); path != "" {
		c.HTTP.Middlewares[middlewareName] = Middleware{
			StripPrefix: &StripPrefix{
				Prefixes: []string{path},
			},
		}

		router.Middlewares = []string{middlewareName}
	}

	c.HTTP.Routers[routerName] = router

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

func (c *TraefikConfig) RemoveService(name string) {
	if c.HTTP.Routers != nil {
		delete(c.HTTP.Routers, "app-"+name)
	}
	if c.HTTP.Services != nil {
		delete(c.HTTP.Services, "svc-"+name)
	}
	if c.HTTP.Middlewares != nil {
		delete(c.HTTP.Middlewares, "mw-strip-"+name)
	}
}

/* ---------- Helpers ---------- */

func normalizeRule(rule string) string {
	rule = strings.TrimSpace(rule)

	// 已经是 Traefik Rule，直接放行
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

func extractPathFromRule(rule string) string {
	rule = strings.TrimSpace(rule)

	// 已经是完整 Traefik rule，不自动 strip
	if strings.Contains(rule, "Host(") {
		return ""
	}

	if idx := strings.Index(rule, "/"); idx != -1 {
		return rule[idx:]
	}

	return ""
}

func normalizeURL(raw string) string {
	if raw == "" {
		return raw
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}

	return "http://" + raw
}
