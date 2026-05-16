# gcpgrlx

`gcpgrlx` is a GCP-specific companion module for [`grlx`](https://github.com/gogrlx/grlx) that renders deployment recipes for shipping containerized **gRPC and HTTP microservices** to **Google Cloud Run**.

It is designed for teams that want to keep `grlx` as the automation layer while using Google Cloud as the runtime target.

## What it does

- validates a microservice deployment spec
- resolves default Artifact Registry image URLs
- renders a `.grlx` recipe with `cmd.run` states for:
  - enabling required GCP APIs
  - ensuring an Artifact Registry repository exists
  - deploying each service to Cloud Run
- applies gRPC-friendly defaults such as internal ingress, HTTP/2, and tighter concurrency
- supports advanced deployment controls like secrets, probes, rollout traffic, Cloud SQL, and revision settings
- exposes the renderer as both a Go package and a small CLI

## Why Cloud Run

Cloud Run is a strong default for microservices because it keeps the deployment contract small:

- each service is a container
- ingress, scaling, CPU, memory, and auth are first-class flags
- it maps cleanly onto generated `grlx` `cmd.run` states

For gRPC services, `gcpgrlx` can emit Cloud Run deploy commands with:

- `--use-http2`
- internal-only ingress by default
- disabled unauthenticated access by default
- service-level concurrency and optional VPC connector settings
- optional startup and liveness probes via `gcloud beta run deploy`

## Install

```powershell
go install .\cmd\gcpgrlx
```

## Example spec

See `examples\microservices.yaml`.

## Render a recipe

```powershell
go run .\cmd\gcpgrlx render -f .\examples\microservices.yaml -out .\dist
```

That writes:

- `dist\deploy.grlx`

## Render directly to stdout

```powershell
go run .\cmd\gcpgrlx render-stdout -f .\examples\microservices.yaml
```

## Preview a deployment plan

```powershell
go run .\cmd\gcpgrlx plan -f .\examples\microservices.yaml
```

## Validate a config

```powershell
go run .\cmd\gcpgrlx validate -f .\examples\microservices.yaml
```

## Initialize a starter config

```powershell
go run .\cmd\gcpgrlx init -out .\microservices.yaml
```

To print the starter config instead of writing a file:

```powershell
go run .\cmd\gcpgrlx init --stdout
```

## Example output shape

The generated recipe includes states like:

- verify `gcloud` is installed
- enable `run.googleapis.com` and `artifactregistry.googleapis.com`
- create the Artifact Registry repository if missing
- deploy each microservice with `gcloud run deploy`
- configure gRPC services with end-to-end HTTP/2 support

## Go package

```go
package main

import (
	"fmt"
	"os"

	"github.com/djburkhart/gcpgrlx"
)

func main() {
	raw, err := os.ReadFile("microservices.yaml")
	if err != nil {
		panic(err)
	}

	cfg, err := gcpgrlx.ParseConfig(raw)
	if err != nil {
		panic(err)
	}

	recipe, err := gcpgrlx.RenderRecipe(cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println(recipe)
}
```

## Spec model

Top-level fields:

- `project_id`
- `region`
- `artifact_registry_repository`
- `deployment_root` (optional)
- `required_services` (optional)
- `services`

Per-service fields:

- `name`
- `image` (optional; default Artifact Registry URL is generated when omitted)
- `service_account` (optional)
- `protocol` (`http` or `grpc`)
- `command`
- `args`
- `port`
- `cpu`
- `memory`
- `concurrency`
- `min_instances`
- `max_instances`
- `timeout`
- `revision_suffix`
- `traffic_percent`
- `no_traffic`
- `cpu_throttling`
- `startup_cpu_boost`
- `execution_environment`
- `use_http2`
- `allow_unauthenticated`
- `ingress`
- `vpc_connector`
- `vpc_egress`
- `cloud_sql_instances`
- `secrets`
- `startup_probe`
- `liveness_probe`
- `env`
- `labels`
- `annotations`

### gRPC defaults

If `protocol: grpc` is set for a service, `gcpgrlx` automatically defaults:

- `ingress: internal`
- `allow_unauthenticated: false`
- `use_http2: true`
- `concurrency: 20`

### Advanced deployment fields

- `secrets` render to `--update-secrets`
- `cloud_sql_instances` render to `--add-cloudsql-instances`
- `traffic_percent` adds a follow-up `gcloud run services update-traffic` step
- `no_traffic: true` keeps a new revision dark after deploy
- `command` and `args` override container startup behavior
- `cpu_throttling`, `startup_cpu_boost`, and `execution_environment` control runtime behavior
- `startup_probe` and `liveness_probe` switch recipe generation to `gcloud beta run deploy`

## Validation

```powershell
go test .\...
```

## License

MIT
