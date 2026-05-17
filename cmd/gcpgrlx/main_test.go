package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/djburkhart/gcpgrlx"
)

func TestRunValidate(t *testing.T) {
	raw, err := gcpgrlx.SampleConfigYAML()
	if err != nil {
		t.Fatalf("SampleConfigYAML returned error: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "microservices.yaml")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"validate", "-f", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "config is valid:") {
		t.Fatalf("unexpected validate output: %s", stdout.String())
	}
}

func TestRunPlan(t *testing.T) {
	raw, err := gcpgrlx.SampleConfigYAML()
	if err != nil {
		t.Fatalf("SampleConfigYAML returned error: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "microservices.yaml")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"plan", "-f", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Plan for project") {
		t.Fatalf("unexpected plan output: %s", output)
	}
	if !strings.Contains(output, "gcloud run services update-traffic 'identity-grpc'") {
		t.Fatalf("plan output missing traffic command: %s", output)
	}
}

func TestRunRenderStdout(t *testing.T) {
	raw, err := gcpgrlx.SampleConfigYAML()
	if err != nil {
		t.Fatalf("SampleConfigYAML returned error: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "microservices.yaml")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"render-stdout", "-f", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "name: gcpgrlx-cloud-run-deploy") {
		t.Fatalf("unexpected render-stdout output: %s", output)
	}
	if !strings.Contains(output, "Update traffic for identity-grpc") {
		t.Fatalf("render-stdout output missing traffic state: %s", output)
	}
}

func TestRunInitWritesSampleConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "microservices.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"init", "-out", configPath}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	cfg, err := gcpgrlx.ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if cfg.ProjectID != "sample-platform-prod" {
		t.Fatalf("unexpected sample config project: %s", cfg.ProjectID)
	}
	if len(cfg.Services) == 0 {
		t.Fatal("expected sample config services")
	}
}

func TestRunRenderCaddyStdout(t *testing.T) {
	raw, err := gcpgrlx.SampleConfigYAML()
	if err != nil {
		t.Fatalf("SampleConfigYAML returned error: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "microservices.yaml")
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"render-caddy", "-f", configPath, "--stdout"}, &stdout, &stderr); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "api.sample-platform.dev {") {
		t.Fatalf("unexpected render-caddy output: %s", output)
	}
	if !strings.Contains(output, "handle_path /identity/* {") {
		t.Fatalf("render-caddy output missing route block: %s", output)
	}
}
