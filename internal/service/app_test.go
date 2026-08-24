package service

import (
	"testing"

	"dockflow/internal/domain"
)

func TestAppEnvBuildArgs(t *testing.T) {
	args := appEnvBuildArgs([]domain.Env{
		{Key: "VITE_API_URL", Value: "https://api.example.com"},
		{Key: "NODE_ENV", Value: "production"},
	})

	if got := args["VITE_API_URL"]; got == nil || *got != "https://api.example.com" {
		t.Fatalf("VITE_API_URL = %v, want https://api.example.com", got)
	}
	if got := args["NODE_ENV"]; got == nil || *got != "production" {
		t.Fatalf("NODE_ENV = %v, want production", got)
	}
}
