# Container Ledger — Shipping Container Tracking on Hyperledger Fabric

A blockchain-based shipping-container tracking application built with **Hyperledger Fabric, Go, and Svelte**. The project includes smart-contract logic for tracking containers through their lifecycle, a Go application layer, and a web-based graphical interface.

Each container is a ledger record with its cargo, weights, seal, destination, customs status and (for reefers) temperature. Every operation appends an immutable entry to that container's **journey**, so the full chain of custody is auditable.

## Features

- **Register containers** with an ID, type, owner, tare weight, and destination.
- **Read a container** by ID, or list all containers on the ledger.
- **Update container details.**
- **Transfer ownership** of a container (recorded on the journey).
- **Delete (de-register) a container.**
- **Report lifecycle events** (GateIn, Loaded, Discharged, GateOut, InTransit, CustomsHold/Release, …) that also drive status transitions.
- **Report damage**, with type and description.
- **Update customs status** (Clear / Hold / Release).
- **Set reefer temperature** setpoints.
- **Query/filter containers** by status, type, owner, or location.
- **View journey history** for any container.
- **Track live GPS positions** per container (latest fix embedded, full track queryable), with **auto-transition to `InTransit`** on the first fix of a loaded container.
- **Ingest GPS/nav files** (NMEA, CSV, JSON, GPX) off-chain and push updates to the ledger, with a dry-run preview.
- **Track map** (Leaflet + OpenStreetMap) plotting each container's GPS route, with antimeridian handling.
- **Web interface** for dashboards, filtering, registering, transfers, events, damage, customs, GPS position, and deletion.
- **Hyperledger Fabric test network** integration for local development.

## Technology Stack

| Component | Technology |
|---|---|
| Blockchain network | Hyperledger Fabric |
| Smart contracts | Go |
| Application layer | Go (Fabric Gateway SDK) |
| Frontend | Svelte, JavaScript, HTML, CSS |
| Frontend tooling | Vite, npm |
| Network containers | Docker, Docker Compose |

## Container Model

```go
type Container struct {
    ID             string  // ledger key
    Type           string  // e.g. 40GP, 40RH, 20OT
    Status         string  // Empty | AtPort | Loaded | InTransit | UnderCustoms | Damaged
    Location       string
    Owner          string
    CargoDesc      string
    GrossWeight    int
    TareWeight     int
    NetWeight      int     // computed as GrossWeight - TareWeight
    SealNumber     string
    Destination    string
    LastInspection string
    TempSetpoint   int     // reefers only
    BillOfLading   string
    CustomsStatus  string
    Position       *Position  // latest GPS fix, nil if never tracked
    Journey        []ContainerEvent
}

type ContainerEvent struct {
    Timestamp string
    Action    string
    Location  string
    By        string
    Notes     string
}

type Position struct {
    Lat       string  // decimal degrees, stored as string for determinism
    Lon       string
    Timestamp string  // RFC3339 UTC (fix time)
    Speed     string  // knots
    Heading   string  // degrees
    Source    string  // e.g. "GPS:track.nmea"
    Temperature string // °C (may be empty)
}
```

## Project Structure

```text
asset-management/
├── install-fabric.sh
├── run-all.sh                         # one-command start / stop
├── application-gateway-go/            # CLI gateway (Fabric Gateway SDK)
│   ├── containercmd.go
│   ├── go.mod
│   └── go.sum
├── gps-ingestor/                      # parses GPS/nav files -> UpdatePosition
│   ├── main.go
│   ├── parse.go
│   ├── parse_test.go
│   ├── testdata/
│   ├── go.mod
│   └── go.sum
├── chaincode/
│   └── asset-transfer-basic/
│       └── chaincode-go/              # custom smart contract
│           ├── chaincode/
│           │   ├── smartcontract.go
│           │   └── smartcontract_test.go
│           ├── go.mod
│           └── go.sum
├── asset-management-gui/
│   ├── run.sh                         # start network + chaincode + backend
│   ├── server/                        # REST API + static server (Go)
│   │   ├── main.go
│   │   ├── go.mod
│   │   └── go.sum
│   └── web/                           # Svelte + Vite frontend
│       ├── index.html
│       ├── package.json
│       ├── vite.config.js
│       └── src/
│           ├── App.svelte
│           ├── api.js
│           ├── stores.js
│           ├── utils.js
│           └── lib/
│               ├── ContainerFilters.svelte
│               ├── ContainerTable.svelte
│               ├── JourneyTimeline.svelte
│               ├── StatsPanel.svelte
│               ├── Toast.svelte
│               └── dialogs/
└── fabric-samples/                    # downloaded Fabric samples + test network
```

