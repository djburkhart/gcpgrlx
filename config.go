package gcpgrlx

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var serviceNamePattern = regexp.MustCompile(`^[a-z]([-a-z0-9]{0,61}[a-z0-9])?$`)

type Config struct {
	ProjectID                  string      `yaml:"project_id"`
	Region                     string      `yaml:"region"`
	ArtifactRegistryRepository string      `yaml:"artifact_registry_repository"`
	DeploymentRoot             string      `yaml:"deployment_root,omitempty"`
	Caddy                      *Caddy      `yaml:"caddy,omitempty"`
	RequiredServices           []string    `yaml:"required_services,omitempty"`
	SmokeTests                 []SmokeTest `yaml:"smoke_tests,omitempty"`
	Services                   []Service   `yaml:"services"`
}

type Caddy struct {
	Enabled  bool              `yaml:"enabled,omitempty"`
	File     string            `yaml:"file,omitempty"`
	Template string            `yaml:"template,omitempty"`
	Snippets map[string]string `yaml:"snippets,omitempty"`
}

type Service struct {
	Name                 string            `yaml:"name"`
	Profile              string            `yaml:"profile,omitempty"`
	Image                string            `yaml:"image,omitempty"`
	Protocol             string            `yaml:"protocol,omitempty"`
	Command              string            `yaml:"command,omitempty"`
	Args                 []string          `yaml:"args,omitempty"`
	ServiceAccount       string            `yaml:"service_account,omitempty"`
	Port                 int               `yaml:"port,omitempty"`
	CPU                  string            `yaml:"cpu,omitempty"`
	Memory               string            `yaml:"memory,omitempty"`
	Concurrency          int               `yaml:"concurrency,omitempty"`
	MinInstances         int               `yaml:"min_instances,omitempty"`
	MaxInstances         int               `yaml:"max_instances,omitempty"`
	Timeout              string            `yaml:"timeout,omitempty"`
	Tasks                int               `yaml:"tasks,omitempty"`
	Parallelism          int               `yaml:"parallelism,omitempty"`
	MaxRetries           int               `yaml:"max_retries,omitempty"`
	RevisionSuffix       string            `yaml:"revision_suffix,omitempty"`
	TrafficPercent       int               `yaml:"traffic_percent,omitempty"`
	NoTraffic            bool              `yaml:"no_traffic,omitempty"`
	CPUThrottling        *bool             `yaml:"cpu_throttling,omitempty"`
	StartupCPUBoost      *bool             `yaml:"startup_cpu_boost,omitempty"`
	ExecutionEnvironment string            `yaml:"execution_environment,omitempty"`
	UseHTTP2             *bool             `yaml:"use_http2,omitempty"`
	AllowUnauthenticated *bool             `yaml:"allow_unauthenticated,omitempty"`
	Ingress              string            `yaml:"ingress,omitempty"`
	VPCConnector         string            `yaml:"vpc_connector,omitempty"`
	VPCEgress            string            `yaml:"vpc_egress,omitempty"`
	CloudSQLInstances    []string          `yaml:"cloud_sql_instances,omitempty"`
	Secrets              []Secret          `yaml:"secrets,omitempty"`
	StartupProbe         *Probe            `yaml:"startup_probe,omitempty"`
	LivenessProbe        *Probe            `yaml:"liveness_probe,omitempty"`
	Cron                 *CronProfile      `yaml:"cron,omitempty"`
	Caddy                *ServiceCaddy     `yaml:"caddy,omitempty"`
	Env                  map[string]string `yaml:"env,omitempty"`
	Labels               map[string]string `yaml:"labels,omitempty"`
	Annotations          map[string]string `yaml:"annotations,omitempty"`
}

