// Container Shipment Management — REST API Gateway.
//
// Serves the Svelte frontend and proxies requests to the
// Hyperledger Fabric test network through the Fabric Gateway SDK.
//
// Endpoints:
//
//	GET    /api/health                       network connectivity check
//	GET    /api/containers                   list all containers
//	GET    /api/containers/{id}              read a single container
//	POST   /api/containers                   register a new container
//	PUT    /api/containers/{id}              update a container
//	DELETE /api/containers/{id}              delete (de-register) a container
//	POST   /api/containers/{id}/transfer     transfer container to new owner
//	POST   /api/containers/{id}/events       report a lifecycle event
//	GET    /api/containers/{id}/history      get container journey history
//	POST   /api/containers/{id}/damage       report container damage
//	POST   /api/containers/{id}/customs      update customs status
//	POST   /api/containers/{id}/temperature  set temperature for reefers
//	GET    /api/containers/query             query containers by status/type/owner/location
//	POST   /api/seed                        load sample data (InitLedger)
package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID          = "Org1MSP"
	gatewayPeer    = "peer0.org1.example.com"
	peerEndpoint   = "dns:///localhost:7051"
	defaultPort    = "8080"
	defaultChannel = "mychannel"
	defaultCC      = "container-cc"
)

type server struct {
	contract  *client.Contract
	channel   string
	chaincode string
}

type containerInput struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	Location       string `json:"location"`
	Owner          string `json:"owner"`
	CargoDesc      string `json:"cargoDesc"`
	GrossWeight    *int   `json:"grossWeight"`
	TareWeight     *int   `json:"tareWeight"`
	SealNumber     string `json:"sealNumber"`
	Destination    string `json:"destination"`
	LastInspection string `json:"lastInspection"`
	TempSetpoint   *int   `json:"tempSetpoint"`
	BillOfLading   string `json:"billOfLading"`
	CustomsStatus  string `json:"customsStatus"`
}

type eventInput struct {
	Action   string `json:"action"`
	Location string `json:"location"`
	By       string `json:"by"`
	Notes    string `json:"notes"`
}

type damageInput struct {
	DamageType  string `json:"damageType"`
	Description string `json:"description"`
	ReportedBy  string `json:"reportedBy"`
}

type customsInput struct {
	CustomsStatus string `json:"customsStatus"`
	By            string `json:"by"`
	Notes         string `json:"notes"`
}

type tempInput struct {
	TempSetpoint int    `json:"tempSetpoint"`
	By           string `json:"by"`
}

type queryInput struct {
	Status   string `json:"status"`
	Type     string `json:"type"`
	Owner    string `json:"owner"`
	Location string `json:"location"`
}