The Hyperledger Fabric samples and generated network files are intentionally excluded from this repository. They are downloaded or generated during setup.

## Prerequisites

Install the following before proceeding:

- Git
- Go, compatible with the project's Go modules (`go.mod` specifies Go 1.25.0)
- Docker Engine
- Docker Compose
- Node.js and npm
- `jq`
- A modern web browser

On Arch Linux, the main packages can be installed with:

```bash
sudo pacman -S git go docker docker-compose nodejs npm jq
```

Enable Docker:

```bash
sudo systemctl enable --now docker
```

If Docker reports a permission error, add your user to the Docker group:

```bash
sudo usermod -aG docker "$USER"
```

Log out and log back in for the group change to take effect. Membership in the Docker group grants root-equivalent privileges, so only do this on a machine you administer.

Verify the tools:

```bash
git --version
go version
docker --version
docker compose version
node --version
npm --version
jq --version
```

## Quick Start (one command)

After the prerequisites above and installing Fabric (step 2), a single script brings up the network, deploys/upgrades the chaincode, starts the backend and frontend, and cleans up leftover processes and ports on exit:

```bash
./run-all.sh --demo
```

| Flag | Effect |
|---|---|
| `--demo` | also ingest `container_gps_simulation.json` and report the fix count |
| `--no-frontend` | headless: network + chaincode + backend only |
| `--fresh` | tear the network down first (clean ledger) |
| `--down` | stop the app processes **and** the Fabric network |
| `--help` | show usage |

Logs and PIDs are written to `.run/`. Ctrl+C stops the app processes; the Fabric network keeps running until you use `--down`.

## 1. Clone the Repository

```bash
git clone https://github.com/Infinte-aditya/asset-management-hyperledger-fabric.git
cd asset-management-hyperledger-fabric
```

## 2. Install Hyperledger Fabric

The repository includes `install-fabric.sh`, which downloads the Fabric samples, binaries, and Docker images.

Run it from the project root:

```bash
chmod +x install-fabric.sh
./install-fabric.sh
```

This step may take some time and require a substantial amount of disk space.

After installation, verify that the samples and test network are present:

```bash
ls fabric-samples/
ls fabric-samples/test-network/
```

**Important:** The downloaded `fabric-samples/` directory is separate from the custom source files tracked in this repository. Keep the project root and its directory structure intact.

## 3. Start the Fabric Test Network

Move into the test-network directory:

```bash
cd fabric-samples/test-network
```

Start the network and create a channel:

```bash
./network.sh up createChannel -ca
```

This starts the local Fabric network and creates a channel using Certificate Authorities.

Check the Docker containers:

```bash
docker ps
```

To stop the network later:

```bash
./network.sh down
```

Run Fabric network commands from `fabric-samples/test-network` unless otherwise specified.

## 4. Deploy the Custom Chaincode

The project contains modified Go smart-contract files based on Fabric's `asset-transfer-basic` sample. The downloaded upstream sample also contains supporting files (including the generated `mocks` used by the tests) required for building, testing, and deploying the chaincode.

From the project root, copy the custom smart-contract files into the downloaded sample:

