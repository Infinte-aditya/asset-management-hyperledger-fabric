/*
Copyright 2021 IBM All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/hash"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	mspID        = "Org1MSP"
	cryptoPath   = "../../test-network/organizations/peerOrganizations/org1.example.com"
	certPath     = cryptoPath + "/users/User1@org1.example.com/msp/signcerts"
	keyPath      = cryptoPath + "/users/User1@org1.example.com/msp/keystore"
	tlsCertPath  = cryptoPath + "/peers/peer0.org1.example.com/tls/ca.crt"
	peerEndpoint = "dns:///localhost:7051"
	gatewayPeer  = "peer0.org1.example.com"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run simpleAsset.go [command] [args...]")
		fmt.Println("Commands:")
		fmt.Println("  CreateAsset <id> <color> <size> <owner> <appraisedValue>")
		fmt.Println("  ReadAsset <id>")
		fmt.Println("  UpdateAsset <id> <color> <size> <owner> <appraisedValue>")
		fmt.Println("  TransferAsset <id> <newOwner>")
		fmt.Println("  DeleteAsset <id>")
		fmt.Println("  GetAllAssets")
		return
	}

	// The gRPC client connection should be shared by all Gateway connections to this endpoint
	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	// Create a Gateway connection for a specific client identity
	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		// Default timeouts for different gRPC calls
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	// Override default values for chaincode and channel name as they may differ in testing contexts.
	chaincodeName := "basic"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	network := gw.GetNetwork(channelName)
	contract := network.GetContract(chaincodeName)

	command := os.Args[1]
	switch command {
	case "CreateAsset":
		if len(os.Args) != 7 {
			fmt.Println("Usage: go run simpleAsset.go CreateAsset <id> <color> <size> <owner> <appraisedValue>")
			return
		}
		createAsset(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])
	case "ReadAsset":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run simpleAsset.go ReadAsset <id>")
			return
		}
		readAssetByID(contract, os.Args[2])
	case "UpdateAsset":
		if len(os.Args) != 7 {
			fmt.Println("Usage: go run simpleAsset.go UpdateAsset <id> <color> <size> <owner> <appraisedValue>")
			return
		}
		updateAsset(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])
	case "TransferAsset":
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run simpleAsset.go TransferAsset <id> <newOwner>")
			return
		}
		transferAsset(contract, os.Args[2], os.Args[3])
	case "DeleteAsset":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run simpleAsset.go DeleteAsset <id>")
			return
		}
		deleteAsset(contract, os.Args[2])
	case "GetAllAssets":
		getAllAssets(contract)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Usage: go run simpleAsset.go [command] [args...]")
	}
}

// newGrpcConnection creates a gRPC connection to the Gateway server.
func newGrpcConnection() *grpc.ClientConn {
	certificatePEM, err := os.ReadFile(tlsCertPath)
	if err != nil {
		panic(fmt.Errorf("failed to read TLS certifcate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(certificate)
	transportCredentials := credentials.NewClientTLSFromCert(certPool, gatewayPeer)

	connection, err := grpc.NewClient(peerEndpoint, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		panic(fmt.Errorf("failed to create gRPC connection: %w", err))
	}

	return connection
}

// newIdentity creates a client identity for this Gateway connection using an X.509 certificate.
func newIdentity() *identity.X509Identity {
	certificatePEM, err := readFirstFile(certPath)
	if err != nil {
		panic(fmt.Errorf("failed to read certificate file: %w", err))
	}

	certificate, err := identity.CertificateFromPEM(certificatePEM)
	if err != nil {
		panic(err)
	}

	id, err := identity.NewX509Identity(mspID, certificate)
	if err != nil {
		panic(err)
	}

	return id
}

// newSign creates a function that generates a digital signature from a message digest using a private key.
func newSign() identity.Sign {
	privateKeyPEM, err := readFirstFile(keyPath)
	if err != nil {
		panic(fmt.Errorf("failed to read private key file: %w", err))
	}

	privateKey, err := identity.PrivateKeyFromPEM(privateKeyPEM)
	if err != nil {
		panic(err)
	}

	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		panic(err)
	}

	return sign
}

func readFirstFile(dirPath string) ([]byte, error) {
	dir, err := os.Open(dirPath)
	if err != nil {
		return nil, err
	}

	fileNames, err := dir.Readdirnames(1)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(path.Join(dirPath, fileNames[0]))
}

// Evaluate a transaction to query ledger state.
func getAllAssets(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllAssets, function returns all the current assets on the ledger")

	evaluateResult, err := contract.EvaluateTransaction("GetAllAssets")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Submit a transaction synchronously, blocking until it has been committed to the ledger.
func createAsset(contract *client.Contract, id string, color string, size string, owner string, appraisedValue string) {
	fmt.Printf("\n--> Submit Transaction: CreateAsset, creates new asset with ID, Color, Size, Owner and AppraisedValue arguments \n")

	_, err := contract.SubmitTransaction("CreateAsset", id, color, size, owner, appraisedValue)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Evaluate a transaction by assetID to query ledger state.
func readAssetByID(contract *client.Contract, id string) {
	fmt.Printf("\n--> Evaluate Transaction: ReadAsset, function returns asset attributes\n")

	evaluateResult, err := contract.EvaluateTransaction("ReadAsset", id)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	result := formatJSON(evaluateResult)

	fmt.Printf("*** Result:%s\n", result)
}

// Submit transaction to transfer an asset
func transferAsset(contract *client.Contract, id string, newOwner string) {
	fmt.Printf("\n--> Submit Transaction: TransferAsset, updates existing asset owner from current owner to new owner\n")

	_, err := contract.SubmitTransaction("TransferAsset", id, newOwner)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Submit transaction to delete an asset
func deleteAsset(contract *client.Contract, id string) {
	fmt.Printf("\n--> Submit Transaction: DeleteAsset, deletes an asset with given ID\n")

	_, err := contract.SubmitTransaction("DeleteAsset", id)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Submit transaction to update an asset
func updateAsset(contract *client.Contract, id string, color string, size string, owner string, appraisedValue string) {
	fmt.Printf("\n--> Submit Transaction: UpdateAsset, updates existing asset\n")

	_, err := contract.SubmitTransaction("UpdateAsset", id, color, size, owner, appraisedValue)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}

	fmt.Printf("*** Transaction committed successfully\n")
}

// Format JSON data
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}