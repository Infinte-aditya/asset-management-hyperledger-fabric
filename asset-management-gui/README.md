# Container Ledger — Web GUI

A graphical interface for the Hyperledger Fabric container-tracking system. Manage shipping-container records on the blockchain from your browser: register, view, update, transfer, report events/damage/customs, and delete containers, with a full journey timeline per container.

**Architecture:** a Go server (Fabric Gateway SDK) exposes a REST API and serves the frontend; the Svelte/Vite app talks to that API.

```
Browser (Svelte)  →  Go REST server :8080  →  Fabric Gateway SDK  →  test-network peers  →  chaincode "container-cc"
```

## Project Structure

```
asset-management-gui/
├── server/
│   ├── main.go        # REST API + static file server (Fabric Gateway SDK)
│   └── go.mod
├── web/               # Svelte + Vite frontend
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── src/
│       ├── App.svelte
│       ├── api.js
│       ├── stores.js
│       ├── utils.js
│       └── lib/       # ContainerTable, ContainerFilters, JourneyTimeline,
│                      # StatsPanel, Toast, and dialogs/*
└── run.sh             # Starts network + chaincode + backend
```

## Prerequisites

1. Go (1.25+)
2. Node.js and npm
3. Fabric test network running with the `container-cc` chaincode deployed (see below)

## Quick Start

### 1. Run the network, chaincode and backend

```bash
./run.sh
```

This automatically starts the Fabric test network and channel (if not running), deploys the `container-cc` chaincode (if not deployed), then starts the backend at http://localhost:8080.

### 2. Start the frontend (development)

In a second terminal:

```bash
cd web
npm install
npm run dev
```

Open the URL printed by Vite (commonly http://localhost:5173). The dev server proxies `/api` to the backend on port 8080.

### 3. Load sample data

If the ledger is empty, click **Load sample data** in the UI (submits `InitLedger` — 6 demo containers).

## Features

- **Dashboard** — totals by status/type, container counts, and other live stats
- **Container table** — filterable and sortable, with status chips and owner info
- **Register Container** — validated form; submits a real blockchain transaction
- **Container details** — full record view
- **Transfer** — change ownership (journey entry records previous owner)
- **Report event** — GateIn, Loaded, Discharged, GateOut, InTransit, CustomsHold/Release, …
- **Report damage** — type + description, sets status to `Damaged`
- **Update customs** — Clear / Hold / Release
- **Journey timeline** — immutable chain of custody per container
- **GPS position** — latest fix in the table, with a Live Position block and lazy Position History in the detail dialog
- **Track map** — Leaflet/OpenStreetMap map of the container's route (antimeridian-aware)
- **Delete** — with confirmation dialog
- **Live status** — connection pill + offline banner with retry; auto-refresh

## REST API

| Method | Endpoint | Description |
|--------|----------------------------------|------------------------------------|
| GET | `/api/health` | Network connectivity check |
| GET | `/api/containers` | List all containers |
| GET | `/api/containers/query` | Filter by `status`/`type`/`owner`/`location` |
| GET | `/api/containers/{id}` | Read container by ID |
| GET | `/api/containers/{id}/history` | Container journey history |
| POST | `/api/containers` | Register container (JSON body) |
| PUT | `/api/containers/{id}` | Update container |
| DELETE | `/api/containers/{id}` | Delete container |
| POST | `/api/containers/{id}/transfer` | Transfer to new owner |
| POST | `/api/containers/{id}/events` | Report lifecycle event |
| POST | `/api/containers/{id}/damage` | Report damage |
| POST | `/api/containers/{id}/customs` | Update customs status |
| POST | `/api/containers/{id}/temperature` | Set reefer temperature |
| GET | `/api/containers/{id}/positions` | GPS position history |
| POST | `/api/containers/{id}/position` | Update GPS position |
| POST | `/api/seed` | Load sample data (`InitLedger`) |

Example:

```bash
curl -X POST http://localhost:8080/api/containers \
  -H "Content-Type: application/json" \
  -d '{"id":"MSCU1234567","type":"40GP","owner":"MSC","tareWeight":3700,"destination":"USNYC"}'

curl http://localhost:8080/api/containers

curl -X GET http://localhost:8080/api/containers/query \
  -H "Content-Type: application/json" \
  -d '{"status":"InTransit"}'
```

## Configuration (environment variables)

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `CHANNEL_NAME` | `mychannel` | Fabric channel |
| `CHAINCODE_NAME` | `container-cc` | Deployed chaincode name |
| `CRYPTO_PATH` | auto-detected | Path to `org1.example.com` crypto material |
| `WEB_DIR` | auto-detected (`web`/`../web`) | Static files directory |

To serve a production build from the backend alone:

```bash
cd web && npm run build
cd ../server && WEB_DIR=../web/dist go run .   # http://localhost:8080
```

## Troubleshooting

- **"Offline" banner / 503 errors** — the test network isn't running. Start it:
  `cd ../fabric-samples/test-network && ./network.sh up createChannel`
  If Docker permissions fail: `sudo usermod -aG docker $USER`, then re-login.
- **"container with ID x does not exist" (404)** — the container isn't on the ledger.
- **"already exists" (409)** — a container with that ID is already committed.
- **Connection timeouts** — verify the chaincode is deployed:
  `./network.sh deployCC -ccn container-cc -ccp ../asset-transfer-basic/chaincode-go -ccl go`
