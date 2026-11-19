# Example GPS Routes

Ready-to-load container shipments, in the same JSON format the `gps-ingestor`
accepts (`containerId`, `timestamp`, `location.latitude/longitude`,
`temperatureCelsius`, `status`, `txId`).

| File | Container | Route | Notes |
|---|---|---|---|
| `shanghai-rotterdam.json` | `CONT-EU-55201B` | Shanghai → Singapore → Suez → Rotterdam | Asia–Europe ocean leg |
| `singapore-jebel-ali.json` | `CONT-ME-77120C` | Singapore → Colombo → Mumbai → Jebel Ali | Indian Ocean / Gulf |
| `la-newyork.json` | `CONT-US-31044D` | Los Angeles → Dallas → Memphis → New York | US inland intermodal |
| `reefer-atlantic.json` | `CONT-RF-90877E` | Rotterdam → Halifax → New York | Reefer, ~-18 °C |
| `../container_gps_simulation.json` | `CONT-UHF-98231A` | Shanghai → Long Beach | Original Pacific demo |

## Load one in a single command

From the repo root (Fabric network running, chaincode deployed):

```bash
./examples/load-example.sh examples/shanghai-rotterdam.json
# optional: [Type] [Owner] [TareWeight] [Destination]
./examples/load-example.sh examples/reefer-atlantic.json 40RH "Reefer Co" 3800 NLRTM
```

The script registers the container if it is not on the ledger, marks it `Loaded`,
ingests the fixes, then prints the resulting track.

## Or do it by hand

```bash
cd application-gateway-go
go run . RegisterContainer CONT-EU-55201B 40GP "Euro Freight" 3700 NLRTM
go run . ReportEvent CONT-EU-55201B Loaded CNSHA Depot-Ops "Loaded aboard vessel"

cd ../gps-ingestor
go run . -file ../examples/shanghai-rotterdam.json -dry-run   # preview
go run . -file ../examples/shanghai-rotterdam.json            # submit
```

## Add stops from the UI

Open a container (or its row's map-pin action) and choose **Add GPS Stop**:

- **Single stop** — enter latitude/longitude only; time defaults to now, source to `GPS:ui`.
- **Bulk paste** — one `lat, lon[, temp]` per line; each becomes a fix 60 s apart.

The map in the container detail dialog renders the track, including Pacific
dateline crossings.
