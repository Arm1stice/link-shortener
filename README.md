# Link Shortener

A small Go link shortener with separate hostnames for the short links and the link-creation website.

- `wcal.xyz/<code>` redirects a base62 code to its stored URL.
- `links.wcalandro.com` provides the creation form and link statistics.
- MySQL stores links and view counts.
- Redis stores short-lived flash sessions.

## Requirements

- Go 1.26.7 or newer
- MySQL 8.4 (the legacy MySQL 5.7 dump is compatible)
- Redis 7

## Environment variables

| Variable | Description | Example |
|---|---|---|
| `MYSQL_URI` | Go MySQL driver DSN | `user:password@tcp(mysql:3306)/link_shortener?charset=utf8mb4&parseTime=true` |
| `REDIS_HOST` | Redis host and port | `redis:6379` |
| `REDIS_PASSWORD` | Redis password | Set a generated secret |
| `WEBSITE_URL` | Hostname for the creation website | `links.wcalandro.com` |
| `SHORT_URL` | Hostname used for short redirects | `wcal.xyz` |
| `SESSION_SECRET` | Session authentication secret | Set a generated secret of at least 32 bytes |
| `PORT` | HTTP port; optional | `5000` |

The creation endpoint is public. Protect `links.wcalandro.com` at the reverse proxy if link creation should be private. Redirects on `wcal.xyz` must remain public.

## Development

```bash
go mod download
go test -race ./...
go vet ./...
go run .
```

The service exposes `GET /healthz` on every hostname and returns `200 OK` once the process is serving requests.

## Container

The multi-stage Dockerfile:

- pins the Go and Alpine base-image digests;
- embeds the HTML template with `go:embed`;
- builds static amd64 and arm64 binaries;
- runs as the non-root `app` user (UID 10001);
- exposes port 5000;
- includes a container health check.

Build the native image:

```bash
docker build -t link-shortener .
```

Verify both deployment architectures:

```bash
docker buildx build --platform linux/amd64,linux/arm64 .
```

## Dokploy deployment

1. Create persistent MySQL 8.4 and Redis 7 services on an internal network.
2. Restore the legacy MySQL dump into the `link_shortener` database while preserving IDs and `AUTO_INCREMENT`.
3. Configure the environment variables above using Dokploy secrets.
4. Deploy this repository using its Dockerfile and internal port 5000.
5. Attach both `wcal.xyz` and `links.wcalandro.com` to the application.
6. Verify `/healthz`, a representative legacy redirect, view increments, and creation of the next ID before changing DNS.

The short-code mapping depends on the exact legacy base62 alphabet. Compatibility tests cover IDs through the current production maximum.