func main() {
	channel := envOr("CHANNEL_NAME", defaultChannel)
	chaincode := envOr("CHAINCODE_NAME", defaultCC)
	port := envOr("PORT", defaultPort)

	gw, err := connectGateway()
	if err != nil {
		log.Fatalf("Failed to connect to Fabric gateway: %v", err)
	}
	defer gw.Close()

	network := gw.GetNetwork(channel)
	s := &server{
		contract:  network.GetContract(chaincode),
		channel:   channel,
		chaincode: chaincode,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/containers", s.handleListContainers)
	mux.HandleFunc("GET /api/containers/query", s.handleQueryContainers)
	mux.HandleFunc("GET /api/containers/{id}", s.handleGetContainer)
	mux.HandleFunc("GET /api/containers/{id}/history", s.handleGetHistory)
	mux.HandleFunc("POST /api/containers", s.handleRegisterContainer)
	mux.HandleFunc("PUT /api/containers/{id}", s.handleUpdateContainer)
	mux.HandleFunc("DELETE /api/containers/{id}", s.handleDeleteContainer)
	mux.HandleFunc("POST /api/containers/{id}/transfer", s.handleTransferContainer)
	mux.HandleFunc("POST /api/containers/{id}/events", s.handleReportEvent)
	mux.HandleFunc("POST /api/containers/{id}/damage", s.handleReportDamage)
	mux.HandleFunc("POST /api/containers/{id}/customs", s.handleCustomsAction)
	mux.HandleFunc("POST /api/containers/{id}/temperature", s.handleSetTemperature)
	mux.HandleFunc("POST /api/seed", s.handleSeed)
	mux.Handle("/", http.FileServer(http.Dir(resolveWebDir())))

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           logRequests(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Container Ledger GUI running at http://localhost:%s", port)
		log.Printf("Channel: %s | Chaincode: %s | MSP: %s | Peer: %s", channel, chaincode, mspID, gatewayPeer)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-done
	log.Println("Shutting down...")
	if err := httpServer.Shutdown(nil); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

// --- Gateway setup ---

func connectGateway() (*client.Gateway, error) {
	conn, err := newGrpcConnection()
	if err != nil {
		return nil, err
	}

	id, err := newIdentity()
	if err != nil {
		return nil, err
	}

	sign, err := newSign()
	if err != nil {
		return nil, err
	}

	return client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(conn),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
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

func resolveWebDir() string {
	if p := os.Getenv("WEB_DIR"); p != "" {
		return p
	}
	candidates := []string{"web", "../web"}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	if exe, err := os.Executable(); err == nil {
		for _, c := range []string{
			filepath.Join(filepath.Dir(exe), "web"),
			filepath.Join(filepath.Dir(exe), "..", "web"),
		} {
			if fi, err := os.Stat(c); err == nil && fi.IsDir() {
				return c
			}
		}
	}
	return "web"
}

// --- HTTP handlers ---

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	_, err := s.contract.EvaluateTransaction("GetAllContainers")
	if err != nil {
		status, msg := classifyErr(err)
		writeJSON(w, status, map[string]any{"ok": false, "error": msg})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"channel":   s.channel,
		"chaincode": s.chaincode,
		"msp":       mspID,
	})
}

func (s *server) handleListContainers(w http.ResponseWriter, r *http.Request) {
	result, err := s.contract.EvaluateTransaction("GetAllContainers")
	if err != nil {
		writeErr(w, err)
		return
	}
	writeRawJSON(w, result)
}

func (s *server) handleGetContainer(w http.ResponseWriter, r *http.Request) {
	result, err := s.contract.EvaluateTransaction("ReadContainer", r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeRawJSON(w, result)
}

func (s *server) handleRegisterContainer(w http.ResponseWriter, r *http.Request) {
	input, ok := readContainerInput(w, r)
	if !ok {
		return
	}
	_, err := s.contract.SubmitTransaction("RegisterContainer",
		input.ID, input.Type, input.Owner, strconv.Itoa(*input.TareWeight), input.Destination)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"message": fmt.Sprintf("Container %q registered", input.ID), "id": input.ID})
}

func (s *server) handleUpdateContainer(w http.ResponseWriter, r *http.Request) {
	var input containerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}

	id := r.PathValue("id")
	gw := intOrZero(input.GrossWeight)
	tw := intOrZero(input.TareWeight)
	ts := intOrZero(input.TempSetpoint)

	_, err := s.contract.SubmitTransaction("UpdateContainer",
		id, input.Type, input.Status, input.Location, input.Owner,
		input.CargoDesc, gw, tw, input.SealNumber, input.Destination,
		input.LastInspection, ts, input.BillOfLading, input.CustomsStatus)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("Container %q updated", id)})
}

func (s *server) handleDeleteContainer(w http.ResponseWriter, r *http.Request) {
	_, err := s.contract.SubmitTransaction("DeleteContainer", r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("Container %q de-registered", r.PathValue("id"))})
}

func (s *server) handleTransferContainer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		NewOwner string `json:"newOwner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}
	body.NewOwner = strings.TrimSpace(body.NewOwner)
	if body.NewOwner == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "newOwner is required"})
		return
	}

	result, err := s.contract.SubmitTransaction("TransferContainer", r.PathValue("id"), body.NewOwner)
	if err != nil {
		writeErr(w, err)
		return
	}
	previousOwner := strings.Trim(string(bytes.TrimSpace(result)), `"`)
	writeJSON(w, http.StatusOK, map[string]any{
		"message":       fmt.Sprintf("Container %q transferred to %q", r.PathValue("id"), body.NewOwner),
		"previousOwner": previousOwner,
	})
}

func (s *server) handleReportEvent(w http.ResponseWriter, r *http.Request) {
	var input eventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}
	input.Action = strings.TrimSpace(input.Action)
	input.By = strings.TrimSpace(input.By)
	if input.Action == "" || input.By == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "action and by are required"})
		return
	}

	_, err := s.contract.SubmitTransaction("ReportEvent", r.PathValue("id"), input.Action, input.Location, input.By, input.Notes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("Event %q reported for container %q", input.Action, r.PathValue("id"))})
}