type ServiceCaddy struct {
	Domain   string            `yaml:"domain,omitempty"`
	Path     string            `yaml:"path,omitempty"`
	Upstream string            `yaml:"upstream,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Presets  []string          `yaml:"presets,omitempty"`
	Snippets []string          `yaml:"snippets,omitempty"`
	Cohorts  []string          `yaml:"cohorts,omitempty"`
}

type CronProfile struct {
	Schedule       string `yaml:"schedule,omitempty"`
	TimeZone       string `yaml:"time_zone,omitempty"`
	ServiceAccount string `yaml:"service_account,omitempty"`
	JobName        string `yaml:"job_name,omitempty"`
}

type Secret struct {
	Target  string `yaml:"target"`
	Secret  string `yaml:"secret"`
	Version string `yaml:"version,omitempty"`
}

type Probe struct {
	Type                string `yaml:"type,omitempty"`
	Path                string `yaml:"path,omitempty"`
	Port                int    `yaml:"port,omitempty"`
	InitialDelaySeconds int    `yaml:"initial_delay_seconds,omitempty"`
	PeriodSeconds       int    `yaml:"period_seconds,omitempty"`
	TimeoutSeconds      int    `yaml:"timeout_seconds,omitempty"`
	FailureThreshold    int    `yaml:"failure_threshold,omitempty"`
	SuccessThreshold    int    `yaml:"success_threshold,omitempty"`
}

type SmokeTest struct {
	Name           string            `yaml:"name,omitempty"`
	Service        string            `yaml:"service,omitempty"`
	Type           string            `yaml:"type,omitempty"`
	URL            string            `yaml:"url,omitempty"`
	Path           string            `yaml:"path,omitempty"`
	Method         string            `yaml:"method,omitempty"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	ExpectedStatus int               `yaml:"expected_status,omitempty"`
	BodyContains   string            `yaml:"body_contains,omitempty"`
	Command        string            `yaml:"command,omitempty"`
	Timeout        string            `yaml:"timeout,omitempty"`
	SkipTLSVerify  bool              `yaml:"skip_tls_verify,omitempty"`
}

func ParseConfig(raw []byte) (Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	// Normalize partially specified configs before enforcing required fields.
	cfg.setDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c *Config) setDefaults() {
	c.setGlobalDefaults()
	for index := range c.Services {
		c.setServiceDefaults(&c.Services[index])
	}
	for index := range c.SmokeTests {
		c.setSmokeTestDefaults(&c.SmokeTests[index])
	}
}

func (c Config) Validate() error {
	if err := c.validateRequiredFields(); err != nil {
		return err
	}
	if err := c.validateCaddyConfig(); err != nil {
		return err
	}
	if err := c.validateSmokeTests(); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	routes := map[string]struct{}{}
	for _, service := range c.Services {
		if err := validateServiceName(service.Name, seen); err != nil {
			return err
		}
		if err := validateService(c, service, routes); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) setGlobalDefaults() {
	c.DeploymentRoot = defaultTrimmed(c.DeploymentRoot, "/var/lib/gcpgrlx")
	if c.Caddy != nil {
		c.Caddy.File = defaultTrimmed(c.Caddy.File, "Caddyfile")
		if c.Caddy.Snippets == nil {
			c.Caddy.Snippets = map[string]string{}
		}
	}
	if len(c.RequiredServices) == 0 {
		c.RequiredServices = []string{
			"run.googleapis.com",
			"artifactregistry.googleapis.com",
		}
	}
	if c.hasCronProfiles() && !containsString(c.RequiredServices, "cloudscheduler.googleapis.com") {
		c.RequiredServices = append(c.RequiredServices, "cloudscheduler.googleapis.com")
	}
}

func (c Config) setServiceDefaults(service *Service) {
	service.Profile = defaultTrimmed(service.Profile, "service")
	service.CPU = defaultTrimmed(service.CPU, "1")
	service.Memory = defaultTrimmed(service.Memory, "512Mi")
	service.Timeout = defaultTrimmed(service.Timeout, "300s")
	service.ExecutionEnvironment = defaultTrimmed(service.ExecutionEnvironment, "gen2")
	if service.IsServiceProfile() {
		service.Protocol = defaultTrimmed(service.Protocol, "http")
		service.Port = defaultInt(service.Port, 8080)
		service.MaxInstances = defaultInt(service.MaxInstances, 3)
		service.Concurrency = defaultInt(service.Concurrency, defaultConcurrency(service.Protocol))
		service.Ingress = defaultTrimmed(service.Ingress, defaultIngress(service.Protocol))

		if service.Protocol == "grpc" {
			// Cloud Run gRPC needs HTTP/2 enabled, and internal auth is the safer default.
			service.UseHTTP2 = defaultBoolPointer(service.UseHTTP2, true)
			service.AllowUnauthenticated = defaultBoolPointer(service.AllowUnauthenticated, false)
		}
	} else {
		service.Tasks = defaultInt(service.Tasks, 1)
		service.Parallelism = defaultInt(service.Parallelism, 1)
		if service.MaxRetries == 0 {
			service.MaxRetries = 3
		}
		if service.IsCronProfile() {
			ensureCronDefaults(service)
		}
	}

	ensureServiceMaps(service)
	ensureServiceCaddyDefaults(service)
	if shouldSetDefaultImage(c, *service) {
		service.Image = defaultImage(c.Region, c.ProjectID, c.ArtifactRegistryRepository, service.Name)
	}
}

func (c Config) setSmokeTestDefaults(test *SmokeTest) {
	test.Name = defaultTrimmed(test.Name, defaultSmokeTestName(*test))
	test.Type = defaultTrimmed(test.Type, defaultSmokeTestType(*test))
	test.Method = defaultTrimmed(test.Method, "GET")
	test.Timeout = defaultTrimmed(test.Timeout, "2m")
	test.Path = defaultTrimmed(test.Path, defaultSmokeTestPath(c, *test))
	test.ExpectedStatus = defaultInt(test.ExpectedStatus, 200)
	if test.Headers == nil {
		test.Headers = map[string]string{}
	}
}

func (c Config) validateRequiredFields() error {
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
	return nil
}

func (c Config) validateCaddyConfig() error {
	if c.Caddy == nil {
		return nil
	}
	for name, body := range c.Caddy.Snippets {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("caddy snippet name cannot be empty")
		}
		if strings.TrimSpace(body) == "" {
			return fmt.Errorf("caddy snippet %q cannot be empty", name)
		}
	}
	return nil
}

