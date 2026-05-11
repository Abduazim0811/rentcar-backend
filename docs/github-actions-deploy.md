# RentCar GitHub Actions Deploy

RentCar has two repositories:

- `rentcar-backend`
- `rentcar-frontend`

The backend workflow deploys the full Docker Compose stack to the server. The frontend workflow only rebuilds the frontend image and restarts the frontend service, so run the backend deploy first at least once.

## Server Preparation

Install Docker on the server:

```bash
apt update
apt install -y ca-certificates curl ufw
curl -fsSL https://get.docker.com | sh
ufw allow 30030/tcp
ufw allow 80/tcp
ufw --force enable
```

Create a deploy SSH key on your laptop or locally:

```bash
ssh-keygen -t ed25519 -C "rentcar-deploy" -f ~/.ssh/rentcar_deploy
```

Add the public key to the server:

```bash
cat ~/.ssh/rentcar_deploy.pub | ssh -p 30030 root@176.96.243.232 "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys && chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys"
```

Put the private key `~/.ssh/rentcar_deploy` into GitHub as `SSH_PRIVATE_KEY`.

## Required GitHub Secrets

Add these secrets to both repositories:

```text
GHCR_TOKEN
SSH_HOST=176.96.243.232
SSH_USER=root
SSH_PORT=30030
SSH_PRIVATE_KEY
```

`GHCR_TOKEN` must be a GitHub Personal Access Token with package read/write permissions.

Add these secrets to the backend repository:

```text
HTTP_PORT=80
POSTGRES_DB=car_rental
POSTGRES_USER=car_rental
POSTGRES_PASSWORD=strong-db-password
JWT_SECRET=at-least-32-random-characters
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=720h
SUPER_ADMIN_NAME=Super Admin
SUPER_ADMIN_EMAIL=admin@rentcar.local
SUPER_ADMIN_PASSWORD=strong-admin-password
IMAGE_PUBLIC_URL=http://176.96.243.232/api/v1/uploads/images
CORS_ALLOWED_ORIGINS=http://176.96.243.232
HTTP_READ_TIMEOUT=5s
HTTP_WRITE_TIMEOUT=10s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=10s
HTTP_MAX_BODY_BYTES=10485760
RATE_LIMIT_MAX_REQUESTS=120
RATE_LIMIT_WINDOW=1m
DB_MAX_CONNS=20
DB_MIN_CONNS=2
MINIO_ROOT_USER=rentcar-minio
MINIO_ROOT_PASSWORD=strong-minio-password
MINIO_BUCKET=rentcar-images
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-gmail-app-password
SMTP_FROM=your-email@gmail.com
SMTP_FROM_NAME=RentCar
SMTP_USE_TLS=false
EMAIL_VERIFICATION_TTL=10m
```

## Deploy Order

1. Push frontend to `main` so `ghcr.io/abduazim0811/rentcar-frontend:latest` exists.
2. Push backend to `main`; backend workflow deploys the whole stack.

After the first backend deploy, future frontend pushes will update only the frontend container.

## Check Server

```bash
ssh -p 30030 root@176.96.243.232
docker compose --env-file ~/rentcar/.env -f ~/rentcar/docker-compose.yml ps
curl http://127.0.0.1/health
```