```bash
cd /path/to/asset-management

cp chaincode/asset-transfer-basic/chaincode-go/chaincode/smartcontract.go \
   fabric-samples/asset-transfer-basic/chaincode-go/chaincode/

cp chaincode/asset-transfer-basic/chaincode-go/chaincode/smartcontract_test.go \
   fabric-samples/asset-transfer-basic/chaincode-go/chaincode/
```

Replace `/path/to/asset-management` with your actual project directory, or run the commands directly from the project root.

Deploy the chaincode:

```bash
cd fabric-samples/test-network

./network.sh deployCC \
  -ccn container-cc \
  -ccp ../asset-transfer-basic/chaincode-go \
  -ccl go
```

The chaincode name used here is `container-cc`. Use the same name in subsequent Fabric CLI operations. The Go gateway and the web backend both default to `container-cc` and can be overridden with the `CHAINCODE_NAME` environment variable.

**Version compatibility:** The installer may download a different upstream Fabric version over time. If chaincode deployment fails, check the downloaded sample's version and compare its Go dependencies and chaincode structure with this repository's custom files.

## 5. Use the Go Command-Line Gateway

The Go application layer provides command-line operations for interacting with the container chaincode. Run all commands from the `application-gateway-go` module directory:

```bash
cd /path/to/asset-management/application-gateway-go
go mod download
```

The module connects to `mychannel` / `container-cc` on the local test network and reads org1 credentials from `../fabric-samples/test-network/organizations/...`. Override the network settings with `CHANNEL_NAME` and `CHAINCODE_NAME`. Run `go run .` with no arguments to print the full command list.

### Seed sample data

```bash
go run . InitLedger
```

### Register a container

```bash
go run . RegisterContainer MSCU1234567 40GP MSC 3700 USNYC
```

Arguments: `id`, `type`, `owner`, `tareWeight`, `destination`.

### Read a container

```bash
go run . ReadContainer MSCU1234567
```

### List all containers

```bash
go run . GetAllContainers
```

### Filter containers

```bash
go run . QueryContainers '{"status":"InTransit"}'
```

Supported filter keys: `status`, `type`, `owner`, `location`.

### Update a container

```bash
go run . UpdateContainer <id> <type> <status> <location> <owner> <cargoDesc> \
  <grossWeight> <tareWeight> <sealNumber> <destination> <lastInspection> \
  <tempSetpoint> <billOfLading> <customsStatus>
```

### Transfer ownership

```bash
go run . TransferContainer MSCU1234567 "New Owner"
```

### Report a lifecycle event

```bash
go run . ReportEvent MSCU1234567 Loaded CNSHA Terminal-Ops "Loaded aboard vessel"
```

### Report damage

```bash
go run . ReportDamage MSCU1234567 Dent "30cm dent on right wall" Surveyor
```

### Update customs status

```bash
go run . UpdateCustoms MSCU1234567 Hold CBP "Random inspection"
```

### Set reefer temperature

```bash
go run . SetTemperature MAEU7890123 -18 Ops
```

### View a container's journey

```bash
go run . GetContainerHistory MSCU1234567
```

### Update a container's GPS position

```bash
go run . UpdatePosition MSCU1234567 31.2304 121.4737 2026-11-11T09:00:00Z 12.5 180 "GPS:manual" "4.5"
```

Arguments: `id`, `lat`, `lon`, `timestamp` (RFC3339 UTC), `speed` (knots), `heading` (degrees), `source`, `temperature` (°C, may be empty).

### View a container's GPS track

```bash
go run . GetPositionHistory MSCU1234567
```

### Delete a container

```bash
go run . DeleteContainer MSCU1234567
```

If the command-line application reports that it cannot find a network connection profile, certificates, or private keys, check that the Fabric test network is running and that the credentials exist under `fabric-samples/test-network/organizations/`. Do not commit generated certificates, private keys, or other network credentials to Git.

## 6. Run the Web Application

