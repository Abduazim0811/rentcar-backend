# HTTPS for RentCar

Use this after DNS works:

```bash
nslookup rent-uz.uz 8.8.8.8
```

It must return:

```text
176.96.243.232
```

## 1. Keep RentCar on an Internal HTTP Port

On the server:

```bash
cd ~/Backend
nano .env
```

Use:

```env
HTTP_PORT=8088
IMAGE_PUBLIC_URL=https://rent-uz.uz/api/v1/uploads/images
CORS_ALLOWED_ORIGINS=https://rent-uz.uz,https://www.rent-uz.uz
TRUSTED_PROXIES=127.0.0.1
```

Restart RentCar:

```bash
docker compose --env-file .env -f docker-compose.server.yml up -d --build
```

## 2. Install Caddy

```bash
apt update
apt install -y debian-keyring debian-archive-keyring apt-transport-https curl
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' > /etc/apt/sources.list.d/caddy-stable.list
apt update
apt install -y caddy
```

## 3. Free Ports 80 and 443

Caddy must own ports `80` and `443`.

```bash
docker ps --format 'table {{.Names}}\t{{.Ports}}' | grep -E ':80|:443' || true
```

If another container uses port `80`, either stop it or move it behind the same reverse proxy.

## 4. Configure Caddy

```bash
cat > /etc/caddy/Caddyfile <<'EOF'
rent-uz.uz, www.rent-uz.uz {
    reverse_proxy 127.0.0.1:8088
}
EOF
```

Reload:

```bash
caddy validate --config /etc/caddy/Caddyfile
systemctl reload caddy
```

## 5. Firewall

```bash
ufw allow 80/tcp
ufw allow 443/tcp
```

## 6. Check

```bash
curl -I https://rent-uz.uz
curl -I https://rent-uz.uz/health
```

The site should open at:

```text
https://rent-uz.uz
```