func (s *server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	result, err := s.contract.EvaluateTransaction("GetContainerHistory", r.PathValue("id"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeRawJSON(w, result)
}

func (s *server) handleReportDamage(w http.ResponseWriter, r *http.Request) {
	var input damageInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}
	input.DamageType = strings.TrimSpace(input.DamageType)
	input.ReportedBy = strings.TrimSpace(input.ReportedBy)
	if input.DamageType == "" || input.ReportedBy == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "damageType and reportedBy are required"})
		return
	}

	_, err := s.contract.SubmitTransaction("ReportDamage", r.PathValue("id"), input.DamageType, input.Description, input.ReportedBy)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("Damage reported for container %q", r.PathValue("id"))})
}

func (s *server) handleCustomsAction(w http.ResponseWriter, r *http.Request) {
	var input customsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}
	input.CustomsStatus = strings.TrimSpace(input.CustomsStatus)
	input.By = strings.TrimSpace(input.By)
	if input.CustomsStatus == "" || input.By == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "customsStatus and by are required"})
		return
	}

	_, err := s.contract.SubmitTransaction("UpdateCustoms", r.PathValue("id"), input.CustomsStatus, input.By, input.Notes)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("Customs status for %q set to %s", r.PathValue("id"), input.CustomsStatus)})
}

func (s *server) handleSetTemperature(w http.ResponseWriter, r *http.Request) {
	var input tempInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}
	input.By = strings.TrimSpace(input.By)
	if input.By == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "by is required"})
		return
	}

	_, err := s.contract.SubmitTransaction("SetTemperature", r.PathValue("id"), strconv.Itoa(input.TempSetpoint), input.By)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": fmt.Sprintf("Temperature for %q set to %d°C", r.PathValue("id"), input.TempSetpoint)})
}

func (s *server) handleQueryContainers(w http.ResponseWriter, r *http.Request) {
	var input queryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return
	}

	filterMap := map[string]string{}
	if input.Status != "" {
		filterMap["status"] = input.Status
	}
	if input.Type != "" {
		filterMap["type"] = input.Type
	}
	if input.Owner != "" {
		filterMap["owner"] = input.Owner
	}
	if input.Location != "" {
		filterMap["location"] = input.Location
	}

	queryJSON, _ := json.Marshal(filterMap)
	result, err := s.contract.EvaluateTransaction("QueryContainers", string(queryJSON))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeRawJSON(w, result)
}

func (s *server) handleSeed(w http.ResponseWriter, r *http.Request) {
	_, err := s.contract.SubmitTransaction("InitLedger")
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Sample container data loaded (6 containers)"})
}

// --- Input helpers ---

func readContainerInput(w http.ResponseWriter, r *http.Request) (containerInput, bool) {
	var input containerInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Invalid JSON body"})
		return input, false
	}

	input.ID = strings.TrimSpace(input.ID)
	input.Type = strings.TrimSpace(input.Type)
	input.Owner = strings.TrimSpace(input.Owner)
	input.Destination = strings.TrimSpace(input.Destination)

	if input.ID == "" || input.Type == "" || input.Owner == "" || input.Destination == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Fields id, type, owner and destination are required"})
		return input, false
	}
	if input.TareWeight == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Field tareWeight is required"})
		return input, false
	}
	if *input.TareWeight < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "tareWeight must be non-negative"})
		return input, false
	}

	validTypes := map[string]bool{"20GP": true, "40GP": true, "40RH": true, "20OT": true, "40FR": true}
	if !validTypes[input.Type] {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "type must be one of: 20GP, 40GP, 40RH, 20OT, 40FR"})
		return input, false
	}

	return input, true
}

func intOrZero(v *int) string {
	if v == nil {
		return "0"
	}
	return strconv.Itoa(*v)
}

// --- Errors and responses ---

func classifyErr(err error) (int, string) {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "does not exist"):
		return http.StatusNotFound, err.Error()
	case strings.Contains(msg, "already exists"):
		return http.StatusConflict, err.Error()
	case strings.Contains(msg, "unavailable"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "failed to connect"),
		strings.Contains(msg, "deadline"):
		return http.StatusServiceUnavailable, "Cannot reach the Fabric network: " + err.Error()
	default:
		return http.StatusInternalServerError, err.Error()
	}
}

func writeErr(w http.ResponseWriter, err error) {
	status, msg := classifyErr(err)
	writeJSON(w, status, map[string]any{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeRawJSON(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}