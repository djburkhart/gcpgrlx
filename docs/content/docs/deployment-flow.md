+++
title = 'Deployment flow'
description = 'How gcpgrlx, grlx, and Google Cloud fit together.'
weight = 3
+++

## The role of gcpgrlx

`gcpgrlx` is not a Cloud Run deployer by itself. It is a renderer that turns service intent into a portable `grlx` recipe.

That means you can:

1. describe services once in YAML
2. generate a deterministic `.grlx` recipe
3. distribute and execute it through `grlx`
4. keep Google Cloud operations inside the machines that already own `gcloud` access

## Flow overview

1. A team defines a deployment spec with service runtime, rollout, networking, secret, probe, Caddy, and job settings.
2. `gcpgrlx` applies defaults and validates the spec.
3. The CLI renders a `deploy.grlx` recipe, and optionally a `Caddyfile`, with `cmd.run` and `file.directory` states.
4. `grlx` cooks that recipe on the target sprout or cohort.
5. The sprout executes the generated `gcloud` commands against Google Cloud Run, Cloud Scheduler, Caddy, and related APIs.

## Architecture visual

![Workflow overview](https://raw.githubusercontent.com/djburkhart/gcpgrlx/main/overview.png)

## Example microservices visual

![Rendered example architecture](https://raw.githubusercontent.com/djburkhart/gcpgrlx/main/microservices-architecture-diagram.png)

## Why this split is useful

- `grlx` stays the automation and distribution layer
- `gcpgrlx` stays focused on Google Cloud deployment intent
- Cloud Run operational flags remain explicit and reviewable
- generated recipes are inspectable before execution

## What v0.3.0 adds to the flow

- Caddy routing can be generated, patched with live Cloud Run URLs, applied on the target, validated, and reloaded in the same rollout.
- Smoke tests can run at the end of the recipe so you can verify the public edge or a direct service URL immediately after deploy.
- Worker and cron profiles let the same config model cover HTTP or gRPC services, queue consumers, and scheduled jobs.
