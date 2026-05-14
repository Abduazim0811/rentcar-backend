# RentCar Backend

Monolithic Go backend for a car rental system. It uses Gin, PostgreSQL, Redis service wiring, MinIO image storage, JWT authentication, clean architecture layers, migrations, structured logging, rate limiting and OpenAPI docs.

## Requirements

- Go 1.25+
- Docker and Docker Compose
- PostgreSQL 16 if running without Docker

## Quick Start

```bash
cp .env.example .env
make docker-up
```

The API starts at `http://localhost:8080`.

Useful endpoints:

- `GET /health/live`
- `GET /health/ready`
- `GET /docs`
- `GET /openapi.yaml`

## Local Development

```bash
make migrate-up
make seed
make dev
```

Default API base URL:

```text
http://localhost:8080/api/v1
```

## Environment

Production must set a strong `JWT_SECRET` with at least 32 characters. Important variables:

- `DATABASE_URL`
- `JWT_SECRET`
- `ACCESS_TOKEN_TTL`
- `REFRESH_TOKEN_TTL`
- `RATE_LIMIT_MAX_REQUESTS`
- `RATE_LIMIT_WINDOW`
- `CORS_ALLOWED_ORIGINS`
- `TRUSTED_PROXIES` (set to reverse proxy CIDRs/IPs when using HTTPS proxy headers)
- `MINIO_ENDPOINT`
- `IMAGE_PUBLIC_URL`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`
- `SMTP_USE_TLS` (`true` for implicit TLS/465, `false` for STARTTLS/587)
- `EMAIL_VERIFICATION_TTL`

## Checks

```bash
make test
make build
docker compose config
```

## Architecture

- `handler`: HTTP transport and request validation
- `service`: business logic
- `repository`: PostgreSQL persistence
- `middleware`: auth, admin guard, logging, recovery, CORS, rate limit, security headers
- `models`: domain entities
- `migrations`: database schema
