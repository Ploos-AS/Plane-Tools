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

### M2.2 — aircraft dataset ingestion

The normal Plane Tools binary can normalize larger CSV sources into the canonical
local dataset:

```sh
plane-tools import-aircraft --input source.csv --output /data/aircraft.csv
```

The importer understands common aliases such as `hex`, `reg`, `type`, `maker`
and `owner`, validates rows, removes duplicate ICAO24/registration entries,
sorts output deterministically and replaces the destination atomically. Import
statistics report total rows, imported rows, duplicates and invalid rows.

### M2.3 — tar1090/readsb adapter

Plane Tools can also ingest the semicolon-delimited aircraft database format
commonly used by tar1090/readsb, including gzip-compressed `aircraft.csv.gz`
files:

```sh
plane-tools import-aircraft \
  --input aircraft.csv.gz \
  --format tar1090 \
  --output /data/aircraft.csv
```

`--format auto` is the default and treats `.gz` input as tar1090 format. Plane
Tools only provides the adapter; it does not bundle or automatically download
third-party aircraft data.

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
