# Changelog

All notable changes to this project will be documented in this file.

The format is based on **Keep a Changelog**, with the current working tree tracked under **Unreleased** until those changes are committed and tagged.

## [v0.3.0] - 2026-05-17

### Added
- Added advanced Caddy support with reusable snippets, per-service cohorts, inline `text/template` rendering, and built-in middleware presets.
- Added `render-caddy` so the CLI can render the generated Caddyfile directly.
- Added deploy-time Caddy install, apply, validate, reload, and live Cloud Run upstream patching steps to generated recipes.
- Added configurable `smoke_tests` for HTTP and command-based end-to-end verification after rollout.
- Added `worker` and `cron` service profiles with Cloud Run Jobs and Cloud Scheduler rendering.
- Added updated sample and example configs covering Caddy routing, smoke tests, worker jobs, cron jobs, and middleware presets.

### Changed
- Expanded config validation to cover Caddy imports, preset resolution, smoke tests, and worker or cron profile constraints.
- Expanded rendered deployment plans and recipes to include Caddy lifecycle work, post-deploy verification, and non-service workloads.
- Refreshed the docs site so the getting started flow, CLI reference, homepage examples, and deployment flow match the current feature set.

## [v0.2.1] - 2026-05-16
- Update `.gitignore`.
- Add release automation.
- split setDefaults() into top-level/service-specific default helpers
- split Validate() into focused validation helpers for required fields, names, scaling, runtime, secrets, and probes
- split renderDeployCommand() into base command, flag appenders, toggle helpers, and map/secret rendering helpers

## [v0.2.0] - 2026-05-16

### Added
- Added explicit CLI commands for `validate`, `init`, `plan`, and `render-stdout`.
- Added sample configuration helpers with a reusable generic gRPC-focused starter config.
- Added advanced Cloud Run deployment fields for:
  - startup command and args
  - revision suffixes and rollout traffic control
  - CPU throttling and startup CPU boost
  - execution environment selection
  - Cloud SQL instance attachment
  - Secret Manager bindings
  - startup and liveness probes
  - service annotations
- Added dedicated test files for config validation, recipe output, and CLI command behavior.
- Added `.gitignore` coverage for generated `dist` output.
- Added `codeql.yml` quality testing.

### Changed
- Expanded recipe rendering so deployment plans can include:
  - `gcloud beta run deploy` when probes are configured
  - follow-up `gcloud run services update-traffic` steps for rollout control
  - additional Cloud Run flags for advanced runtime, networking, and secret settings
- Expanded the example deployment spec to show realistic gRPC microservice settings, including traffic control, probes, Cloud SQL, and secrets.
- Updated the CLI and library flow so config validation, sample generation, plan rendering, recipe rendering, and file output are all available as first-class workflows.
- Added targeted inline comments to the config and rendering flow to make the execution path easier to follow.

### Documentation
- Updated the README with the new CLI commands and examples for:
  - `validate`
  - `init`
  - `plan`
  - `render-stdout`
- Documented the newer advanced deployment fields and gRPC-specific behavior in the README.

## [v0.1.0] - 2026-05-16

### Added
- Added gRPC-aware Cloud Run deployment support.
- Added gRPC deployment defaults for internal ingress, HTTP/2, and tighter concurrency.
- Added support for VPC connector, VPC egress, and gRPC-oriented example services.

### Changed
- Updated the generic example config and README to focus on gRPC deployment to Google Cloud Run.
