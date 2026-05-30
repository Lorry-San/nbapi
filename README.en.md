# NBAPI

NBAPI is a customized distribution based on [New API](https://github.com/QuantumNous/new-api). It keeps the OpenAI, Claude, Gemini, and other multi-protocol gateway capabilities, and adds operator-focused features for self-hosted deployments.

## Highlights

- Branding defaults to `NBAPI` while keeping the upstream attribution as “Based on New API”.
- User IP logging is forced on and cannot be disabled by normal users.
- Super admins can promote other users to super admin.
- Super admins can one-click sign in as other users.
- Classic usage logs can be exported as CSV.
- OpenAI-compatible channels can use a custom API version path when an upstream does not use `/v1`.
- Payment compliance confirmation blocking was removed from redemption code, subscription, invite rebate, and payment settings flows.
- GHCR images and ready-to-deploy Docker Compose files are provided.

## Image

```text
ghcr.io/lorry-san/nbapi:beta-5.30-1
ghcr.io/lorry-san/nbapi:custom
```

The `main` branch build publishes:

- `ghcr.io/lorry-san/nbapi:custom`
- `ghcr.io/lorry-san/nbapi:beta-5.30-1`
- `ghcr.io/lorry-san/nbapi:custom-<sha>`

## Quick Start

```bash
cp .env.docker.example .env.docker
```

Edit `.env.docker` and change at least:

```text
POSTGRES_PASSWORD=...
REDIS_PASSWORD=...
SESSION_SECRET=...
```

Start the stack:

```bash
docker compose --env-file .env.docker -f docker-compose.github.yml pull nbapi
docker compose --env-file .env.docker -f docker-compose.github.yml up -d
```

Default URL:

```text
http://localhost:3000
```

## Updating an Existing Deployment

If you already run New API or NBAPI with Docker Compose, back up your database first, then change only the application service image to:

```text
ghcr.io/lorry-san/nbapi:beta-5.30-1
```

Update only the application container:

```bash
docker compose pull nbapi
docker compose up -d --no-deps nbapi
```

Do not run `docker compose down -v`, because it deletes database volumes.

## Local Source Build

```bash
docker compose --env-file .env.docker -f docker-compose.local.yml up -d --build
```

## Docs

- GHCR deployment guide: [docs/local-docker-deploy.zh-CN.md](docs/local-docker-deploy.zh-CN.md)
- Upstream project: [QuantumNous/new-api](https://github.com/QuantumNous/new-api)

## License

NBAPI is derived from New API and follows the upstream AGPL v3.0 license requirements.
