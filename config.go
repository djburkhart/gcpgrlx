package gcpgrlx

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`)

type Config struct {
	ProjectID                  string    `yaml:"project_id"`
	Region                     string    `yaml:"region"`
	ArtifactRegistryRepository string    `yaml:"artifact_registry_repository"`
	DeploymentRoot             string    `yaml:"deployment_root,omitempty"`
	RequiredServices           []string  `yaml:"required_services,omitempty"`
	Services                   []Service `yaml:"services"`
}

type Service struct {
	Name                 string            `yaml:"name"`
	Image                string            `yaml:"image,omitempty"`
	Protocol             string            `yaml:"protocol,omitempty"`
	ServiceAccount       string            `yaml:"service_account,omitempty"`
	Port                 int               `yaml:"port,omitempty"`
	CPU                  string            `yaml:"cpu,omitempty"`
	Memory               string            `yaml:"memory,omitempty"`
	Concurrency          int               `yaml:"concurrency,omitempty"`
	MinInstances         int               `yaml:"min_instances,omitempty"`
	MaxInstances         int               `yaml:"max_instances,omitempty"`
	Timeout              string            `yaml:"timeout,omitempty"`
	UseHTTP2             *bool             `yaml:"use_http2,omitempty"`
	AllowUnauthenticated *bool             `yaml:"allow_unauthenticated,omitempty"`
	Ingress              string            `yaml:"ingress,omitempty"`
	VPCConnector         string            `yaml:"vpc_connector,omitempty"`
	VPCEgress            string            `yaml:"vpc_egress,omitempty"`
	Env                  map[string]string `yaml:"env,omitempty"`
	Labels               map[string]string `yaml:"labels,omitempty"`
}

func ParseConfig(raw []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg.setDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) setDefaults() {
	if strings.TrimSpace(c.DeploymentRoot) == "" {
		c.DeploymentRoot = "/var/lib/gcpgrlx"
	}
	if len(c.RequiredServices) == 0 {
		c.RequiredServices = []string{
			"run.googleapis.com",
			"artifactregistry.googleapis.com",
		}
	}
	for index := range c.Services {
		service := &c.Services[index]
		if strings.TrimSpace(service.Protocol) == "" {
			service.Protocol = "http"
		}
		if service.Port == 0 {
			service.Port = 8080
		}
		if strings.TrimSpace(service.CPU) == "" {
			service.CPU = "1"
		}
		if strings.TrimSpace(service.Memory) == "" {
			service.Memory = "512Mi"
		}
		if service.MaxInstances == 0 {
			service.MaxInstances = 3
		}
		if strings.TrimSpace(service.Timeout) == "" {
			service.Timeout = "300s"
		}
		if service.Concurrency == 0 {
			if service.Protocol == "grpc" {
				service.Concurrency = 20
			} else {
				service.Concurrency = 80
			}
		}
		if strings.TrimSpace(service.Ingress) == "" {
			if service.Protocol == "grpc" {
				service.Ingress = "internal"
			} else {
				service.Ingress = "all"
			}
		}
		if service.Protocol == "grpc" {
			if service.UseHTTP2 == nil {
				value := true
				service.UseHTTP2 = &value
			}
			if service.AllowUnauthenticated == nil {
				value := false
				service.AllowUnauthenticated = &value
			}
		}
		if service.Env == nil {
			service.Env = map[string]string{}
		}
		if service.Labels == nil {
			service.Labels = map[string]string{}
		}
		if strings.TrimSpace(service.Image) == "" &&
			strings.TrimSpace(c.ProjectID) != "" &&
			strings.TrimSpace(c.Region) != "" &&
			strings.TrimSpace(c.ArtifactRegistryRepository) != "" &&
			strings.TrimSpace(service.Name) != "" {
			service.Image = defaultImage(c.Region, c.ProjectID, c.ArtifactRegistryRepository, service.Name)
		}
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ProjectID) == "" {
		return fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(c.Region) == "" {
		return fmt.Errorf("region is required")
	}
	if strings.TrimSpace(c.ArtifactRegistryRepository) == "" {
		return fmt.Errorf("artifact_registry_repository is required")
	}
	if len(c.Services) == 0 {
		return fmt.Errorf("at least one service is required")
	}

	seen := map[string]struct{}{}
	for _, service := range c.Services {
		if !serviceNamePattern.MatchString(service.Name) {
			return fmt.Errorf("service %q must be a valid Cloud Run service name", service.Name)
		}
		if _, exists := seen[service.Name]; exists {
			return fmt.Errorf("service %q is duplicated", service.Name)
		}
		seen[service.Name] = struct{}{}

		if service.Port <= 0 {
			return fmt.Errorf("service %q port must be greater than zero", service.Name)
		}
		if service.Protocol != "http" && service.Protocol != "grpc" {
			return fmt.Errorf("service %q protocol must be http or grpc", service.Name)
		}
		if service.Concurrency <= 0 {
			return fmt.Errorf("service %q concurrency must be greater than zero", service.Name)
		}
		if service.MinInstances < 0 {
			return fmt.Errorf("service %q min_instances cannot be negative", service.Name)
		}
		if service.MaxInstances < 0 {
			return fmt.Errorf("service %q max_instances cannot be negative", service.Name)
		}
		if service.MaxInstances > 0 && service.MaxInstances < service.MinInstances {
			return fmt.Errorf("service %q max_instances must be greater than or equal to min_instances", service.Name)
		}
		if strings.TrimSpace(service.Image) == "" {
			return fmt.Errorf("service %q image cannot be empty", service.Name)
		}
		if service.Ingress != "all" && service.Ingress != "internal" && service.Ingress != "internal-and-cloud-load-balancing" {
			return fmt.Errorf("service %q ingress must be all, internal, or internal-and-cloud-load-balancing", service.Name)
		}
		if service.VPCEgress != "" && service.VPCEgress != "all-traffic" && service.VPCEgress != "private-ranges-only" {
			return fmt.Errorf("service %q vpc_egress must be all-traffic or private-ranges-only", service.Name)
		}
		if service.Protocol == "grpc" && service.UseHTTP2 != nil && !*service.UseHTTP2 {
			return fmt.Errorf("service %q grpc services must enable use_http2", service.Name)
		}
	}

	return nil
}

func (c Config) SortedRequiredServices() []string {
	values := append([]string(nil), c.RequiredServices...)
	sort.Strings(values)
	return values
}

func (s Service) SortedEnvKeys() []string {
	keys := make([]string, 0, len(s.Env))
	for key := range s.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s Service) SortedLabelKeys() []string {
	keys := make([]string, 0, len(s.Labels))
	for key := range s.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func defaultImage(region string, projectID string, repository string, serviceName string) string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s:latest", region, projectID, repository, serviceName)
}
