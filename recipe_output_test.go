package gcpgrlx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderRecipeHTTPServiceOmitsGRPCFlags(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Services: []Service{
			{
				Name:         "public-api",
				Image:        "us-central1-docker.pkg.dev/sample-project/platform-services/public-api:latest",
				Protocol:     "http",
				Port:         8080,
				CPU:          "1",
				Memory:       "512Mi",
				Concurrency:  80,
				MinInstances: 0,
				MaxInstances: 3,
				Timeout:      "300s",
				Ingress:      "all",
				Env: map[string]string{
					"APP_ENV": "production",
				},
			},
		},
	}

	recipe, err := RenderRecipe(cfg)
	if err != nil {
		t.Fatalf("RenderRecipe returned error: %v", err)
	}

	assertContains(t, recipe, "--concurrency 80")
	assertNotContains(t, recipe, "--use-http2")
	assertNotContains(t, recipe, "--vpc-connector")
	assertNotContains(t, recipe, "gcloud beta run deploy")
}

func TestWriteRecipeFileCreatesDeployRecipe(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Services: []Service{
			{
				Name:         "public-api",
				Image:        "us-central1-docker.pkg.dev/sample-project/platform-services/public-api:latest",
				Protocol:     "http",
				Port:         8080,
				CPU:          "1",
				Memory:       "512Mi",
				Concurrency:  80,
				MaxInstances: 3,
				Timeout:      "300s",
				Ingress:      "all",
			},
		},
	}

	outDir := t.TempDir()
	path, err := WriteRecipeFile(cfg, outDir)
	if err != nil {
		t.Fatalf("WriteRecipeFile returned error: %v", err)
	}

	if filepath.Base(path) != "deploy.grlx" {
		t.Fatalf("expected deploy.grlx, got %s", filepath.Base(path))
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	if !strings.Contains(string(raw), "gcloud run deploy 'public-api'") {
		t.Fatalf("unexpected recipe contents:\n%s", string(raw))
	}
}

func TestRenderRecipeIncludesAdvancedDeploymentFlags(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Services: []Service{
			{
				Name:                 "identity-grpc",
				Image:                "us-central1-docker.pkg.dev/sample-project/platform-services/identity-grpc:latest",
				Protocol:             "grpc",
				Command:              "/app/server",
				Args:                 []string{"serve", "--config", "/etc/cohora/config.yaml"},
				ServiceAccount:       "identity-grpc@sample-project.iam.gserviceaccount.com",
				Port:                 50051,
				CPU:                  "2",
				Memory:               "1Gi",
				Concurrency:          25,
				MinInstances:         1,
				MaxInstances:         6,
				Timeout:              "600s",
				RevisionSuffix:       "candidate",
				TrafficPercent:       100,
				CPUThrottling:        boolPtr(false),
				StartupCPUBoost:      boolPtr(true),
				ExecutionEnvironment: "gen2",
				UseHTTP2:             boolPtr(true),
				AllowUnauthenticated: boolPtr(false),
				Ingress:              "internal",
				VPCConnector:         "projects/sample-project/locations/us-central1/connectors/core",
				VPCEgress:            "private-ranges-only",
				CloudSQLInstances:    []string{"sample-project:us-central1:primary-db"},
				Secrets: []Secret{
					{Target: "DB_PASSWORD", Secret: "db-password", Version: "5"},
				},
				StartupProbe: &Probe{
					Type:                "grpc",
					Port:                50051,
					InitialDelaySeconds: 5,
					PeriodSeconds:       10,
				},
				LivenessProbe: &Probe{
					Type:             "grpc",
					Port:             50051,
					FailureThreshold: 3,
				},
				Env: map[string]string{
					"APP_ENV": "production",
				},
				Labels: map[string]string{
					"tier": "backend",
				},
				Annotations: map[string]string{
					"run.googleapis.com/launch-stage": "BETA",
				},
			},
		},
	}

	recipe, err := RenderRecipe(cfg)
	if err != nil {
		t.Fatalf("RenderRecipe returned error: %v", err)
	}

	assertContains(t, recipe, "gcloud beta run deploy 'identity-grpc'")
	assertContains(t, recipe, "--command '/app/server'")
	assertContains(t, recipe, "--args 'serve,--config,/etc/cohora/config.yaml'")
	assertContains(t, recipe, "--revision-suffix 'candidate'")
	assertContains(t, recipe, "--no-cpu-throttling")
	assertContains(t, recipe, "--startup-cpu-boost")
	assertContains(t, recipe, "--add-cloudsql-instances 'sample-project:us-central1:primary-db'")
	assertContains(t, recipe, "--update-secrets 'DB_PASSWORD=db-password:5'")
	assertContains(t, recipe, "--update-annotations 'run.googleapis.com/launch-stage=BETA'")
	assertContains(t, recipe, "--startup-probe-grpc-port 50051")
	assertContains(t, recipe, "--liveness-probe-grpc-port 50051")
	assertContains(t, recipe, "gcloud run services update-traffic 'identity-grpc'")
	assertContains(t, recipe, "--to-latest")
}

func assertNotContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("expected recipe not to contain %q\nrecipe:\n%s", needle, haystack)
	}
}
