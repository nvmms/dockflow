package traefik

import (
	"bufio"
	"dockflow/internal/config"
	"dockflow/internal/service/docker"
	"dockflow/internal/service/system"
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
	print(email)

	if err := system.CheckPorts(80, 443); err != nil {
		return err
	}

	// if err := EnsureImage(TraefikImage); err != nil {
	// 	return err
	// }

	if err := ensureContainer(cfg, email); err != nil {
		return err
	}

	return nil
}

// var (
// 	ErrAcmeEmailMissing = errors.New("traefik acme email is missing")
// 	ErrInvalidEmail     = errors.New("invalid email")
// )

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

func ensureContainer(cfg *config.Config, acmeEmail string) (err error) {
	containerId := strings.TrimSpace(cfg.Platform.Traefik.ContainerId)
	if containerId == "" {
		containerId, err = createTraefikContainer(acmeEmail)
		if err != nil {
			return err
		}
	}

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

	docker.RunContainer(opt)
	return "", nil
}