The web application is located in `asset-management-gui/`. It contains a Go backend (REST API + static server) and a Svelte/Vite frontend.

### One-step start (network + chaincode + backend)

```bash
cd asset-management-gui
./run.sh
```

This starts the Fabric test network and channel (if not running), deploys `container-cc` (if not deployed), and starts the backend at http://localhost:8080. Leave it running.

### Start the frontend (development)

In a separate terminal:

```bash
cd asset-management-gui/web
npm install
npm run dev
```

Open the URL printed by Vite (commonly http://localhost:5173). The dev server proxies `/api` requests to the backend on port 8080.

### Single-port build (optional)

To serve a production build and the API from one port:

```bash
cd asset-management-gui/web
npm run build            # outputs to web/dist

cd ../server
WEB_DIR=../web/dist go run .   # http://localhost:8080
```

If the interface loads but asset operations fail, check the backend terminal, its API configuration, and its access to the Fabric network. If it is the first run, click **Load sample data** in the UI (submits `InitLedger`).

### Backend API

The backend exposes a REST API over the chaincode:

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/health` | Network connectivity check |
| GET | `/api/containers` | List all containers |
| GET | `/api/containers/query` | Filter by `status`/`type`/`owner`/`location` |
| GET | `/api/containers/{id}` | Read a container |
| GET | `/api/containers/{id}/history` | Journey history |
| POST | `/api/containers` | Register a container |
| PUT | `/api/containers/{id}` | Update a container |
| DELETE | `/api/containers/{id}` | Delete a container |
| POST | `/api/containers/{id}/transfer` | Transfer ownership |
| POST | `/api/containers/{id}/events` | Report a lifecycle event |
| POST | `/api/containers/{id}/damage` | Report damage |
| POST | `/api/containers/{id}/customs` | Update customs status |
| POST | `/api/containers/{id}/temperature` | Set reefer temperature |
| GET | `/api/containers/{id}/positions` | GPS position history (oldest→newest) |
| POST | `/api/containers/{id}/position` | Update GPS position |
| POST | `/api/seed` | Load sample data (`InitLedger`) |

Backend environment variables: `PORT` (default `8080`), `CHANNEL_NAME` (default `mychannel`), `CHAINCODE_NAME` (default `container-cc`), `CRYPTO_PATH` (auto-detected), and `WEB_DIR` (static files directory).

## 7. GPS Tracking & Auto-Update

Containers can carry a live GPS position. Because Fabric chaincode is deterministic and cannot call external services, GPS data is ingested **off-chain** by the `gps-ingestor` tool and submitted to the ledger as `UpdatePosition` transactions.

- The latest fix is embedded on the container as `Position` (shown in the container table and detail dialog).
- The full track is stored separately under composite keys; `GetPositionHistory` returns it oldest→newest, and the detail dialog plots it on a **Leaflet/OpenStreetMap map** (with antimeridian unwrapping so Pacific crossings render correctly).
- The **first fix on a `Loaded` container auto-transitions it to `InTransit`** and appends one `InTransit` journey entry.
- Each fix may carry a `Temperature` (°C).

Coordinates are stored as strings (chaincode determinism) and validated to ±90 / ±180.

### Ingesting a GPS/nav file

```bash
cd gps-ingestor

# Preview: parse, map, and print every planned transaction. No ledger writes.
go run . -file /path/to/track.nmea -dry-run

# Submit for real.
go run . -file /path/to/track.nmea
```

Supported formats (auto-detected): **NMEA** (`RMC`/`GGA`), **CSV**, **JSON**, **GPX**. Records are mapped to containers by explicit `-container`, by container ID, or by vessel (`Location == "OnBoard:<vessel>"`); unmapped records are skipped with a warning — never guessed. Submissions are rate-limited (`-rate`, default 4/s) and **idempotent**: the track key derives from the container ID and fix time, so re-running the same file does not duplicate history. Run `go run . -h` for all flags.

## Recording GPS stops from the UI

Open any container's detail dialog (or its row's map-pin action) and choose **Add GPS Stop**:

- **Single stop** — enter latitude/longitude only; timestamp defaults to now and source defaults to `GPS:ui`. Click anywhere on the small map to drop a pin, then click **Add stop**.
- **Bulk paste** — one `lat, lon[, temp]` per line; each fix is spaced 60s apart so the track orders correctly.

The track renders on an OpenStreetMap Leaflet map with antimeridian unwrapping, and Position History shows fix count, distance travelled and elapsed time.

## Example routes

Ready-to-load shipments live in `examples/` (JSON, CSV, GPX). Load any route in one command:

```bash
./examples/load-example.sh examples/shanghai-rotterdam.csv
./examples/load-example.sh examples/reefer-atlantic.json 40RH "Reefer Co" 3800 NLRTM
```

See `examples/README.md` for the full list and per-format instructions.

## 8. Running Tests

The custom chaincode includes Go tests that use generated mocks shipped with the downloaded Fabric sample. Because the tests import the sample's `mocks` package, run them from the copy inside `fabric-samples` after step 4:

```bash
cd fabric-samples/asset-transfer-basic/chaincode-go
go test ./...
```

Running `go test ./...` from the repository's own `chaincode/asset-transfer-basic/chaincode-go/` directory fails to resolve the `mocks` package, since that supporting code lives only in the downloaded sample.

The GPS ingestor has its own tests and fixtures and needs no network:

```bash
cd gps-ingestor
go test ./...
```

## Troubleshooting

### Docker permission denied

Verify that Docker is running:

```bash
sudo systemctl status docker
```

If you just added yourself to the Docker group, log out and log back in.

### Fabric network is not running

Check running containers:

```bash
docker ps
```

If necessary, restart the network from `fabric-samples/test-network`:

```bash
./network.sh down
./network.sh up createChannel -ca
```

Redeploy the chaincode afterward if needed.

### Chaincode deployment fails

Check the following:

- Fabric binaries and Docker images were installed successfully.
- Docker is running.
- The network and channel have been created.
- The custom smart-contract files were copied into the downloaded sample.
- The downloaded Fabric sample version is compatible with the custom chaincode.
- Go dependencies can be downloaded.

### The CLI or GUI cannot connect to the network

- Confirm the test network is running and `container-cc` is deployed on `mychannel`.
- For the CLI, confirm the credentials exist under `fabric-samples/test-network/organizations/` (the gateway reads them relative to `application-gateway-go/`).
- For the backend, check the terminal output and its `CRYPTO_PATH` / `CHAINCODE_NAME` settings.

### Missing certificates or private keys

Start and configure the Fabric test network so its organization identities and certificates are generated. The project intentionally excludes generated network identities from version control.

### Port already in use

Inspect the terminal output to identify the port in use. Stop the conflicting process or configure the application to use another available port (for example `PORT=9090`).

## Security Notes

- Never commit private keys, real `.env` files, access tokens, or credentials.
- Keep generated Fabric organization identities and network artifacts out of Git.
- Do not expose development API servers or Fabric ports to untrusted networks without appropriate security controls.
- Use a private repository if your source code or configuration contains confidential business information.

## Current Scope and Future Improvements

The repository provides the container-tracking smart contract, the Go CLI gateway, the REST backend, and the Svelte web interface. The Fabric network and application must be configured together for end-to-end operation.

Potential improvements include:

- User authentication and authorization, plus Fabric attribute-based access control on sensitive operations.
- Automated end-to-end tests that exercise the gateway and REST API against a running network.
- More detailed input validation and error handling.
- Improved filtering, sorting, and pagination in the web interface.
- Bulk container operations and richer reporting.

## License

No license is currently specified. Unless a license is added, the repository should not be assumed to grant others permission to reuse, modify, or redistribute the code.
