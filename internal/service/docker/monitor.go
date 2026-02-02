package docker

import (
	"context"
	"dockflow/internal/domain"
	"dockflow/internal/service/filesystem"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/samber/lo"
)

func ListenDockerEvents(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return err
	}

	// 过滤条件（非常重要，别全量监听）
	filter := filters.NewArgs()
	filter.Add("type", string(events.ContainerEventType))

	msgCh, errCh := cli.Events(ctx, types.EventsOptions{
		Filters: filter,
	})

	log.Println("[dockflow] docker event listener started")

	for {
		select {
		case msg := <-msgCh:
			handleDockerEvent(msg)

		case err := <-errCh:
			if err != nil {
				log.Println("[dockflow] docker event error:", err)
				return err
			}

		case <-ctx.Done():
			log.Println("[dockflow] docker event listener stopped")
			return nil
		}
	}
}

func handleDockerEvent(msg events.Message) {
	switch msg.Type {

	case events.ContainerEventType:
		handleContainerEvent(msg)

	case events.VolumeEventType:
		handleVolumeEvent(msg)
	}
}

func handleContainerEvent(msg events.Message) {
	containerID := msg.Actor.ID
	action := msg.Action

	switch action {
	case "start":
		containerInfo, err := InspectContainer(containerID)
		if err != nil {
			print(err)
		}
		labels := containerInfo.Config.Labels

		namespace, exists := labels["dockflow.namespace"]
		if !exists || namespace == "" {
			return
		}

		name, exists := labels["dockflow.name"]
		if !exists || name == "" {
			return
		}

		version, exists := labels["dockflow.version"]
		if !exists || version == "" {
			return
		}

		ns, err := filesystem.LoadNamespace(namespace)
		if err != nil || ns == nil {
			return
		}

		app, found := lo.Find(ns.App, func(app domain.AppSpec) bool {
			return app.Name == name
		})
		if !found || app.Name == "" {
			return
		}

	case "die":
		log.Println("[container die]", containerID)

	case "destroy":
		log.Println("[container destroy]", containerID)
	}
}

func handleVolumeEvent(msg events.Message) {
	volumeName := msg.Actor.ID
	action := msg.Action

	switch action {
	case "create":
		log.Println("[volume create]", volumeName)

	case "remove":
		log.Println("[volume remove]", volumeName)
	}
}
