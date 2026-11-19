# Example GPS Routes

Ready-to-load container shipments, in the formats the `gps-ingestor` accepts:
JSON, CSV, NMEA and GPX.

| File | Container | Route | Notes |
|---|---|---|---|
| `shanghai-rotterdam.json` | `CONT-EU-55201B` | Shanghai → Singapore → Suez → Rotterdam | Asia–Europe ocean leg (JSON) |
| `shanghai-rotterdam.csv` | `CONT-EU-55201B` | Shanghai → Singapore → Suez → Rotterdam | Same route (CSV; container in file) |
| `shanghai-rotterdam.gpx` | `CONT-EU-55201B` | Shanghai → Singapore → Suez → Rotterdam | Same route (GPX; pass `-container`) |
| `singapore-jebel-ali.json` | `CONT-ME-77120C` | Singapore → Colombo → Mumbai → Jebel Ali | Indian Ocean / Gulf |
| `la-newyork.json` | `CONT-US-31044D` | Los Angeles → Dallas → Memphis → New York | US inland intermodal |
| `reefer-atlantic.json` | `CONT-RF-90877E` | Rotterdam → Halifax → New York | Reefer, ~-18 °C |
| `../container_gps_simulation.json` | `CONT-UHF-98231A` | Shanghai → Long Beach | Original Pacific demo |

## Load one in a single command

From the repo root (Fabric network running, chaincode deployed):

```bash
./examples/load-example.sh examples/shanghai-rotterdam.json
./examples/load-example.sh examples/shanghai-rotterdam.csv
# GPX + NMEA have no container column, so pass one explicitly:
./examples/load-example.sh examples/shanghai-rotterdam.gpx 40GP "Euro Freight" 3700 NLRTM
```

The script registers the container if it is not on the ledger, marks it `Loaded`,
ingests the fixes, then prints the resulting track.

## Or do it by hand

JSON / CSV (container is inside the file):

```bash
cd application-gateway-go
go run . RegisterContainer CONT-EU-55201B 40GP "Euro Freight" 3700 NLRTM
go run . ReportEvent CONT-EU-55201B Loaded CNSHA Depot-Ops "Loaded aboard vessel"

cd ../gps-ingestor
go run . -file ../examples/shanghai-rotterdam.json -dry-run   # preview
go run . -file ../examples/shanghai-rotterdam.json            # submit
```

GPX / NMEA (no container column → supply `-container`):

```bash
cd gps-ingestor
go run . -container CONT-EU-55201B -file ../examples/shanghai-rotterdam.gpx -dry-run
go run . -container CONT-EU-55201B -file ../examples/shanghai-rotterdam.gpx
```

NMEA files should use `$GPRMC`/`$GPGGA` pairs (see `gps-ingestor/testdata/sample.nmea`).

## Add stops from the UI

Open a container (or its row's map-pin action) and choose **Add GPS Stop**:

- **Single stop** — enter latitude/longitude only; time defaults to now, source to `GPS:ui`. Click the map to pick a coordinate.
- **Bulk paste** — one `lat, lon[, temp]` per line; each becomes a fix 60 s apart.

The map in the container detail dialog renders the track, including Pacific
dateline crossings.