func (c Config) validateSmokeTests() error {
	serviceNames := c.ServiceNames()
	seen := map[string]struct{}{}
	for _, test := range c.SmokeTests {
		if strings.TrimSpace(test.Name) == "" {
			return fmt.Errorf("smoke test name is required")
		}
		if _, exists := seen[test.Name]; exists {
			return fmt.Errorf("smoke test %q is duplicated", test.Name)
		}
		seen[test.Name] = struct{}{}
		if test.Type != "http" && test.Type != "command" {
			return fmt.Errorf("smoke test %q type must be http or command", test.Name)
		}
		if strings.TrimSpace(test.Timeout) == "" {
			return fmt.Errorf("smoke test %q timeout is required", test.Name)
		}
		switch test.Type {
		case "command":
			if strings.TrimSpace(test.Command) == "" {
				return fmt.Errorf("smoke test %q command is required for command tests", test.Name)
			}
		case "http":
			if strings.TrimSpace(test.URL) == "" && strings.TrimSpace(test.Service) == "" {
				return fmt.Errorf("smoke test %q must set service or url", test.Name)
			}
			if strings.TrimSpace(test.Service) != "" {
				if _, exists := serviceNames[test.Service]; !exists {
					return fmt.Errorf("smoke test %q references unknown service %q", test.Name, test.Service)
				}
				if service, ok := c.ServiceByName(test.Service); ok && !service.IsServiceProfile() {
					return fmt.Errorf("smoke test %q http checks require a service profile target, got %q", test.Name, service.Profile)
				}
			}
			if !isValidSmokeTestPath(test.Path) {
				return fmt.Errorf("smoke test %q path must start with /", test.Name)
			}
			if test.ExpectedStatus < 100 || test.ExpectedStatus > 599 {
				return fmt.Errorf("smoke test %q expected_status must be between 100 and 599", test.Name)
			}
			for key, value := range test.Headers {
				if strings.TrimSpace(key) == "" {
					return fmt.Errorf("smoke test %q header name cannot be empty", test.Name)
				}
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("smoke test %q header %q cannot be empty", test.Name, key)
				}
			}
		}
	}
	return nil
}

