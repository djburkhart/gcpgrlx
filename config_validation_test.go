package gcpgrlx

import "testing"

func TestValidateRejectsInvalidServiceConfigurations(t *testing.T) {
	tests := []struct {
		name    string
		service Service
	}{
		{
			name: "invalid protocol",
			service: Service{
				Name:        "api",
				Image:       "us-central1-docker.pkg.dev/sample/repo/api:latest",
				Protocol:    "tcp",
				Port:        8080,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 10,
				Timeout:     "300s",
				Ingress:     "all",
			},
		},
		{
			name: "invalid vpc egress",
			service: Service{
				Name:        "api",
				Image:       "us-central1-docker.pkg.dev/sample/repo/api:latest",
				Protocol:    "http",
				Port:        8080,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 10,
				Timeout:     "300s",
				Ingress:     "all",
				VPCEgress:   "internet-only",
			},
		},
		{
			name: "grpc without http2",
			service: Service{
				Name:                 "identity-grpc",
				Image:                "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
				Protocol:             "grpc",
				Port:                 50051,
				CPU:                  "1",
				Memory:               "512Mi",
				Concurrency:          20,
				Timeout:              "300s",
				Ingress:              "internal",
				UseHTTP2:             boolPtr(false),
				AllowUnauthenticated: boolPtr(false),
			},
		},
		{
			name: "traffic and no traffic together",
			service: Service{
				Name:           "api",
				Image:          "us-central1-docker.pkg.dev/sample/repo/api:latest",
				Protocol:       "http",
				Port:           8080,
				CPU:            "1",
				Memory:         "512Mi",
				Concurrency:    80,
				Timeout:        "300s",
				Ingress:        "all",
				TrafficPercent: 25,
				NoTraffic:      true,
			},
		},
		{
			name: "max instances lower than min",
			service: Service{
				Name:         "worker",
				Image:        "us-central1-docker.pkg.dev/sample/repo/worker:latest",
				Protocol:     "http",
				Port:         8080,
				CPU:          "1",
				Memory:       "512Mi",
				Concurrency:  80,
				MinInstances: 3,
				MaxInstances: 1,
				Timeout:      "300s",
				Ingress:      "all",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Services:                   []Service{test.service},
			}

			if err := cfg.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateRejectsDuplicateServices(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Services: []Service{
			{
				Name:        "public-api",
				Image:       "us-central1-docker.pkg.dev/sample/repo/public-api:latest",
				Protocol:    "http",
				Port:        8080,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 80,
				Timeout:     "300s",
				Ingress:     "all",
			},
			{
				Name:        "public-api",
				Image:       "us-central1-docker.pkg.dev/sample/repo/public-api:v2",
				Protocol:    "http",
				Port:        8080,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 80,
				Timeout:     "300s",
				Ingress:     "all",
			},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duplicate service validation error")
	}
}

func TestValidateRejectsInvalidProbesAndSecrets(t *testing.T) {
	cfg := Config{
		ProjectID:                  "sample-project",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		Services: []Service{
			{
				Name:        "identity-grpc",
				Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
				Protocol:    "grpc",
				Port:        50051,
				CPU:         "1",
				Memory:      "512Mi",
				Concurrency: 20,
				Timeout:     "300s",
				Ingress:     "internal",
				UseHTTP2:    boolPtr(true),
				Secrets: []Secret{
					{Target: "", Secret: "db-password"},
				},
				StartupProbe: &Probe{Type: "grpc", Port: 0},
			},
		},
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error for invalid secrets/probes")
	}
}

func TestValidateRejectsInvalidCaddyConfiguration(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "service caddy without top level enablement",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
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
						},
					},
				},
			},
		},
		{
			name: "missing caddy domain",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Caddy:                      &Caddy{Enabled: true},
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
						Protocol:    "grpc",
						Port:        50051,
						CPU:         "1",
						Memory:      "512Mi",
						Concurrency: 20,
						Timeout:     "300s",
						Ingress:     "internal",
						UseHTTP2:    boolPtr(true),
						Caddy: &ServiceCaddy{
							Path: "/identity",
						},
					},
				},
			},
		},
		{
			name: "duplicate caddy route",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Caddy:                      &Caddy{Enabled: true},
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
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
					{
						Name:        "matching-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/matching-grpc:latest",
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
			},
		},
		{
			name: "undefined caddy snippet",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Caddy:                      &Caddy{Enabled: true, Snippets: map[string]string{"common-security": "header X-Frame-Options DENY"}},
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
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
							Snippets: []string{"missing-snippet"},
						},
					},
				},
			},
		},
		{
			name: "undefined caddy preset",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Caddy:                      &Caddy{Enabled: true},
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
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
							Presets: []string{"unknown-preset"},
						},
					},
				},
			},
		},
		{
			name: "smoke test unknown service",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				SmokeTests: []SmokeTest{
					{Name: "missing-service", Service: "does-not-exist"},
				},
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
						Protocol:    "grpc",
						Port:        50051,
						CPU:         "1",
						Memory:      "512Mi",
						Concurrency: 20,
						Timeout:     "300s",
						Ingress:     "internal",
						UseHTTP2:    boolPtr(true),
					},
				},
			},
		},
		{
			name: "command smoke test missing command",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				SmokeTests: []SmokeTest{
					{Name: "bad-command", Type: "command"},
				},
				Services: []Service{
					{
						Name:        "identity-grpc",
						Image:       "us-central1-docker.pkg.dev/sample/repo/identity-grpc:latest",
						Protocol:    "grpc",
						Port:        50051,
						CPU:         "1",
						Memory:      "512Mi",
						Concurrency: 20,
						Timeout:     "300s",
						Ingress:     "internal",
						UseHTTP2:    boolPtr(true),
					},
				},
			},
		},
		{
			name: "cron missing schedule",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				Services: []Service{
					{
						Name:           "nightly-sync",
						Profile:        "cron",
						Image:          "us-central1-docker.pkg.dev/sample/repo/nightly-sync:latest",
						CPU:            "1",
						Memory:         "512Mi",
						Timeout:        "1800s",
						Tasks:          1,
						Parallelism:    1,
						MaxRetries:     2,
						ServiceAccount: "nightly-sync@sample-project.iam.gserviceaccount.com",
						Cron:           &CronProfile{},
					},
				},
			},
		},
		{
			name: "http smoke test cannot target worker",
			cfg: Config{
				ProjectID:                  "sample-project",
				Region:                     "us-central1",
				ArtifactRegistryRepository: "platform-services",
				SmokeTests: []SmokeTest{
					{Name: "worker-http", Service: "queue-worker"},
				},
				Services: []Service{
					{
						Name:           "queue-worker",
						Profile:        "worker",
						Image:          "us-central1-docker.pkg.dev/sample/repo/queue-worker:latest",
						CPU:            "1",
						Memory:         "512Mi",
						Timeout:        "1800s",
						Tasks:          1,
						Parallelism:    1,
						MaxRetries:     1,
						ServiceAccount: "queue-worker@sample-project.iam.gserviceaccount.com",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.cfg.setDefaults()
			if err := test.cfg.Validate(); err == nil {
				t.Fatal("expected caddy validation error")
			}
		})
	}
}
