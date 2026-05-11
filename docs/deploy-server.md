# RentCar Server Deploy

This deployment runs the full RentCar stack with Docker Compose:

- React frontend served by Nginx
- Go API
- PostgreSQL
- Redis
- MinIO
- one-shot migration and seed jobs

## 1. Server Requirements

Ubuntu 22.04/24.04 is recommended.

```bash
sudo apt update
sudo apt install -y git ca-certificates curl ufw
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
```

Log out and back in after adding the user to the `docker` group.

## 2. Copy Project

```bash
mkdir -p /opt/rentcar
cd /opt/rentcar
```

Copy both folders to the server so the structure looks like:

```text
/opt/rentcar/Backend
/opt/rentcar/frontend
```

## 3. Configure Environment

```bash
cd /opt/rentcar/Backend
cp .env.production.example .env
nano .env
```

Change these values before starting:

```env
POSTGRES_PASSWORD=strong-db-password
JWT_SECRET=at-least-32-random-characters
SUPER_ADMIN_EMAIL=admin@example.com
SUPER_ADMIN_PASSWORD=strong-admin-password
IMAGE_PUBLIC_URL=https://your-domain.com/api/v1/uploads/images
CORS_ALLOWED_ORIGINS=https://your-domain.com
MINIO_ROOT_PASSWORD=strong-minio-password
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-gmail-app-password
SMTP_FROM=your-email@gmail.com
```

For IP-only testing, use:

```env
IMAGE_PUBLIC_URL=http://SERVER_IP/api/v1/uploads/images
CORS_ALLOWED_ORIGINS=http://SERVER_IP
```

## 4. Start Production Stack

```bash
cd /opt/rentcar/Backend
docker compose -f docker-compose.prod.yml up -d --build
```

Check status:

```bash
docker compose -f docker-compose.prod.yml ps
curl http://127.0.0.1/health
```

## 5. Firewall

```bash
sudo ufw allow OpenSSH
sudo ufw allow 80/tcp
sudo ufw enable
```

If you add HTTPS on the server, also run:

```bash
sudo ufw allow 443/tcp
```

## 6. Logs

```bash
docker compose -f docker-compose.prod.yml logs -f app
docker compose -f docker-compose.prod.yml logs -f frontend
```

## 7. Update Deployment

After changing code:

```bash
cd /opt/rentcar/Backend
docker compose -f docker-compose.prod.yml up -d --build
```

## 8. Backup Database

```bash
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U car_rental car_rental > rentcar-backup.sql
```
