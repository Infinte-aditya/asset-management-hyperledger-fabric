// Command gps-ingestor parses a GPS/nav file (JSON, CSV, NMEA or GPX), maps
// each reading to a container, and submits UpdatePosition transactions to the
// container chaincode through the Hyperledger Fabric Gateway.
//
// Usage:
//
//	gps-ingestor -file <path> [-dry-run] [-out <path>] [-format auto|csv|json|nmea|gpx]
//	             [-container <ID>] [-vessel <NAME>] [-rate <n>] [-date <YYYY-MM-DD>]
//	             [-delimiter <c>]
//
// Env overrides: CHAINCODE_NAME, CHANNEL_NAME, CRYPTO_PATH.
package main

import (
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID        = "Org1MSP"
	gatewayPeer  = "peer0.org1.example.com"
	peerEndpoint = "dns:///localhost:7051"
	defaultCC    = "container-cc"
	defaultChan  = "mychannel"
)

func main() {
	var (
		filePath  = flag.String("file", "", "path to GPS/nav file (required)")
		dryRun    = flag.Bool("dry-run", false, "parse + map + print planned transactions as JSON lines (no network)")
		outPath   = flag.String("out", "", "optional: write dry-run plan to file")
		format    = flag.String("format", "auto", "auto|csv|json|nmea|gpx")
		container = flag.String("container", "", "apply all records to one container")
		vessel    = flag.String("vessel", "", "override vessel mapping")
		rate      = flag.Int("rate", 4, "max transactions per second")
		date      = flag.String("date", "", "YYYY-MM-DD needed for GGA-only files lacking a date")
		delim     = flag.String("delimiter", ",", "CSV field delimiter")
	)
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "error: -file is required")
		flag.Usage()
		os.Exit(2)
	}

	delimRune := rune(',')
	if *delim != "" {
		delimRune = []rune(*delim)[0]
	}

	opts := Options{
		Format:    *format,
		Container: *container,
		Vessel:    *vessel,
		Date:      *date,
		Delimiter: delimRune,
		Source:    "GPS:" + filepath.Base(*filePath),
	}

	records, parseErrs, err := Parse(*filePath, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	for _, pe := range parseErrs {
		fmt.Fprintf(os.Stderr, "warning: %v\n", pe)
	}

	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "error: zero valid records parsed from file")
		os.Exit(1)
	}

	if *dryRun {
		plan := Plan(records, opts, nil)
		w := io.Writer(os.Stdout)
		if *outPath != "" {
			f, err := os.Create(*outPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: cannot create -out file: %v\n", err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}
		enc := json.NewEncoder(w)
		for _, fix := range plan {
			_ = enc.Encode(fix)
		}
		planned := 0
		for _, fix := range plan {
			if fix.Status == "planned" {
				planned++
			}
		}
		fmt.Fprintf(os.Stderr, "dry-run: %d parsed, %d planned, %d unmapped\n",
			len(records), planned, len(plan)-planned)
		return
	}

	// Live mode.
	runLive(records, opts, *rate)
}

func runLive(records []RawRecord, opts Options, rate int) {
	gw, conn, err := connectGateway()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot connect to Fabric network: %v\n", err)
		fmt.Fprintln(os.Stderr, "hint: start the test-network first (./network.sh up createChannel)")
		os.Exit(1)
	}
	defer gw.Close()
	defer conn.Close()

	chaincodeName := envOr("CHAINCODE_NAME", defaultCC)
	channelName := envOr("CHANNEL_NAME", defaultChan)
	contract := gw.GetNetwork(channelName).GetContract(chaincodeName)

	var index map[string]ContainerInfo
	if needsIndex(records, opts) {
		result, err := contract.EvaluateTransaction("GetAllContainers")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: GetAllContainers failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "hint: is the chaincode deployed and InitLedger run? (go run . InitLedger)")
			os.Exit(1)
		}
		index = buildIndex(result)
	}

	plan := Plan(records, opts, index)

	var planned []Fix
	for _, fix := range plan {
		if fix.Status == "planned" {
			planned = append(planned, fix)
		} else {
			fmt.Fprintf(os.Stderr, "skipping %s: %s\n", fix.RawLine, fix.Reason)
		}
	}
	if len(planned) == 0 {
		fmt.Fprintln(os.Stderr, "error: no records could be mapped to a container")
		os.Exit(1)
	}

	// Sort by timestamp ascending (RFC3339 sorts lexicographically = chronologically).
	sortFixesByTimestamp(planned)

	lim := newLimiter(rate)
	submit := func(fix Fix) error { return submitWithRetry(contract, fix, lim) }
	submitted, failed, notFound := runSubmissions(submit, planned)

	fmt.Fprintf(os.Stderr, "live: %d submitted, %d failed\n", submitted, failed)
	if failed > 0 || notFound {
		os.Exit(1)
	}
}

// submitFunc submits a single fix and returns the outcome.
type submitFunc func(Fix) error

