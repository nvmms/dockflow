package main

import (
	"context"
	"dockflow/internal/cli"
	"dockflow/internal/service/docker"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "daemon" {
		runDaemon()
		return
	}

	runCLI()
}

func runCLI() {
	cli.Execute()
}

func runDaemon() {
	ctx, cancel := context.WithCancel(context.Background())

	go docker.ListenDockerEvents(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	<-sig
	cancel()
	log.Println("dockflow daemon stopped")
}