func validateServiceName(name string, seen map[string]struct{}) error {
	if !serviceNamePattern.MatchString(name) {
		return fmt.Errorf("service %q must be a valid Cloud Run service name", name)
	}
	if _, exists := seen[name]; exists {
		return fmt.Errorf("service %q is duplicated", name)
	}
	seen[name] = struct{}{}
	return nil
}

func validateService(cfg Config, service Service, routes map[string]struct{}) error {
	validators := []func(Config, Service, map[string]struct{}) error{
		validateServiceCore,
		validateServiceScaling,
		validateServiceRuntime,
		validateServiceSecrets,
		validateServiceProbes,
		validateServiceCron,
		validateServiceCaddy,
	}
	for _, validate := range validators {
		if err := validate(cfg, service, routes); err != nil {
			return err
		}
	}
	return nil
}

func validateServiceCore(_ Config, service Service, _ map[string]struct{}) error {
	if !isValidServiceProfile(service.Profile) {
		return fmt.Errorf("service %q profile must be service, worker, or cron", service.Name)
	}
	if service.IsServiceProfile() && service.Port <= 0 {
		return fmt.Errorf("service %q port must be greater than zero", service.Name)
	}
	if service.IsServiceProfile() && service.Protocol != "http" && service.Protocol != "grpc" {
		return fmt.Errorf("service %q protocol must be http or grpc", service.Name)
	}
	if strings.TrimSpace(service.Image) == "" {
		return fmt.Errorf("service %q image cannot be empty", service.Name)
	}
	if service.IsServiceProfile() && service.NoTraffic && service.TrafficPercent > 0 {
		return fmt.Errorf("service %q cannot set both no_traffic and traffic_percent", service.Name)
	}
	if service.IsServiceProfile() && (service.TrafficPercent < 0 || service.TrafficPercent > 100) {
		return fmt.Errorf("service %q traffic_percent must be between 0 and 100", service.Name)
	}
	if !service.IsServiceProfile() && (service.TrafficPercent > 0 || service.NoTraffic) {
		return fmt.Errorf("service %q profile %q cannot use traffic settings", service.Name, service.Profile)
	}
	return nil
}

func validateServiceScaling(_ Config, service Service, _ map[string]struct{}) error {
	if service.IsServiceProfile() {
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
		return nil
	}
	if service.Tasks <= 0 {
		return fmt.Errorf("service %q tasks must be greater than zero", service.Name)
	}
	if service.Parallelism <= 0 {
		return fmt.Errorf("service %q parallelism must be greater than zero", service.Name)
	}
	if service.MaxRetries < 0 {
		return fmt.Errorf("service %q max_retries cannot be negative", service.Name)
	}
	if service.MinInstances != 0 || service.MaxInstances != 0 {
		return fmt.Errorf("service %q profile %q cannot use min_instances or max_instances", service.Name, service.Profile)
	}
	return nil
}

func validateServiceRuntime(_ Config, service Service, _ map[string]struct{}) error {
	if !service.IsServiceProfile() {
		if service.Ingress != "" {
			return fmt.Errorf("service %q profile %q cannot use ingress", service.Name, service.Profile)
		}
		if service.UseHTTP2 != nil {
			return fmt.Errorf("service %q profile %q cannot use use_http2", service.Name, service.Profile)
		}
		if service.AllowUnauthenticated != nil {
			return fmt.Errorf("service %q profile %q cannot use allow_unauthenticated", service.Name, service.Profile)
		}
		return nil
	}
	if service.Ingress != "all" && service.Ingress != "internal" && service.Ingress != "internal-and-cloud-load-balancing" {
		return fmt.Errorf("service %q ingress must be all, internal, or internal-and-cloud-load-balancing", service.Name)
	}
	if service.VPCEgress != "" && service.VPCEgress != "all-traffic" && service.VPCEgress != "private-ranges-only" {
		return fmt.Errorf("service %q vpc_egress must be all-traffic or private-ranges-only", service.Name)
	}
	if service.ExecutionEnvironment != "" && service.ExecutionEnvironment != "gen1" && service.ExecutionEnvironment != "gen2" {
		return fmt.Errorf("service %q execution_environment must be gen1 or gen2", service.Name)
	}
	if service.Protocol == "grpc" && service.UseHTTP2 != nil && !*service.UseHTTP2 {
		return fmt.Errorf("service %q grpc services must enable use_http2", service.Name)
	}
	return nil
}

