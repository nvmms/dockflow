package monitor

import (
	"context"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
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

	}
}

func handleContainerEvent(msg events.Message) {
	containerId := msg.Actor.ID
	action := msg.Action

	containerMonitor := NewMonitorContainer(containerId)
	if containerMonitor == nil {
		return
	}

	switch action {
	case "start":
		containerMonitor.onStart()
	case "die":
		containerMonitor.onDie()
	}
}
