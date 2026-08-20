package main

import (
	"context"
	"dockflow/internal/cli"
	"dockflow/internal/service/monitor"
	"dockflow/internal/service/webhook"
	"dockflow/internal/usecase"
	"dockflow/internal/webapi"
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
	log.Println("dockflow daemon initializing environment")
	if err := usecase.Init(); err != nil {
		log.Fatalf("dockflow daemon initialization failed: %v", err)
	}
	log.Println("dockflow environment ready")

	ctx, cancel := context.WithCancel(context.Background())

	gitService := webhook.NewGitService()
	webhookServer := webhook.NewServer(":8090", gitService, webapi.NewHandler())
	webhookServer.Start(ctx)

	go monitor.ListenDockerEvents(ctx)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)

	<-sig
	cancel()
	log.Println("dockflow daemon stopped")
}
