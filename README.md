# Plane Tools

Self-hosted, local-first tools for plane spotting and ADS-B workflows.

## Current status

### M0 — foundation

- Go backend with embedded web UI
- `GET /healthz` health endpoint
- minimal multi-stage OCI image
- non-root runtime user
- persistent `/data` volume
- Docker Compose example
- amd64/arm64-friendly source layout
- MIT licensed

### M1 — local aviation utilities

Plane Tools includes useful functionality with no API keys or external datasets:

- great-circle distance in nautical miles
- initial true bearing
- nautical miles ↔ kilometres
- knots ↔ kilometres per hour
- feet ↔ metres
- JSON API and browser UI for the utilities

### M2 / M2.1 — local aircraft lookup

Aircraft lookup supports ICAO24 and registration through `GET /api/v1/aircraft`.
Plane Tools loads `/data/aircraft.csv` at startup and builds in-memory indexes for
fast lookup. If the file is absent, the small built-in seed dataset is used.
Malformed datasets fail explicitly at startup.

Copy `examples/aircraft.csv` to `/data/aircraft.csv` to try the import contract,
or override the path with `PLANE_TOOLS_AIRCRAFT_CSV`.

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

## API examples

```sh
curl 'http://localhost:8080/api/v1/distance?lat1=58.2042&lon1=8.0854&lat2=59.9111&lon2=10.7528'
curl 'http://localhost:8080/api/v1/convert?value=100&from=kt&to=kph'
curl 'http://localhost:8080/api/v1/aircraft?icao24=4787a2'
curl 'http://localhost:8080/api/v1/aircraft?registration=LN-NGM'
curl http://localhost:8080/healthz
```

## Direction

Plane Tools is intended to grow into a toolbox for:

- richer aircraft/type/operator datasets
- airport lookup and runway data
- local spotting logbook
- optional readsb/dump1090 receiver integrations
- optional provider integrations

Core functionality should remain useful without API keys, and private spotting
data should remain local by default.

## License

MIT. See [LICENSE](LICENSE).