func validateServiceSecrets(_ Config, service Service, _ map[string]struct{}) error {
	for _, secret := range service.Secrets {
		if strings.TrimSpace(secret.Target) == "" {
			return fmt.Errorf("service %q secret target is required", service.Name)
		}
		if strings.TrimSpace(secret.Secret) == "" {
			return fmt.Errorf("service %q secret name is required", service.Name)
		}
	}
	return nil
}

func validateServiceProbes(_ Config, service Service, _ map[string]struct{}) error {
	if !service.IsServiceProfile() && (service.StartupProbe != nil || service.LivenessProbe != nil) {
		return fmt.Errorf("service %q profile %q cannot use startup_probe or liveness_probe", service.Name, service.Profile)
	}
	if err := validateProbe(service.Name, "startup_probe", service.StartupProbe); err != nil {
		return err
	}
	if err := validateProbe(service.Name, "liveness_probe", service.LivenessProbe); err != nil {
		return err
	}
	return nil
}

func validateServiceCaddy(cfg Config, service Service, routes map[string]struct{}) error {
	if service.Caddy == nil {
		return nil
	}
	if !service.IsServiceProfile() {
		return fmt.Errorf("service %q profile %q cannot use caddy", service.Name, service.Profile)
	}
	if !cfg.caddyEnabled() {
		return fmt.Errorf("service %q caddy block requires top-level caddy.enabled to be true", service.Name)
	}
	if strings.TrimSpace(service.Caddy.Domain) == "" {
		return fmt.Errorf("service %q caddy domain is required", service.Name)
	}
	if !isValidCaddyPath(service.Caddy.Path) {
		return fmt.Errorf("service %q caddy path must start with /", service.Name)
	}
	for key, value := range service.Caddy.Headers {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("service %q caddy header name cannot be empty", service.Name)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("service %q caddy header %q cannot be empty", service.Name, key)
		}
	}
	for _, snippet := range service.CaddySnippetRefs() {
		if !cfg.hasCaddyImport(snippet) {
			return fmt.Errorf("service %q references undefined caddy snippet %q", service.Name, snippet)
		}
	}
	routeKey := caddyRouteKey(service.Caddy.Domain, caddyRoutePath(service.Caddy.Path))
	if _, exists := routes[routeKey]; exists {
		return fmt.Errorf("service %q duplicates caddy route %s", service.Name, routeKey)
	}
	routes[routeKey] = struct{}{}
	return nil
}

func validateServiceCron(_ Config, service Service, _ map[string]struct{}) error {
	if !service.IsCronProfile() {
		if service.Cron != nil {
			return fmt.Errorf("service %q cron block requires profile cron", service.Name)
		}
		return nil
	}
	if service.Cron == nil {
		return fmt.Errorf("service %q profile cron requires a cron block", service.Name)
	}
	if strings.TrimSpace(service.Cron.Schedule) == "" {
		return fmt.Errorf("service %q cron schedule is required", service.Name)
	}
	if strings.TrimSpace(service.Cron.ServiceAccount) == "" {
		return fmt.Errorf("service %q cron service_account is required", service.Name)
	}
	return nil
}

func (c Config) SortedRequiredServices() []string {
	values := append([]string(nil), c.RequiredServices...)
	sort.Strings(values)
	return values
}

func (c Config) SortedSmokeTests() []SmokeTest {
	tests := append([]SmokeTest(nil), c.SmokeTests...)
	sort.Slice(tests, func(i int, j int) bool {
		return tests[i].Name < tests[j].Name
	})
	return tests
}

func (c Config) ServiceNames() map[string]struct{} {
	names := make(map[string]struct{}, len(c.Services))
	for _, service := range c.Services {
		names[service.Name] = struct{}{}
	}
	return names
}

func (c Config) ServiceByName(name string) (Service, bool) {
	for _, service := range c.Services {
		if service.Name == name {
			return service, true
		}
	}
	return Service{}, false
}

func (s Service) IsServiceProfile() bool {
	return defaultTrimmed(s.Profile, "service") == "service"
}

