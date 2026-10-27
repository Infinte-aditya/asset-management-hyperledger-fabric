# Asset Management System using Hyperledger Fabric

A blockchain-based asset management application built with **Hyperledger Fabric, Go, and Svelte**. The project includes smart-contract logic for managing assets, a Go application layer, and a web-based graphical interface.

## Features

- **Create assets** with an ID, color, size, owner, and appraised value.
- **Read assets** by ID.
- **Update asset details.**
- **Transfer ownership** of an asset.
- **Delete assets.**
- **View all assets** stored on the ledger.
- **Web interface** for asset registration, listing, transfers, deletion, and asset-related operations.
- **Hyperledger Fabric test network** integration for local development.

## Technology Stack

| Component | Technology |
|---|---|
| Blockchain network | Hyperledger Fabric |
| Smart contracts | Go |
| Application layer | Go |
| Frontend | Svelte, JavaScript, HTML, CSS |
| Frontend tooling | Vite, npm |
| Network containers | Docker, Docker Compose |

## Project Structure

```text
asset-management-hyperledger-fabric/
├── .gitignore
├── install-fabric.sh
├── application-gateway-go/
│   ├── assetManagement.go
│   ├── containercmd.go
│   ├── simpleAsset.go
│   ├── go.mod
│   └── go.sum
├── chaincode/
│   └── asset-transfer-basic/
│       └── chaincode-go/
│           ├── chaincode/
│           │   ├── smartcontract.go
│           │   └── smartcontract_test.go
│           ├── go.mod
│           └── go.sum
└── asset-management-gui/
    ├── run.sh
    ├── server/
    │   ├── main.go
    │   ├── go.mod
    │   └── go.sum
    └── web/
        ├── package.json
        ├── package-lock.json
        ├── index.html
        ├── src/
        └── vite.config.js
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

The project contains modified Go smart-contract files based on Fabric's `asset-transfer-basic` sample. The downloaded upstream sample also contains supporting files required for building and deploying the chaincode.

From the project root, copy the custom smart-contract files into the downloaded sample:

```bash
cd /path/to/asset-management-hyperledger-fabric

cp chaincode/asset-transfer-basic/chaincode-go/chaincode/smartcontract.go \
   fabric-samples/asset-transfer-basic/chaincode-go/chaincode/

cp chaincode/asset-transfer-basic/chaincode-go/chaincode/smartcontract_test.go \
   fabric-samples/asset-transfer-basic/chaincode-go/chaincode/
```

Replace `/path/to/asset-management-hyperledger-fabric` with your actual project directory, or run the commands directly from the project root.

Deploy the chaincode:

```bash
cd fabric-samples/test-network

./network.sh deployCC \
  -ccn asset \
  -ccp ../asset-transfer-basic/chaincode-go \
  -ccl go
```

The chaincode name used here is `asset`. Use the same name in subsequent Fabric CLI operations.

**Version compatibility:** The installer may download a different upstream Fabric version over time. If chaincode deployment fails, check the downloaded sample's version and compare its Go dependencies and chaincode structure with this repository's custom files.

## 5. Use the Go Asset Management Application

The Go application layer provides command-line operations for interacting with the asset-management functionality.

Open a new terminal and navigate to the Go application module:

```bash
cd /path/to/asset-management-hyperledger-fabric/application-gateway-go
```

Download the Go dependencies:

```bash
go mod download
```

The commands below follow the project's usage reference. The application must be able to connect to the running Fabric network and access the required network credentials.

### Create an asset

```bash
go run . CreateAsset asset7 pink 11 Aditya 345
```

Arguments:

- `asset7`: asset ID
- `pink`: color
- `11`: size
- `Aditya`: owner
- `345`: appraised value

### Read an asset

```bash
go run . ReadAsset asset7
```

### Update an asset

```bash
go run . UpdateAsset asset7 blue 12 Aditya 400
```

### Transfer an asset

```bash
go run . TransferAsset asset7 NewOwner
```

### Delete an asset

```bash
go run . DeleteAsset asset7
```

### View all assets

```bash
go run . GetAllAssets
```

If the command-line application reports that it cannot find a network connection profile, certificates, or private keys, check the Fabric network setup and the paths configured in the application. Do not commit generated certificates, private keys, or other network credentials to Git.

## 6. Run the Web Application

The web application is located in `asset-management-gui/`. It contains a Go backend and a Svelte/Vite frontend.

### Install frontend dependencies

Open a terminal:

```bash
cd /path/to/asset-management-hyperledger-fabric/asset-management-gui/web
npm install
```

### Start the Go backend

Open another terminal:

```bash
cd /path/to/asset-management-hyperledger-fabric/asset-management-gui/server
go mod download
go run .
```

Keep this terminal open while using the interface.

### Start the frontend

In a separate terminal:

```bash
cd /path/to/asset-management-hyperledger-fabric/asset-management-gui/web
npm run dev
```

Open the local URL printed by Vite in your browser. It is commonly `http://localhost:5173`, but use the address shown in your terminal.

The frontend requires the backend to be running and correctly configured. If the interface loads but asset operations fail, check the backend terminal, its API configuration, and its access to the Fabric network.

The repository also includes an executable `asset-management-gui/run.sh` script. You can inspect it with:

```bash
cat asset-management-gui/run.sh
```

Follow the script's own instructions if you prefer to use it to launch the application.

## 7. Running Tests

The custom chaincode includes Go tests.

Run them from the chaincode module directory:

```bash
cd /path/to/asset-management-hyperledger-fabric/chaincode/asset-transfer-basic/chaincode-go
go test ./...
```

This checks the copied Go package and its tests. The downloaded upstream Fabric sample may also contain additional supporting code not included in the custom-source directory.

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

### GUI cannot connect to the backend

Check that the Go backend is running and that the frontend is configured to use its API address. Review both terminal outputs for errors.

### Missing certificates or private keys

Start and configure the Fabric test network so its organization identities and certificates are generated. The project intentionally excludes generated network identities from version control.

### Port already in use

Inspect the terminal output to identify the port in use. Stop the conflicting process or configure the application to use another available port.

## Security Notes

- Never commit private keys, real `.env` files, access tokens, or credentials.
- Keep generated Fabric organization identities and network artifacts out of Git.
- Do not expose development API servers or Fabric ports to untrusted networks without appropriate security controls.
- Use a private repository if your source code or configuration contains confidential business information.

## Current Scope and Future Improvements

The repository provides the asset-management smart contract, Go application files, and web-interface source. The Fabric network and application must be configured together for end-to-end operation.

Potential improvements include:

- Automated tests for all asset operations.
- More detailed input validation and error handling.
- User authentication and authorization.
- Improved filtering and sorting in the web interface.
- Bulk asset operations.
- A reproducible, automated setup process.

## License

No license is currently specified. Unless a license is added, the repository should not be assumed to grant others permission to reuse, modify, or redistribute the code.

