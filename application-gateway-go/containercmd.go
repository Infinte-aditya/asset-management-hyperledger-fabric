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
	"strconv"
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
		printUsage()
		return
	}

	clientConnection := newGrpcConnection()
	defer clientConnection.Close()

	id := newIdentity()
	sign := newSign()

	gw, err := client.Connect(
		id,
		client.WithSign(sign),
		client.WithHash(hash.SHA256),
		client.WithClientConnection(clientConnection),
		client.WithEvaluateTimeout(5*time.Second),
		client.WithEndorseTimeout(15*time.Second),
		client.WithSubmitTimeout(5*time.Second),
		client.WithCommitStatusTimeout(1*time.Minute),
	)
	if err != nil {
		panic(err)
	}
	defer gw.Close()

	chaincodeName := "container-cc"
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
	case "RegisterContainer":
		if len(os.Args) != 7 {
			fmt.Println("Usage: go run containercmd.go RegisterContainer <id> <type> <owner> <tareWeight> <destination>")
			return
		}
		registerContainer(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])
	case "ReadContainer":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run containercmd.go ReadContainer <id>")
			return
		}
		readContainer(contract, os.Args[2])
	case "UpdateContainer":
		if len(os.Args) != 16 {
			fmt.Println("Usage: go run containercmd.go UpdateContainer <id> <type> <status> <location> <owner> <cargoDesc> <grossWeight> <tareWeight> <sealNumber> <destination> <lastInspection> <tempSetpoint> <billOfLading> <customsStatus>")
			return
		}
		updateContainer(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6], os.Args[7], os.Args[8], os.Args[9], os.Args[10], os.Args[11], os.Args[12], os.Args[13], os.Args[14], os.Args[15])
	case "TransferContainer":
		if len(os.Args) != 4 {
			fmt.Println("Usage: go run containercmd.go TransferContainer <id> <newOwner>")
			return
		}
		transferContainer(contract, os.Args[2], os.Args[3])
	case "DeleteContainer":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run containercmd.go DeleteContainer <id>")
			return
		}
		deleteContainer(contract, os.Args[2])
	case "ReportEvent":
		if len(os.Args) != 7 {
			fmt.Println("Usage: go run containercmd.go ReportEvent <id> <action> <location> <by> <notes>")
			return
		}
		reportEvent(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6])
	case "ReportDamage":
		if len(os.Args) != 6 {
			fmt.Println("Usage: go run containercmd.go ReportDamage <id> <damageType> <description> <reportedBy>")
			return
		}
		reportDamage(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "UpdateCustoms":
		if len(os.Args) != 6 {
			fmt.Println("Usage: go run containercmd.go UpdateCustoms <id> <customsStatus> <by> <notes>")
			return
		}
		updateCustoms(contract, os.Args[2], os.Args[3], os.Args[4], os.Args[5])
	case "SetTemperature":
		if len(os.Args) != 5 {
			fmt.Println("Usage: go run containercmd.go SetTemperature <id> <tempSetpoint> <by>")
			return
		}
		setTemperature(contract, os.Args[2], os.Args[3], os.Args[4])
	case "GetContainerHistory":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run containercmd.go GetContainerHistory <id>")
			return
		}
		getContainerHistory(contract, os.Args[2])
	case "GetAllContainers":
		getAllContainers(contract)
	case "QueryContainers":
		if len(os.Args) != 3 {
			fmt.Println("Usage: go run containercmd.go QueryContainers <queryJSON>")
			return
		}
		queryContainers(contract, os.Args[2])
	case "InitLedger":
		initLedger(contract)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: go run containercmd.go [command] [args...]")
	fmt.Println("Commands:")
	fmt.Println("  InitLedger")
	fmt.Println("  RegisterContainer <id> <type> <owner> <tareWeight> <destination>")
	fmt.Println("  ReadContainer <id>")
	fmt.Println("  UpdateContainer <id> <type> <status> <location> <owner> <cargoDesc> <grossWeight> <tareWeight> <sealNumber> <destination> <lastInspection> <tempSetpoint> <billOfLading> <customsStatus>")
	fmt.Println("  TransferContainer <id> <newOwner>")
	fmt.Println("  DeleteContainer <id>")
	fmt.Println("  ReportEvent <id> <action> <location> <by> <notes>")
	fmt.Println("  ReportDamage <id> <damageType> <description> <reportedBy>")
	fmt.Println("  UpdateCustoms <id> <customsStatus> <by> <notes>")
	fmt.Println("  SetTemperature <id> <tempSetpoint> <by>")
	fmt.Println("  GetContainerHistory <id>")
	fmt.Println("  GetAllContainers")
	fmt.Println("  QueryContainers <queryJSON>")
}

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

