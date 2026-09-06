# Plane Tools

Self-hosted, local-first tools for plane spotting and ADS-B workflows.

## M0 foundation

M0 establishes the runnable project baseline:

- Go backend with embedded web UI
- `GET /healthz` health endpoint
- minimal multi-stage OCI image
- non-root runtime user
- persistent `/data` volume reserved for future SQLite/logbook data
- Docker Compose example
- amd64/arm64-friendly source layout
- MIT licensed

The application does not require an SDR or external API provider at M0.

## Run with Go

```sh
go run ./cmd/plane-tools
```

Open <http://localhost:8080>.

## Run with Docker/Podman Compose

```sh
docker compose up --build
```

Then open <http://localhost:8080>.

## Health check

```sh
curl http://localhost:8080/healthz
```

## Direction

Plane Tools is intended to grow into a toolbox for:

- ICAO24 / aircraft registration lookup
- aircraft type and operator data
- airport and coordinate utilities
- distance, bearing and aviation unit conversion
- local spotting logbook
- optional readsb/dump1090 receiver integrations
- optional provider integrations

Core functionality should remain useful without API keys, and private spotting
data should remain local by default.

## License

MIT. See [LICENSE](LICENSE).
