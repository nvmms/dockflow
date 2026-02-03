package traefik

import (
	"bufio"
	"dockflow/internal/config"
	"dockflow/internal/service/docker"
	"fmt"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
)

const (
	TraefikImage         = "traefik:v3.6"
	TraefikContainerName = "dockflow-traefik"
	TraefikVolume        = "dockflow-traefik-acme"
	TraefikNetwork       = "dockflow-traefik"
)

func Init() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	email, err := ensureAcmeEmail(cfg)
	if err != nil {
		return err
	}
	config.Save(cfg)

	// if err := system.CheckPorts(80, 443); err != nil {
	// 	return err
	// }

	if err := ensureNetwork(cfg); err != nil {
		return err
	}

	if err := docker.PullImage(TraefikImage); err != nil {
		return err
	}

	if err := ensureContainer(cfg, email); err != nil {
		return err
	}

	return nil
}

func ensureAcmeEmail(cfg *config.Config) (string, error) {
	email := strings.TrimSpace(cfg.Platform.Traefik.AcmeEmail)
	if email != "" {
		return email, nil
	}

	in := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter email for Let's Encrypt (ACME): ")
		line, err := in.ReadString('\n')
		if err != nil {
			return "", err
		}

		email = strings.TrimSpace(line)
		if email == "" || !strings.Contains(email, "@") {
			fmt.Println("Invalid email, please try again.")
			continue
		}

		cfg.Platform.Traefik.AcmeEmail = email
		if err := config.Save(cfg); err != nil {
			return "", err
		}
		return email, nil
	}
}

func ensureNetwork(cfg *config.Config) (err error) {
	networkId := strings.TrimSpace(cfg.Platform.Traefik.ContainerId)
	if networkId == "" {
		networkId, err = createTraefikNetwork(cfg)
		if err != nil {
			return err
		}
	} else {
		networkId, err := docker.HasNetwork(networkId)
		if err != nil {
			return err
		}
		if networkId == "" {
			networkId, err = createTraefikNetwork(cfg)
			if err != nil {
				return err
			}
		}
	}
	config.Save(cfg)
	return nil
}

func createTraefikNetwork(cfg *config.Config) (string, error) {
	networkId, err := docker.HasNetwork(TraefikNetwork)
	if err != nil {
		return "", err
	}
	if networkId != "" {
		return networkId, nil
	}
	opts := docker.NetworkCreateOptions{
		Name:       TraefikNetwork,
		Driver:     "bridge",
		Subnet:     "10.0.0.0/8",
		Gateway:    "10.0.0.1",
		Attachable: true,
	}
	networkId, err = docker.CreateNetwork(opts)
	if err != nil {
		return "", err
	}
	cfg.Platform.Traefik.NetworkId = networkId
	config.Save(cfg)

	return networkId, nil
}

func ensureContainer(cfg *config.Config, acmeEmail string) (err error) {
	containerId := strings.TrimSpace(cfg.Platform.Traefik.ContainerId)
	if containerId == "" {
		containerId, err = createTraefikContainer(acmeEmail)
		if err != nil {
			return err
		}

		cfg.Platform.Traefik.ContainerId = containerId
		config.Save(cfg)
	} else {
		containerId, err := docker.HasContainer(containerId)
		if err != nil {
			return err
		}
		if containerId != "" {
			containerId, err = createTraefikContainer(acmeEmail)
			cfg.Platform.Traefik.ContainerId = containerId
			config.Save(cfg)
		} else {
			isRun, err := docker.ContainerRunning(containerId)
			if err != nil {
				return err
			}
			if !isRun {
				err = docker.StartContainer(containerId)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

func createTraefikContainer(acmeEmail string) (containerId string, err error) {

	opt := docker.NewRunOptions(TraefikContainerName, TraefikImage)

	opt.WithRestart(container.RestartPolicyAlways)
	opt.WithNetwork(TraefikNetwork)

	opt.WithPort(80, 80)
	opt.WithPort(443, 443)

	opt.WithVolume("/var/run/docker.sock", "/var/run/docker.sock", "ro")
	opt.WithVolume(TraefikVolume, "/letsencrypt")

	opt.WithCommand(
		"--api=true",
		"--api.dashboard=false",
		"--api.insecure=false",
		"--ping=true",

		"--entrypoints.http.address=:80",
		"--entrypoints.https.address=:443",
		"--providers.docker=true",
		"--providers.docker.endpoint=unix:///var/run/docker.sock",
		"--providers.docker.exposedbydefault=false",

		"--certificatesresolvers.le.acme.email="+acmeEmail,
		"--certificatesresolvers.le.acme.storage=/letsencrypt/acme.json",
		"--certificatesresolvers.le.acme.httpchallenge=true",
		"--certificatesresolvers.le.acme.httpchallenge.entrypoint=http",
	)

	containerId, err = docker.RunContainer(opt)

	if err != nil {
		return containerId, err
	}
	return "", err
}
