+++
title = 'Getting started'
description = 'Install gcpgrlx, render your first recipe, and cook it with grlx.'
weight = 1
+++

## 1. Install the CLI

```powershell
go install github.com/djburkhart/gcpgrlx/cmd/gcpgrlx@v0.3.0
```

## 2. Start from the sample config

```powershell
gcpgrlx init -out .\microservices.yaml
```

You can also start from the example in `examples\microservices.yaml`.

## 3. Validate before rendering

```powershell
gcpgrlx validate -f .\microservices.yaml
```

## 4. Render the recipe

```powershell
gcpgrlx render -f .\microservices.yaml -out .\dist
```

That produces:

```text
dist\deploy.grlx
```

## 5. Render the Caddyfile when edge routing is enabled

```powershell
gcpgrlx render-caddy -f .\microservices.yaml -out .\dist
```

That also produces:

```text
dist\caddy\Caddyfile
```

## 6. Preview the plan

```powershell
gcpgrlx plan -f .\microservices.yaml
```

## 7. Run the recipe with grlx

Dry run:

```powershell
grlx cook .\dist\deploy.grlx -T gcp-runner --test
```

Apply:

```powershell
grlx cook .\dist\deploy.grlx -T gcp-runner
```

Target a cohort:

```powershell
grlx cook .\dist\deploy.grlx -C platform
```

## What gcpgrlx handles

- default Artifact Registry image resolution
- Cloud Run deployment command generation
- gRPC-aware defaults such as HTTP/2 and internal ingress
- advanced rollout controls like probes, secrets, Cloud SQL, and traffic updates
- Caddy rendering plus install, validate, reload, and live upstream patching steps
- smoke tests that run after deploy
- worker and cron profiles for background jobs
