package gcpgrlx

import "gopkg.in/yaml.v3"

// SampleConfig returns a ready-to-edit deployment spec for common gRPC services.
func SampleConfig() Config {
	disablePublic := false
	enableHTTP2 := true
	disableCPUThrottling := false
	enableStartupCPUBoost := true

	return Config{
		ProjectID:                  "sample-platform-prod",
		Region:                     "us-central1",
		ArtifactRegistryRepository: "platform-services",
		DeploymentRoot:             "/var/lib/gcpgrlx/platform",
		RequiredServices: []string{
			"artifactregistry.googleapis.com",
			"run.googleapis.com",
		},
		Services: []Service{
			{
				Name:                 "identity-grpc",
				Protocol:             "grpc",
				Command:              "/app/identity",
				Args:                 []string{"serve", "--config", "/etc/platform/config.yaml"},
				ServiceAccount:       "identity-grpc@sample-platform-prod.iam.gserviceaccount.com",
				Port:                 50051,
				CPU:                  "1",
				Memory:               "512Mi",
				Concurrency:          25,
				MinInstances:         1,
				MaxInstances:         6,
				Timeout:              "300s",
				RevisionSuffix:       "stable",
				TrafficPercent:       100,
				CPUThrottling:        &disableCPUThrottling,
				StartupCPUBoost:      &enableStartupCPUBoost,
				ExecutionEnvironment: "gen2",
				UseHTTP2:             &enableHTTP2,
				AllowUnauthenticated: &disablePublic,
				Ingress:              "internal",
				VPCConnector:         "projects/sample-platform-prod/locations/us-central1/connectors/core",
				VPCEgress:            "private-ranges-only",
				CloudSQLInstances:    []string{"sample-platform-prod:us-central1:identity-db"},
				Secrets: []Secret{
					{Target: "DB_PASSWORD", Secret: "identity-db-password", Version: "latest"},
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
					"APP_ENV":   "production",
					"GRPC_PORT": ":50051",
				},
				Labels: map[string]string{
					"app":      "platform",
					"protocol": "grpc",
					"tier":     "control-plane",
				},
			},
			{
				Name:                 "matching-grpc",
				Protocol:             "grpc",
				ServiceAccount:       "matching-grpc@sample-platform-prod.iam.gserviceaccount.com",
				Port:                 50051,
				CPU:                  "2",
				Memory:               "1Gi",
				Concurrency:          40,
				MinInstances:         0,
				MaxInstances:         10,
				Timeout:              "300s",
				NoTraffic:            true,
				ExecutionEnvironment: "gen2",
				UseHTTP2:             &enableHTTP2,
				AllowUnauthenticated: &disablePublic,
				Ingress:              "internal",
				VPCConnector:         "projects/sample-platform-prod/locations/us-central1/connectors/core",
				VPCEgress:            "private-ranges-only",
				Annotations: map[string]string{
					"run.googleapis.com/launch-stage": "BETA",
				},
				Env: map[string]string{
					"APP_ENV":        "production",
					"MATCH_PIPELINE": "default",
				},
				Labels: map[string]string{
					"app":      "platform",
					"protocol": "grpc",
					"tier":     "matching",
				},
			},
		},
	}
}

// SampleConfigYAML renders the sample config in the same YAML format expected by ParseConfig.
func SampleConfigYAML() ([]byte, error) {
	return yaml.Marshal(SampleConfig())
}
