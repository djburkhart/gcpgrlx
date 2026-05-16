package gcpgrlx

import (
	"strings"
	"testing"
)

func boolPtr(value bool) *bool {
	return &value
}

func TestParseConfigAppliesDefaults(t *testing.T) {
	raw := []byte(`
project_id: cohora-prod
region: us-central1
artifact_registry_repository: cohora
services:
  - name: api
`)

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if got := cfg.Services[0].Image; got != "us-central1-docker.pkg.dev/cohora-prod/cohora/api:latest" {
		t.Fatalf("unexpected default image: %s", got)
	}
	if got := cfg.Services[0].Port; got != 8080 {
		t.Fatalf("unexpected default port: %d", got)
	}
}

func TestValidateRejectsBadServiceName(t *testing.T) {
	cfg := Config{
		ProjectID:                  "cohora-prod",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "cohora",
		Services: []Service{
			{Name: "Bad_Name", Image: "us-central1-docker.pkg.dev/cohora-prod/cohora/bad:latest", Port: 8080, CPU: "1", Memory: "512Mi", Timeout: "300s", Ingress: "all"},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid service name")
	}
}

func TestRenderRecipeIncludesCloudRunDeployFlags(t *testing.T) {
	cfg := Config{
		ProjectID:                  "cohora-prod",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "cohora",
		Services: []Service{
			{
				Name:                 "identity-api",
				Image:                "us-central1-docker.pkg.dev/cohora-prod/cohora/identity-api:latest",
				ServiceAccount:       "identity@cohora-prod.iam.gserviceaccount.com",
				Port:                 8081,
				CPU:                  "2",
				Memory:               "1Gi",
				MinInstances:         1,
				MaxInstances:         5,
				Timeout:              "600s",
				Ingress:              "internal",
				AllowUnauthenticated: boolPtr(false),
				Env: map[string]string{
					"APP_ENV":   "prod",
					"GRPC_PORT": "50051",
				},
				Labels: map[string]string{
					"service": "identity",
					"stack":   "cohora",
				},
			},
		},
	}

	recipe, err := RenderRecipe(cfg)
	if err != nil {
		t.Fatalf("RenderRecipe returned error: %v", err)
	}

	assertContains(t, recipe, "gcloud services enable")
	assertContains(t, recipe, "file.directory:")
	assertContains(t, recipe, "gcloud run deploy 'identity-api'")
	assertContains(t, recipe, "--no-allow-unauthenticated")
	assertContains(t, recipe, "--service-account 'identity@cohora-prod.iam.gserviceaccount.com'")
	assertContains(t, recipe, "--set-env-vars 'APP_ENV=prod,GRPC_PORT=50051'")
	assertContains(t, recipe, "--labels 'service=identity,stack=cohora'")
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected recipe to contain %q\nrecipe:\n%s", needle, haystack)
	}
}
