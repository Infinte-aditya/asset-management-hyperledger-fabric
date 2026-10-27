package chaincode_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/hyperledger/fabric-chaincode-go/v2/shim"
	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
	"github.com/hyperledger/fabric-protos-go-apiv2/ledger/queryresult"
	"github.com/hyperledger/fabric-samples/asset-transfer-basic/chaincode-go/chaincode"
	"github.com/hyperledger/fabric-samples/asset-transfer-basic/chaincode-go/chaincode/mocks"
	"github.com/stretchr/testify/require"
)

//go:generate counterfeiter -o mocks/transaction.go -fake-name TransactionContext . transactionContext
type transactionContext interface {
	contractapi.TransactionContextInterface
}

//go:generate counterfeiter -o mocks/chaincodestub.go -fake-name ChaincodeStub . chaincodeStub
type chaincodeStub interface {
	shim.ChaincodeStubInterface
}

//go:generate counterfeiter -o mocks/statequeryiterator.go -fake-name StateQueryIterator . stateQueryIterator
type stateQueryIterator interface {
	shim.StateQueryIteratorInterface
}

func TestInitLedger(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	cc := chaincode.SmartContract{}
	err := cc.InitLedger(transactionContext)
	require.NoError(t, err)

	chaincodeStub.PutStateReturns(errors.New("failed inserting key"))
	err = cc.InitLedger(transactionContext)
	require.EqualError(t, err, "failed to put to world state: failed inserting key")
}

func TestRegisterContainer(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	cc := chaincode.SmartContract{}
	err := cc.RegisterContainer(transactionContext, "", "", "", 0, "")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns([]byte{}, nil)
	err = cc.RegisterContainer(transactionContext, "MSCU1234567", "", "", 0, "")
	require.EqualError(t, err, "container MSCU1234567 already exists")

	chaincodeStub.GetStateReturns(nil, errors.New("unable to check existence"))
	err = cc.RegisterContainer(transactionContext, "MSCU0001", "", "", 0, "")
	require.EqualError(t, err, "failed to read from world state: unable to check existence")
}

func TestReadContainer(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	expected := &chaincode.Container{ID: "MSCU0001"}
	bytes, err := json.Marshal(expected)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	container, err := cc.ReadContainer(transactionContext, "")
	require.NoError(t, err)
	require.Equal(t, expected, container)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve container"))
	_, err = cc.ReadContainer(transactionContext, "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve container")

	chaincodeStub.GetStateReturns(nil, nil)
	container, err = cc.ReadContainer(transactionContext, "MSCU0000")
	require.EqualError(t, err, "container MSCU0000 does not exist")
	require.Nil(t, container)
}

