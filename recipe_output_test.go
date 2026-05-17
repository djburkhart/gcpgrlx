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
				Args:                 []string{"serve", "--config", "/etc/example/config.yaml"},
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
	assertContains(t, recipe, "--args 'serve,--config,/etc/example/config.yaml'")
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

func TestRenderRecipeIncludesWorkerAndCronProfiles(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Services: []Service{
			{
				Name:           "queue-worker",
				Profile:        "worker",
				Image:          "us-central1-docker.pkg.dev/sample-project/platform-services/queue-worker:latest",
				Command:        "/app/worker",
				Args:           []string{"run", "--queue", "events"},
				ServiceAccount: "queue-worker@sample-project.iam.gserviceaccount.com",
				CPU:            "1",
				Memory:         "512Mi",
				Timeout:        "1800s",
				Tasks:          2,
				Parallelism:    1,
				MaxRetries:     4,
				Env: map[string]string{
					"QUEUE_NAME": "events",
				},
			},
			{
				Name:           "nightly-sync",
				Profile:        "cron",
				Image:          "us-central1-docker.pkg.dev/sample-project/platform-services/nightly-sync:latest",
				Command:        "/app/sync",
				Args:           []string{"nightly"},
				ServiceAccount: "nightly-sync@sample-project.iam.gserviceaccount.com",
				CPU:            "1",
				Memory:         "512Mi",
				Timeout:        "1800s",
				Tasks:          1,
				Parallelism:    1,
				MaxRetries:     2,
				Cron: &CronProfile{
					Schedule:       "0 3 * * *",
					TimeZone:       "UTC",
					ServiceAccount: "scheduler@sample-project.iam.gserviceaccount.com",
					JobName:        "nightly-sync-schedule",
				},
			},
		},
	}

	recipe, err := RenderRecipe(cfg)
	if err != nil {
		t.Fatalf("RenderRecipe returned error: %v", err)
	}

	assertContains(t, recipe, "gcloud run jobs deploy 'queue-worker'")
	assertContains(t, recipe, "--tasks 2")
	assertContains(t, recipe, "--parallelism 1")
	assertContains(t, recipe, "--max-retries 4")
	assertContains(t, recipe, "--task-timeout '1800s'")
	assertContains(t, recipe, "--args 'run,--queue,events'")
	assertContains(t, recipe, "gcloud run jobs deploy 'nightly-sync'")
	assertContains(t, recipe, "Schedule nightly-sync")
	assertContains(t, recipe, "gcloud scheduler jobs create http")
	assertContains(t, recipe, "nightly-sync-schedule")
	assertContains(t, recipe, "0 3 * * *")
	assertContains(t, recipe, "scheduler@sample-project.iam.gserviceaccount.com")
	assertContains(t, recipe, "https://run.googleapis.com/v2/projects/sample-project/locations/us-central1/jobs/nightly-sync:run")
}

func TestRenderCaddyfileUsesOverridesAndHeaders(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Caddy: &Caddy{
			Enabled: true,
			File:    "caddy/Caddyfile",
		},
		Services: []Service{
			{
				Name:        "identity-grpc",
				Image:       "us-central1-docker.pkg.dev/sample-project/platform-services/identity-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain: "api.example.com",
					Path:   "/identity",
					Headers: map[string]string{
						"Cache-Control": "no-store",
					},
				},
			},
			{
				Name:        "matching-grpc",
				Image:       "us-central1-docker.pkg.dev/sample-project/platform-services/matching-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain:   "api.example.com",
					Path:     "/matching",
					Upstream: "https://matching.example.internal",
				},
			},
		},
	}

	rendered, err := RenderCaddyfile(cfg)
	if err != nil {
		t.Fatalf("RenderCaddyfile returned error: %v", err)
	}

	assertContains(t, rendered, "api.example.com {")
	assertContains(t, rendered, "handle_path /identity/* {")
	assertContains(t, rendered, "Cache-Control 'no-store'")
	assertContains(t, rendered, "reverse_proxy https://matching.example.internal")
}

func TestRenderCaddyfileSupportsSnippetsAndTemplate(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Caddy: &Caddy{
			Enabled: true,
			Snippets: map[string]string{
				"common-security": "header X-Frame-Options DENY",
				"cors":            "header Access-Control-Allow-Origin *",
			},
			Template: `{{- range .Snippets -}}
({{ .Name }}) {
{{ indent .Body "    " }}
}
{{ end -}}
{{- range .Domains }}
{{ .Name }} {
{{- range .Routes }}
    route {{ .Matcher }} {
{{- range .Imports }}
        import {{ . }}
{{- end }}
        reverse_proxy {{ .Upstream }}
    }
{{- end }}
}
{{ end -}}`,
		},
		Services: []Service{
			{
				Name:        "identity-grpc",
				Image:       "us-central1-docker.pkg.dev/sample-project/platform-services/identity-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain:   "api.example.com",
					Path:     "/identity",
					Snippets: []string{"common-security"},
					Cohorts:  []string{"cors"},
				},
			},
		},
	}

	rendered, err := RenderCaddyfile(cfg)
	if err != nil {
		t.Fatalf("RenderCaddyfile returned error: %v", err)
	}

	assertContains(t, rendered, "(common-security) {")
	assertContains(t, rendered, "(cors) {")
	assertContains(t, rendered, "route /identity/* {")
	assertContains(t, rendered, "import common-security")
	assertContains(t, rendered, "import cors")
}

