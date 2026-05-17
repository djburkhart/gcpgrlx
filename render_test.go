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
project_id: example-prod
region: us-central1
artifact_registry_repository: example
caddy:
  enabled: true
smoke_tests:
  - service: api
services:
  - name: api
    caddy:
      domain: api.example.dev
`)

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	if got := cfg.Services[0].Image; got != "us-central1-docker.pkg.dev/example-prod/example/api:latest" {
		t.Fatalf("unexpected default image: %s", got)
	}
	if got := cfg.Services[0].Port; got != 8080 {
		t.Fatalf("unexpected default port: %d", got)
	}
	if got := cfg.Services[0].Protocol; got != "http" {
		t.Fatalf("unexpected default protocol: %s", got)
	}
	if cfg.Caddy == nil || cfg.Caddy.File != "Caddyfile" {
		t.Fatalf("unexpected default caddy file: %#v", cfg.Caddy)
	}
	if cfg.Services[0].Caddy == nil || cfg.Services[0].Caddy.Path != "/" {
		t.Fatalf("unexpected default caddy path: %#v", cfg.Services[0].Caddy)
	}
	if len(cfg.SmokeTests) != 1 {
		t.Fatalf("unexpected smoke test count: %d", len(cfg.SmokeTests))
	}
	if cfg.SmokeTests[0].Type != "http" || cfg.SmokeTests[0].Method != "GET" {
		t.Fatalf("unexpected smoke test defaults: %#v", cfg.SmokeTests[0])
	}
	if cfg.SmokeTests[0].Path != "/" || cfg.SmokeTests[0].ExpectedStatus != 200 {
		t.Fatalf("unexpected smoke test path/status defaults: %#v", cfg.SmokeTests[0])
	}
}

func TestValidateRejectsBadServiceName(t *testing.T) {
	cfg := Config{
		ProjectID:                  "example-prod",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "example",
		Services: []Service{
			{Name: "Bad_Name", Image: "us-central1-docker.pkg.dev/example-prod/example/bad:latest", Port: 8080, CPU: "1", Memory: "512Mi", Timeout: "300s", Ingress: "all"},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid service name")
	}
}

func TestParseConfigAppliesGRPCDefaults(t *testing.T) {
	raw := []byte(`
project_id: grpc-prod
region: us-central1
artifact_registry_repository: grpc-services
services:
  - name: identity-grpc
    protocol: grpc
`)

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	service := cfg.Services[0]
	if service.Concurrency != 20 {
		t.Fatalf("unexpected grpc concurrency: %d", service.Concurrency)
	}
	if service.Ingress != "internal" {
		t.Fatalf("unexpected grpc ingress: %s", service.Ingress)
	}
	if service.UseHTTP2 == nil || !*service.UseHTTP2 {
		t.Fatal("expected grpc service to enable http2")
	}
	if service.AllowUnauthenticated == nil || *service.AllowUnauthenticated {
		t.Fatal("expected grpc service to disable unauthenticated access by default")
	}
}

func TestParseConfigAppliesWorkerAndCronDefaults(t *testing.T) {
	raw := []byte(`
project_id: jobs-prod
region: us-central1
artifact_registry_repository: jobs
services:
  - name: queue-worker
    profile: worker
  - name: nightly-sync
    profile: cron
    service_account: nightly-sync@jobs-prod.iam.gserviceaccount.com
    cron:
      schedule: 0 3 * * *
`)

	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}

	worker := cfg.Services[0]
	if worker.Tasks != 1 || worker.Parallelism != 1 || worker.MaxRetries != 3 {
		t.Fatalf("unexpected worker defaults: %#v", worker)
	}
	if worker.Port != 0 {
		t.Fatalf("expected worker port to remain unset, got %d", worker.Port)
	}

	cron := cfg.Services[1]
	if cron.Cron == nil || cron.Cron.JobName != "nightly-sync-schedule" || cron.Cron.TimeZone != "UTC" {
		t.Fatalf("unexpected cron defaults: %#v", cron.Cron)
	}
	if cron.Cron.ServiceAccount != "nightly-sync@jobs-prod.iam.gserviceaccount.com" {
		t.Fatalf("unexpected cron scheduler service account: %#v", cron.Cron)
	}
}

func TestRenderCaddyfileUsesDefaultUpstream(t *testing.T) {
	cfg := Config{
		ProjectID:                  "example-prod",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "example",
		Caddy: &Caddy{
			Enabled: true,
		},
		Services: []Service{
			{
				Name:        "identity-api",
				Protocol:    "grpc",
				Image:       "us-central1-docker.pkg.dev/example-prod/example/identity-api:latest",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain: "api.example.dev",
					Path:   "/identity",
				},
			},
		},
	}

	rendered, err := RenderCaddyfile(cfg)
	if err != nil {
		t.Fatalf("RenderCaddyfile returned error: %v", err)
	}

	assertContains(t, rendered, "api.example.dev {")
	assertContains(t, rendered, "handle_path /identity/* {")
	assertContains(t, rendered, "reverse_proxy https://identity-api-us-central1.a.run.app")
}

func TestRenderRecipeIncludesCloudRunDeployFlags(t *testing.T) {
	cfg := Config{
		ProjectID:                  "example-prod",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "example",
		Services: []Service{
			{
				Name:                 "identity-api",
				Protocol:             "grpc",
				Image:                "us-central1-docker.pkg.dev/example-prod/example/identity-api:latest",
				ServiceAccount:       "identity@example-prod.iam.gserviceaccount.com",
				Port:                 8081,
				CPU:                  "2",
				Memory:               "1Gi",
				Concurrency:          25,
				MinInstances:         1,
				MaxInstances:         5,
				Timeout:              "600s",
				Ingress:              "internal",
				UseHTTP2:             boolPtr(true),
				AllowUnauthenticated: boolPtr(false),
				VPCConnector:         "projects/example-prod/locations/us-central1/connectors/core",
				VPCEgress:            "private-ranges-only",
				Env: map[string]string{
					"APP_ENV":   "prod",
					"GRPC_PORT": "50051",
				},
				Labels: map[string]string{
					"service": "identity",
					"stack":   "example",
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
	assertContains(t, recipe, "--use-http2")
	assertContains(t, recipe, "--concurrency 25")
	assertContains(t, recipe, "--vpc-connector 'projects/example-prod/locations/us-central1/connectors/core'")
	assertContains(t, recipe, "--vpc-egress 'private-ranges-only'")
	assertContains(t, recipe, "--service-account 'identity@example-prod.iam.gserviceaccount.com'")
	assertContains(t, recipe, "--set-env-vars 'APP_ENV=prod,GRPC_PORT=50051'")
	assertContains(t, recipe, "--labels 'service=identity,stack=example'")
}

func assertContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected recipe to contain %q\nrecipe:\n%s", needle, haystack)
	}
}