func TestUpdateContainer(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	expected := &chaincode.Container{ID: "MSCU0001"}
	bytes, err := json.Marshal(expected)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	err = cc.UpdateContainer(transactionContext, "", "", "", "", "", "", 0, 0, "", "", "", 0, "", "")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, nil)
	err = cc.UpdateContainer(transactionContext, "MSCU0000", "", "", "", "", "", 0, 0, "", "", "", 0, "", "")
	require.EqualError(t, err, "container MSCU0000 does not exist")

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	err = cc.UpdateContainer(transactionContext, "MSCU0001", "", "", "", "", "", 0, 0, "", "", "", 0, "", "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestDeleteContainer(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "MSCU0001"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	chaincodeStub.DelStateReturns(nil)
	cc := chaincode.SmartContract{}
	err = cc.DeleteContainer(transactionContext, "")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, nil)
	err = cc.DeleteContainer(transactionContext, "MSCU0000")
	require.EqualError(t, err, "container MSCU0000 does not exist")

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	err = cc.DeleteContainer(transactionContext, "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestTransferContainer(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "MSCU0001", Owner: "OldCo"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	oldOwner, err := cc.TransferContainer(transactionContext, "", "NewCo")
	require.NoError(t, err)
	require.Equal(t, "\"OldCo\"", oldOwner)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	_, err = cc.TransferContainer(transactionContext, "", "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestReportEvent(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "MSCU0001", Status: "Empty"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	err = cc.ReportEvent(transactionContext, "", "GateIn", "USNYC", "Ops", "Gate in")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	err = cc.ReportEvent(transactionContext, "MSCU0000", "", "", "", "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestReportDamage(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "MSCU0001", Status: "Loaded"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	err = cc.ReportDamage(transactionContext, "", "Dent", "30cm dent", "Surveyor")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	err = cc.ReportDamage(transactionContext, "", "", "", "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestUpdateCustoms(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "MSCU0001"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	err = cc.UpdateCustoms(transactionContext, "", "Hold", "CBP", "Random inspection")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	err = cc.UpdateCustoms(transactionContext, "", "", "", "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestSetTemperature(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "REEF0001"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	err = cc.SetTemperature(transactionContext, "", -18, "Ops")
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	err = cc.SetTemperature(transactionContext, "", 0, "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestGetContainerHistory(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	container := &chaincode.Container{ID: "MSCU0001", Journey: []chaincode.ContainerEvent{
		{Action: "GateIn", Location: "CNSHA", By: "Ops"},
	}}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	events, err := cc.GetContainerHistory(transactionContext, "")
	require.NoError(t, err)
	require.Equal(t, 1, len(events))
	require.Equal(t, "GateIn", events[0].Action)

	chaincodeStub.GetStateReturns(nil, errors.New("unable to retrieve"))
	_, err = cc.GetContainerHistory(transactionContext, "")
	require.EqualError(t, err, "failed to read from world state: unable to retrieve")
}

func TestGetAllContainers(t *testing.T) {
	container := &chaincode.Container{ID: "MSCU0001"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	iterator := &mocks.StateQueryIterator{}
	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, false)
	iterator.NextReturns(&queryresult.KV{Value: bytes}, nil)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	chaincodeStub.GetStateByRangeReturns(iterator, nil)
	cc := &chaincode.SmartContract{}
	containers, err := cc.GetAllContainers(transactionContext)
	require.NoError(t, err)
	require.Equal(t, []*chaincode.Container{container}, containers)

	iterator.HasNextReturns(true)
	iterator.NextReturns(nil, errors.New("failed retrieving next item"))
	containers, err = cc.GetAllContainers(transactionContext)
	require.EqualError(t, err, "failed retrieving next item")
	require.Nil(t, containers)

	chaincodeStub.GetStateByRangeReturns(nil, errors.New("failed retrieving all"))
	containers, err = cc.GetAllContainers(transactionContext)
	require.EqualError(t, err, "failed retrieving all")
	require.Nil(t, containers)
}

func TestQueryContainers(t *testing.T) {
	c1 := &chaincode.Container{ID: "MSCU0001", Status: "InTransit", Type: "40GP", Owner: "MSC"}
	c2 := &chaincode.Container{ID: "MAEU0002", Status: "AtPort", Type: "40RH", Owner: "Maersk"}

	bytes1, err := json.Marshal(c1)
	require.NoError(t, err)
	bytes2, err := json.Marshal(c2)
	require.NoError(t, err)

	iterator := &mocks.StateQueryIterator{}
	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, true)
	iterator.HasNextReturnsOnCall(2, false)
	iterator.NextReturnsOnCall(0, &queryresult.KV{Value: bytes1}, nil)
	iterator.NextReturnsOnCall(1, &queryresult.KV{Value: bytes2}, nil)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)
	chaincodeStub.GetStateByRangeReturns(iterator, nil)

	cc := &chaincode.SmartContract{}
	result, err := cc.QueryContainers(transactionContext, `{"status":"InTransit"}`)
	require.NoError(t, err)
	require.Equal(t, 1, len(result))
	require.Equal(t, "MSCU0001", result[0].ID)
}