// runSubmissions drives submission over all planned fixes, tracking per-record
// outcomes. It returns the counts of submitted and failed fixes, plus whether
// any "does not exist" (container-not-on-ledger) error was encountered. A
// connection/deadline error aborts the run early. It exits with a non-zero
// status if any fix fails (handled by the caller via the returned counts).
func runSubmissions(submit submitFunc, planned []Fix) (submitted, failed int, notFound bool) {
	for _, fix := range planned {
		err := submit(fix)
		if err == nil {
			submitted++
			fmt.Printf("submitted %s @ %s\n", fix.Container, fix.Timestamp)
			continue
		}
		failed++
		var nf *notFoundError
		if errors.As(err, &nf) {
			notFound = true
			fmt.Fprintf(os.Stderr, "error: container %q does not exist on ledger\n", nf.id)
			fmt.Fprintln(os.Stderr, "hint: run `go run . InitLedger` (from application-gateway-go) to load sample containers")
			continue
		}
		if isConnError(err) {
			fmt.Fprintf(os.Stderr, "error: network unreachable after retries: %v\n", err)
			fmt.Fprintln(os.Stderr, "hint: start the test-network first (./network.sh up createChannel)")
			return submitted, failed, notFound
		}
		fmt.Fprintf(os.Stderr, "error: submit failed for %s: %v\n", fix.Container, err)
	}
	return submitted, failed, notFound
}

func sortFixesByTimestamp(fixes []Fix) {
	for i := 1; i < len(fixes); i++ {
		for j := i; j > 0 && fixes[j].Timestamp < fixes[j-1].Timestamp; j-- {
			fixes[j], fixes[j-1] = fixes[j-1], fixes[j]
		}
	}
}

type notFoundError struct{ id string }

func (e *notFoundError) Error() string { return fmt.Sprintf("container %q does not exist", e.id) }

func isConnError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unavailable") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "failed to connect") ||
		strings.Contains(msg, "deadline")
}

// submitWithRetry submits a single UpdatePosition, retrying 3x on transient
// connection/deadline errors with exponential backoff.
func submitWithRetry(contract *client.Contract, fix Fix, lim *limiter) error {
	lim.Wait()
	var err error
	for attempt := 0; attempt <= 3; attempt++ {
		_, err = contract.SubmitTransaction("UpdatePosition",
			fix.Container,
			strconv.FormatFloat(fix.Lat, 'f', -1, 64),
			strconv.FormatFloat(fix.Lon, 'f', -1, 64),
			fix.Timestamp, fix.Speed, fix.Heading, fix.Source, fix.Temperature)
		if err == nil {
			return nil
		}
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "does not exist") {
			return &notFoundError{id: fix.Container}
		}
		if isConnError(err) && attempt < 3 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			time.Sleep(backoff)
			continue
		}
		return err
	}
	return err
}

// --- token-bucket limiter (min interval between submissions) ---

type limiter struct {
	interval time.Duration
	last     time.Time
	mu       sync.Mutex
}

func newLimiter(rate int) *limiter {
	iv := time.Duration(0)
	if rate > 0 {
		iv = time.Second / time.Duration(rate)
	}
	return &limiter{interval: iv}
}

func (l *limiter) Wait() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.interval <= 0 {
		return
	}
	now := time.Now()
	if l.last.IsZero() {
		l.last = now
		return
	}
	next := l.last.Add(l.interval)
	if now.Before(next) {
		time.Sleep(next.Sub(now))
		l.last = next
	} else {
		l.last = now
	}
}

// --- Gateway boilerplate (reused from application-gateway-go/containercmd.go,
// with CRYPTO_PATH resolution from asset-management-gui/server/main.go) ---

func connectGateway() (*client.Gateway, *grpc.ClientConn, error) {
	conn, err := newGrpcConnection()
	if err != nil {
		return nil, nil, err
	}
	id, err := newIdentity()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	sign, err := newSign()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	return gw, conn, nil
}

func newGrpcConnection() (*grpc.ClientConn, error) {
	cryptoPath := resolveCryptoPath()
	certificatePEM, err := os.ReadFile(path.Join(cryptoPath, "peers", "peer0.org1.example.com", "tls", "ca.crt"))
	if err != nil {
		return nil, fmt.Errorf("failed to read TLS certificate: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)
	conn, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}
	return conn, nil
}

func newIdentity() (*identity.X509Identity, error) {
	cryptoPath := resolveCryptoPath()
	certificatePEM, err := readFirstFile(path.Join(cryptoPath, "users", "User1@org1.example.com", "msp", "signcerts"))
	if err != nil {
		return nil, fmt.Errorf("failed to read client certificate: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		return nil, err
	}
	return identity.NewX509Identity(mspID, certificate)
}

func newSign() (identity.Sign, error) {
	cryptoPath := resolveCryptoPath()
	privateKeyPEM, err := readFirstFile(path.Join(cryptoPath, "users", "User1@org1.example.com", "msp", "keystore"))
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		return nil, err
	}
	return identity.NewPrivateKeySign(privateKey)
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

func resolveCryptoPath() string {
	if p := os.Getenv("CRYPTO_PATH"); p != "" {
		return p
	}
	candidates := []string{
		"../fabric-samples/test-network/organizations/peerOrganizations/org1.example.com",
		"../../fabric-samples/test-network/organizations/peerOrganizations/org1.example.com",
		"fabric-samples/test-network/organizations/peerOrganizations/org1.example.com",
		"../../test-network/organizations/peerOrganizations/org1.example.com",
		"test-network/organizations/peerOrganizations/org1.example.com",
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	return candidates[0]
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