func (s Service) IsCronProfile() bool {
	return defaultTrimmed(s.Profile, "service") == "cron"
}

func (s Service) IsJobProfile() bool {
	return !s.IsServiceProfile()
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

func (s Service) SortedAnnotationKeys() []string {
	keys := make([]string, 0, len(s.Annotations))
	for key := range s.Annotations {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s Service) SortedSecrets() []Secret {
	secrets := append([]Secret(nil), s.Secrets...)
	sort.Slice(secrets, func(i int, j int) bool {
		return secrets[i].Target < secrets[j].Target
	})
	return secrets
}

func (s Service) SortedCaddyHeaderKeys() []string {
	if s.Caddy == nil {
		return nil
	}
	keys := make([]string, 0, len(s.Caddy.Headers))
	for key := range s.Caddy.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func (s Service) CaddySnippetRefs() []string {
	if s.Caddy == nil {
		return nil
	}
	refs := make([]string, 0, len(s.Caddy.Presets)+len(s.Caddy.Snippets)+len(s.Caddy.Cohorts))
	seen := map[string]struct{}{}
	for _, values := range [][]string{s.Caddy.Presets, s.Caddy.Snippets, s.Caddy.Cohorts} {
		for _, value := range values {
			name := strings.TrimSpace(value)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			seen[name] = struct{}{}
			refs = append(refs, name)
		}
	}
	return refs
}

func (c Config) SortedCaddySnippetNames() []string {
	if c.Caddy == nil {
		return nil
	}
	names := make([]string, 0, len(c.Caddy.Snippets)+len(c.UsedCaddyPresetNames()))
	for name := range c.Caddy.Snippets {
		names = append(names, name)
	}
	names = append(names, c.UsedCaddyPresetNames()...)
	sort.Strings(names)
	return names
}

func (c Config) UsedCaddyPresetNames() []string {
	seen := map[string]struct{}{}
	names := []string{}
	for _, service := range c.Services {
		if service.Caddy == nil {
			continue
		}
		for _, preset := range service.Caddy.Presets {
			name := strings.TrimSpace(preset)
			if name == "" {
				continue
			}
			if _, exists := seen[name]; exists {
				continue
			}
			if _, exists := caddyMiddlewarePresets()[name]; !exists {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func validateProbe(serviceName string, field string, probe *Probe) error {
	if probe == nil {
		return nil
	}
	if probe.Type == "" {
		probe.Type = "grpc"
	}
	if probe.Type != "http" && probe.Type != "grpc" {
		return fmt.Errorf("service %q %s type must be http or grpc", serviceName, field)
	}
	if probe.Type == "http" && strings.TrimSpace(probe.Path) == "" {
		return fmt.Errorf("service %q %s path is required for http probes", serviceName, field)
	}
	if probe.Type == "grpc" && probe.Port <= 0 {
		return fmt.Errorf("service %q %s port must be greater than zero for grpc probes", serviceName, field)
	}
	return nil
}

func defaultImage(region string, projectID string, repository string, serviceName string) string {
	return fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s:latest", region, projectID, repository, serviceName)
}

func defaultTrimmed(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultInt(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultBoolPointer(value *bool, fallback bool) *bool {
	if value != nil {
		return value
	}
	result := fallback
	return &result
}

func defaultConcurrency(protocol string) int {
	if protocol == "grpc" {
		return 20
	}
	return 80
}

func defaultIngress(protocol string) string {
	if protocol == "grpc" {
		// gRPC services are usually private, HTTP/2-backed backends rather than public endpoints.
		return "internal"
	}
	return "all"
}

func ensureServiceMaps(service *Service) {
	if service.Env == nil {
		service.Env = map[string]string{}
	}
	if service.Labels == nil {
		service.Labels = map[string]string{}
	}
	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}
}

func ensureServiceCaddyDefaults(service *Service) {
	if service.Caddy == nil {
		return
	}
	service.Caddy.Path = caddyRoutePath(service.Caddy.Path)
	if service.Caddy.Headers == nil {
		service.Caddy.Headers = map[string]string{}
	}
	if service.Caddy.Presets == nil {
		service.Caddy.Presets = []string{}
	}
	if service.Caddy.Snippets == nil {
		service.Caddy.Snippets = []string{}
	}
	if service.Caddy.Cohorts == nil {
		service.Caddy.Cohorts = []string{}
	}
}

func ensureCronDefaults(service *Service) {
	if service.Cron == nil {
		return
	}
	service.Cron.TimeZone = defaultTrimmed(service.Cron.TimeZone, "UTC")
	service.Cron.JobName = defaultTrimmed(service.Cron.JobName, fmt.Sprintf("%s-schedule", service.Name))
	service.Cron.ServiceAccount = defaultTrimmed(service.Cron.ServiceAccount, service.ServiceAccount)
}

func shouldSetDefaultImage(cfg Config, service Service) bool {
	return strings.TrimSpace(service.Image) == "" &&
		strings.TrimSpace(cfg.ProjectID) != "" &&
		strings.TrimSpace(cfg.Region) != "" &&
		strings.TrimSpace(cfg.ArtifactRegistryRepository) != "" &&
		strings.TrimSpace(service.Name) != ""
}

func (c Config) caddyEnabled() bool {
	return c.Caddy != nil && c.Caddy.Enabled
}

func (c Config) hasCronProfiles() bool {
	for _, service := range c.Services {
		if service.IsCronProfile() {
			return true
		}
	}
	return false
}

func caddyOutputFile(cfg Config) string {
	if cfg.Caddy == nil {
		return "Caddyfile"
	}
	return defaultTrimmed(cfg.Caddy.File, "Caddyfile")
}

func caddyDeploymentPath(cfg Config) string {
	return path.Clean(path.Join(cfg.DeploymentRoot, strings.TrimLeft(caddyOutputFile(cfg), "/")))
}

func caddyRoutePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "/"
	}
	return path
}

func isValidCaddyPath(path string) bool {
	path = caddyRoutePath(path)
	return strings.HasPrefix(path, "/")
}

func caddyRouteKey(domain string, path string) string {
	return fmt.Sprintf("%s|%s", strings.TrimSpace(domain), path)
}

func (c Config) hasCaddyImport(name string) bool {
	if c.Caddy != nil {
		if _, exists := c.Caddy.Snippets[name]; exists {
			return true
		}
	}
	_, exists := caddyMiddlewarePresets()[name]
	return exists
}

func caddyMiddlewarePresets() map[string]string {
	return map[string]string{
		"compression":      `encode zstd gzip`,
		"security-headers": "header X-Frame-Options DENY\nheader X-Content-Type-Options nosniff\nheader Referrer-Policy strict-origin-when-cross-origin",
		"hsts":             `header Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"`,
		"cors-permissive":  "header Access-Control-Allow-Origin *\nheader Access-Control-Allow-Methods \"GET, POST, PUT, PATCH, DELETE, OPTIONS\"\nheader Access-Control-Allow-Headers *",
		"no-store":         `header Cache-Control "no-store, no-cache, must-revalidate"`,
	}
}

func defaultSmokeTestType(test SmokeTest) string {
	if strings.TrimSpace(test.Command) != "" {
		return "command"
	}
	return "http"
}

func defaultSmokeTestName(test SmokeTest) string {
	if strings.TrimSpace(test.Service) != "" {
		return fmt.Sprintf("%s smoke test", test.Service)
	}
	if strings.TrimSpace(test.URL) != "" {
		return fmt.Sprintf("smoke test %s", test.URL)
	}
	return "smoke test"
}

func defaultSmokeTestPath(cfg Config, test SmokeTest) string {
	if strings.TrimSpace(test.Path) != "" {
		return test.Path
	}
	if service, ok := cfg.ServiceByName(test.Service); ok && service.Caddy != nil {
		return caddyRoutePath(service.Caddy.Path)
	}
	return "/"
}

func isValidSmokeTestPath(path string) bool {
	if strings.TrimSpace(path) == "" {
		return true
	}
	return strings.HasPrefix(path, "/")
}

func isValidServiceProfile(profile string) bool {
	switch defaultTrimmed(profile, "service") {
	case "service", "worker", "cron":
		return true
	default:
		return false
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
