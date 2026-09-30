# Link Shortener

A small Go link shortener with one hostname for the short links and the link-creation website.

- `wcal.xyz/<code>` redirects a base62 code to its stored URL.
- `wcal.xyz/` provides the creation form; `wcal.xyz/stats/<code>` shows link statistics.
- MySQL stores links and view counts.
- `POST /api/links` creates links and returns the short URL as JSON without a page refresh.

## Requirements

- Go 1.26.7 or newer
- MySQL 8.4 (the legacy MySQL 5.7 dump is compatible)

## Environment variables

| Variable | Description | Example |
|---|---|---|
| `MYSQL_URI` | Go MySQL driver DSN | `user:password@tcp(mysql:3306)/link_shortener?charset=utf8mb4&parseTime=true` |
| `SHORT_URL` | Public hostname for the website and generated short URLs (no scheme or path) | `wcal.xyz` |
| `PORT` | HTTP port; optional | `5000` |

The creation endpoint is public. All routes are available on any hostname pointing to the app; generated short URLs use `SHORT_URL`. `WEBSITE_URL` is no longer used and can be removed from existing deployments. Existing short links and database IDs are unchanged.

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

1. Create a persistent MySQL 8.4 service on an internal network.
2. Restore the legacy MySQL dump into the `link_shortener` database while preserving IDs and `AUTO_INCREMENT`.
3. Configure the environment variables above using Dokploy secrets.
4. Deploy this repository using its Dockerfile and internal port 5000.
5. Attach `wcal.xyz` to the application. The old website hostname is optional and can continue serving the same app.
6. Verify `/healthz`, a representative legacy redirect, view increments, and creation of the next ID before changing DNS.

The short-code mapping depends on the exact legacy base62 alphabet. Compatibility tests cover IDs through the current production maximum.
