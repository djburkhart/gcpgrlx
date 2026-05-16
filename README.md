# gcpgrlx

`gcpgrlx` is a GCP-specific companion module for [`grlx`](https://github.com/gogrlx/grlx) that renders deployment recipes for shipping containerized microservices to **Google Cloud Run**.

It is designed for teams that want to keep `grlx` as the automation layer while using Google Cloud as the runtime target.

## What it does

- validates a microservice deployment spec
- resolves default Artifact Registry image URLs
- renders a `.grlx` recipe with `cmd.run` states for:
  - enabling required GCP APIs
  - ensuring an Artifact Registry repository exists
  - deploying each service to Cloud Run
- exposes the renderer as both a Go package and a small CLI

## Why Cloud Run

Cloud Run is a strong default for microservices because it keeps the deployment contract small:

- each service is a container
- ingress, scaling, CPU, memory, and auth are first-class flags
- it maps cleanly onto generated `grlx` `cmd.run` states

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

## Example output shape

The generated recipe includes states like:

- verify `gcloud` is installed
- enable `run.googleapis.com` and `artifactregistry.googleapis.com`
- create the Artifact Registry repository if missing
- deploy each microservice with `gcloud run deploy`

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
- `port`
- `cpu`
- `memory`
- `min_instances`
- `max_instances`
- `timeout`
- `allow_unauthenticated`
- `ingress`
- `env`
- `labels`

## Validation

```powershell
go test .\...
```

## License

MIT