func TestRenderCaddyfileSupportsMiddlewarePresets(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Caddy: &Caddy{
			Enabled: true,
		},
		Services: []Service{
			{
				Name:        "identity-grpc",
				Image:       "us-central1-docker.pkg.dev/sample-project/platform-services/identity-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain:  "api.example.com",
					Path:    "/identity",
					Presets: []string{"compression", "security-headers", "no-store"},
				},
			},
		},
	}

	rendered, err := RenderCaddyfile(cfg)
	if err != nil {
		t.Fatalf("RenderCaddyfile returned error: %v", err)
	}

	assertContains(t, rendered, "(compression) {")
	assertContains(t, rendered, "encode zstd gzip")
	assertContains(t, rendered, "(security-headers) {")
	assertContains(t, rendered, "header X-Frame-Options DENY")
	assertContains(t, rendered, "(no-store) {")
	assertContains(t, rendered, "import compression")
	assertContains(t, rendered, "import security-headers")
	assertContains(t, rendered, "import no-store")
}

func TestRenderRecipeIncludesLiveCaddyPatchStep(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		DeploymentRoot:             "/var/lib/gcpgrlx/platform",
		Caddy: &Caddy{
			Enabled: true,
			File:    "caddy/Caddyfile",
		},
		Services: []Service{
			{
				Name:        "identity-grpc",
				Image:       "us-central1-docker.pkg.dev/sample-project/platform-services/identity-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain: "api.example.com",
					Path:   "/identity",
				},
			},
		},
	}

	recipe, err := RenderRecipe(cfg)
	if err != nil {
		t.Fatalf("RenderRecipe returned error: %v", err)
	}

	assertContains(t, recipe, "Patch live Caddyfile")
	assertContains(t, recipe, "Install Caddy")
	assertContains(t, recipe, "dl.cloudsmith.io/public/caddy/stable")
	assertContains(t, recipe, "gcloud run services describe")
	assertContains(t, recipe, "value(status.url)")
	assertContains(t, recipe, "Apply Caddyfile")
	assertContains(t, recipe, "install -D -m 0644")
	assertContains(t, recipe, "/etc/caddy/Caddyfile")
	assertContains(t, recipe, "Validate Caddy config")
	assertContains(t, recipe, "caddy validate")
	assertContains(t, recipe, "Reload Caddy")
	assertContains(t, recipe, "systemctl enable --now caddy")
	assertContains(t, recipe, "/var/lib/gcpgrlx/platform/caddy/Caddyfile")
	assertContains(t, recipe, "__GCPGRLX_UPSTREAM_IDENTITY_GRPC__")
}

func TestRenderRecipeIncludesSmokeTests(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		DeploymentRoot:             "/var/lib/gcpgrlx/platform",
		Caddy: &Caddy{
			Enabled: true,
			File:    "caddy/Caddyfile",
		},
		SmokeTests: []SmokeTest{
			{
				Name:           "identity edge route",
				Service:        "identity-grpc",
				Path:           "/identity/healthz",
				ExpectedStatus: 200,
				BodyContains:   "ok",
				SkipTLSVerify:  true,
				Timeout:        "90s",
			},
			{
				Name:    "custom command",
				Type:    "command",
				Command: "bash -lc 'echo smoke works'",
				Timeout: "1m",
			},
		},
		Services: []Service{
			{
				Name:        "identity-grpc",
				Image:       "us-central1-docker.pkg.dev/sample-project/platform-services/identity-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Caddy: &ServiceCaddy{
					Domain: "api.example.com",
					Path:   "/identity",
				},
			},
		},
	}

	recipe, err := RenderRecipe(cfg)
	if err != nil {
		t.Fatalf("RenderRecipe returned error: %v", err)
	}

	assertContains(t, recipe, "Smoke test identity edge route")
	assertContains(t, recipe, "https://api.example.com")
	assertContains(t, recipe, "/identity/healthz")
	assertContains(t, recipe, "curl -sS -o")
	assertContains(t, recipe, "smoke test identity edge route expected HTTP 200")
	assertContains(t, recipe, "smoke test identity edge route missing expected body text")
	assertContains(t, recipe, "timeout: \"90s\"")
	assertContains(t, recipe, "Smoke test custom command")
	assertContains(t, recipe, "bash -lc 'echo smoke works'")

	reloadIndex := strings.Index(recipe, "Reload Caddy")
	smokeIndex := strings.Index(recipe, "Smoke test identity edge route")
	if reloadIndex == -1 || smokeIndex == -1 || smokeIndex <= reloadIndex {
		t.Fatalf("expected smoke tests after caddy reload\nrecipe:\n%s", recipe)
	}
}

func TestWriteCaddyFileCreatesConfiguredPath(t *testing.T) {
	cfg := SampleConfig()
	outDir := t.TempDir()

	path, err := WriteCaddyFile(cfg, outDir)
	if err != nil {
		t.Fatalf("WriteCaddyFile returned error: %v", err)
	}

	if filepath.Base(path) != "Caddyfile" {
		t.Fatalf("expected Caddyfile, got %s", filepath.Base(path))
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	assertContains(t, string(raw), "api.sample-platform.dev {")
	assertContains(t, string(raw), "handle_path /identity/* {")
}

func assertNotContains(t *testing.T, haystack string, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("expected recipe not to contain %q\nrecipe:\n%s", needle, haystack)
	}
}
