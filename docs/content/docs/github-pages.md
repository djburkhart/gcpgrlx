+++
title = 'GitHub Pages'
description = 'How the docs site is published for djburkhart/gcpgrlx.'
weight = 4
+++

## Site location

The documentation site is published to:

```text
https://djburkhart.github.io/gcpgrlx/
```

## Source layout

The Hugo site lives under:

```text
docs/
```

The custom theme lives under:

```text
docs/themes/gcpgrlx-modern/
```

## Deployment workflow

The GitHub Actions workflow in `.github/workflows/deploy-pages.yml`:

1. checks out the repository
2. configures GitHub Pages
3. installs Hugo
4. builds the site from `docs/`
5. uploads the generated `docs/public` output
6. deploys the built site to GitHub Pages

## Local build

If Hugo is installed locally, build the site with:

```powershell
hugo --source .\docs
```