func initLedger(contract *client.Contract) {
	fmt.Println("\n--> Submit Transaction: InitLedger, loads sample container data")
	_, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Println("*** Sample container data loaded successfully")
}

func getAllContainers(contract *client.Contract) {
	fmt.Println("\n--> Evaluate Transaction: GetAllContainers")
	result, err := contract.EvaluateTransaction("GetAllContainers")
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	fmt.Printf("*** Result:%s\n", formatJSON(result))
}

func registerContainer(contract *client.Contract, id string, containerType string, owner string, tareWeight string, destination string) {
	fmt.Printf("\n--> Submit Transaction: RegisterContainer\n")
	_, err := contract.SubmitTransaction("RegisterContainer", id, containerType, owner, tareWeight, destination)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Container %s registered successfully\n", id)
}

func readContainer(contract *client.Contract, id string) {
	fmt.Printf("\n--> Evaluate Transaction: ReadContainer\n")
	result, err := contract.EvaluateTransaction("ReadContainer", id)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	fmt.Printf("*** Result:%s\n", formatJSON(result))
}

func updateContainer(contract *client.Contract, id string, containerType string, status string, location string, owner string, cargoDesc string, grossWeight string, tareWeight string, sealNumber string, destination string, lastInspection string, tempSetpoint string, billOfLading string, customsStatus string) {
	fmt.Printf("\n--> Submit Transaction: UpdateContainer\n")
	_, err := contract.SubmitTransaction("UpdateContainer", id, containerType, status, location, owner, cargoDesc, grossWeight, tareWeight, sealNumber, destination, lastInspection, tempSetpoint, billOfLading, customsStatus)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Container %s updated successfully\n", id)
}

func transferContainer(contract *client.Contract, id string, newOwner string) {
	fmt.Printf("\n--> Submit Transaction: TransferContainer\n")
	result, err := contract.SubmitTransaction("TransferContainer", id, newOwner)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	oldOwner := result
	fmt.Printf("*** Container %s transferred from %s to %s\n", id, string(oldOwner), newOwner)
}

func deleteContainer(contract *client.Contract, id string) {
	fmt.Printf("\n--> Submit Transaction: DeleteContainer\n")
	_, err := contract.SubmitTransaction("DeleteContainer", id)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Container %s deleted successfully\n", id)
}

func reportEvent(contract *client.Contract, id string, action string, location string, by string, notes string) {
	fmt.Printf("\n--> Submit Transaction: ReportEvent\n")
	_, err := contract.SubmitTransaction("ReportEvent", id, action, location, by, notes)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Event %s reported for container %s\n", action, id)
}

func reportDamage(contract *client.Contract, id string, damageType string, description string, reportedBy string) {
	fmt.Printf("\n--> Submit Transaction: ReportDamage\n")
	_, err := contract.SubmitTransaction("ReportDamage", id, damageType, description, reportedBy)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Damage reported for container %s\n", id)
}

func updateCustoms(contract *client.Contract, id string, customsStatus string, by string, notes string) {
	fmt.Printf("\n--> Submit Transaction: UpdateCustoms\n")
	_, err := contract.SubmitTransaction("UpdateCustoms", id, customsStatus, by, notes)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Customs status for %s set to %s\n", id, customsStatus)
}

func setTemperature(contract *client.Contract, id string, tempSetpoint string, by string) {
	fmt.Printf("\n--> Submit Transaction: SetTemperature\n")
	_, err := contract.SubmitTransaction("SetTemperature", id, tempSetpoint, by)
	if err != nil {
		panic(fmt.Errorf("failed to submit transaction: %w", err))
	}
	fmt.Printf("*** Temperature for %s set to %s°C\n", id, tempSetpoint)
}

func getContainerHistory(contract *client.Contract, id string) {
	fmt.Printf("\n--> Evaluate Transaction: GetContainerHistory\n")
	result, err := contract.EvaluateTransaction("GetContainerHistory", id)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	fmt.Printf("*** Journey history for %s:%s\n", id, formatJSON(result))
}

func queryContainers(contract *client.Contract, queryJSON string) {
	fmt.Printf("\n--> Evaluate Transaction: QueryContainers\n")
	result, err := contract.EvaluateTransaction("QueryContainers", queryJSON)
	if err != nil {
		panic(fmt.Errorf("failed to evaluate transaction: %w", err))
	}
	fmt.Printf("*** Query result:%s\n", formatJSON(result))
}

func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err != nil {
		panic(fmt.Errorf("failed to parse JSON: %w", err))
	}
	return prettyJSON.String()
}

func intOrZero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

func init() {
	// Suppress unused function warning for strconv — used indirectly via intOrZero callers
	_ = strconv.Itoa
}