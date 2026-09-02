package main

import (
	"os"
	"strings"
	"testing"
)

func TestDockerHealthCheckUsesConfiguredPort(t *testing.T) {
	t.Parallel()

	dockerfile, err := os.ReadFile("Dockerfile")
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}

	contents := string(dockerfile)
	if !strings.Contains(contents, "${PORT:-5000}") {
		t.Fatal("Docker health check must use PORT with a 5000 default")
	}
	if strings.Contains(contents, "http://127.0.0.1:5000/healthz") {
		t.Fatal("Docker health check must not hardcode port 5000")
	}
}
