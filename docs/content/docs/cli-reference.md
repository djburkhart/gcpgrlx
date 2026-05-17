+++
title = 'CLI reference'
description = 'Command-level reference for the gcpgrlx CLI.'
weight = 2
+++

## Commands

### `render`

Render a `deploy.grlx` file into an output directory.

```powershell
gcpgrlx render -f .\examples\microservices.yaml -out .\dist
```

### `render-stdout`

Stream the rendered recipe directly to standard output.

```powershell
gcpgrlx render-stdout -f .\examples\microservices.yaml
```

### `render-caddy`

Render the generated Caddyfile when top-level `caddy.enabled: true` is set.

```powershell
gcpgrlx render-caddy -f .\examples\microservices.yaml -out .\dist
gcpgrlx render-caddy -f .\examples\microservices.yaml --stdout
```

### `plan`

Print the high-level deployment steps and generated Google Cloud commands.

```powershell
gcpgrlx plan -f .\examples\microservices.yaml
```

### `validate`

Validate a config file after defaults are applied.

```powershell
gcpgrlx validate -f .\examples\microservices.yaml
```

### `init`

Write a starter config or print one to stdout.

```powershell
gcpgrlx init -out .\microservices.yaml
gcpgrlx init --stdout
```

## Key config fields

Top-level fields:

- `project_id`
- `region`
- `artifact_registry_repository`
- `deployment_root`
- `caddy`
- `required_services`
- `smoke_tests`
- `services`

Common service fields:

- `name`
- `image`
- `profile`
- `protocol`
- `command`
- `args`
- `service_account`
- `cpu`
- `memory`
- `concurrency`
- `min_instances`
- `max_instances`
- `timeout`
- `revision_suffix`
- `traffic_percent`
- `no_traffic`
- `secrets`
- `startup_probe`
- `liveness_probe`
- `cloud_sql_instances`
- `vpc_connector`
- `vpc_egress`
- `annotations`
- `caddy`
- `cron`

Top-level `caddy` fields:

- `enabled`
- `file`
- `template`
- `snippets`

Top-level `smoke_tests` fields:

- `name`
- `service`
- `type`
- `url`
- `path`
- `method`
- `headers`
- `expected_status`
- `body_contains`
- `command`
- `timeout`
- `skip_tls_verify`

Per-service `caddy` fields:

- `domain`
- `path`
- `upstream`
- `headers`
- `presets`
- `snippets`
- `cohorts`

Per-service `cron` fields:

- `schedule`
- `time_zone`
- `service_account`
- `job_name`

## gRPC defaults

When `protocol: grpc` is set, `gcpgrlx` defaults to:

- `ingress: internal`
- `allow_unauthenticated: false`
- `use_http2: true`
- `concurrency: 20`

## Profile notes

- `profile: service` renders Cloud Run service deploys
- `profile: worker` renders Cloud Run Job deploys
- `profile: cron` renders Cloud Run Job deploys plus a Cloud Scheduler trigger

## Built-in Caddy presets

- `compression`
- `security-headers`
- `hsts`
- `cors-permissive`
- `no-store`
