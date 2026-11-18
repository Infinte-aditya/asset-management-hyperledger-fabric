package chaincode_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

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

func TestUpdateContainerPreservesPositionAndJourney(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	// Pre-existing container with a live GPS Position and Journey history.
	existing := &chaincode.Container{
		ID:       "MSCU0001",
		Type:     "40GP",
		Status:   "InTransit",
		Location: "OnBoard:MSC_SVEVA",
		Owner:    "MSC",
		Position: &chaincode.Position{
			Lat: "31.2304", Lon: "121.4737", Timestamp: "2026-09-21T15:00:00Z",
			Speed: "12.5", Heading: "134", Source: "GPS",
		},
		Journey: []chaincode.ContainerEvent{
			{Timestamp: "2026-09-20T08:00:00Z", Action: "GateIn", Location: "CNSHA", By: "Depot-Ops", Notes: "Gate in at Shanghai depot"},
			{Timestamp: "2026-09-21T14:30:00Z", Action: "Loaded", Location: "CNSHA", By: "Terminal-Ops", Notes: "Loaded aboard MSC SVEVA"},
		},
	}
	bytes, err := json.Marshal(existing)
	require.NoError(t, err)

	chaincodeStub.GetStateReturns(bytes, nil)
	cc := chaincode.SmartContract{}
	err = cc.UpdateContainer(transactionContext, "MSCU0001", "40RH", "AtPort", "NLRTM", "Maersk", "Frozen seafood", 26500, 3800, "MK789012", "NLRTM", "2026-09-22", -18, "BL-990", "Hold")
	require.NoError(t, err)

	require.Equal(t, 1, chaincodeStub.PutStateCallCount())
	key, payload := chaincodeStub.PutStateArgsForCall(0)
	require.Equal(t, "MSCU0001", key)

	// Managed fields were overwritten.
	var updated chaincode.Container
	err = json.Unmarshal(payload, &updated)
	require.NoError(t, err)

	require.Equal(t, "40RH", updated.Type)
	require.Equal(t, "AtPort", updated.Status)
	require.Equal(t, "NLRTM", updated.Location)
	require.Equal(t, "Maersk", updated.Owner)
	require.Equal(t, "Frozen seafood", updated.CargoDesc)
	require.Equal(t, 26500, updated.GrossWeight)
	require.Equal(t, 3800, updated.TareWeight)
	require.Equal(t, 26500-3800, updated.NetWeight)
	require.Equal(t, "MK789012", updated.SealNumber)
	require.Equal(t, "NLRTM", updated.Destination)
	require.Equal(t, "2026-09-22", updated.LastInspection)
	require.Equal(t, -18, updated.TempSetpoint)
	require.Equal(t, "BL-990", updated.BillOfLading)
	require.Equal(t, "Hold", updated.CustomsStatus)

	// Position and Journey preserved in the marshaled PutState payload.
	require.NotNil(t, updated.Position)
	require.Equal(t, "31.2304", updated.Position.Lat)
	require.Equal(t, "121.4737", updated.Position.Lon)
	require.Equal(t, "2026-09-21T15:00:00Z", updated.Position.Timestamp)
	require.Equal(t, "12.5", updated.Position.Speed)
	require.Equal(t, "134", updated.Position.Heading)
	require.Equal(t, "GPS", updated.Position.Source)
	require.Len(t, updated.Journey, 2)
	require.Equal(t, "GateIn", updated.Journey[0].Action)
	require.Equal(t, "Loaded", updated.Journey[1].Action)
	require.Contains(t, string(payload), `"Position":{"Lat":"31.2304"`)
	require.Contains(t, string(payload), `"Journey":[`)
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
	require.Equal(t, "OldCo", oldOwner)

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

func TestUpdatePosition(t *testing.T) {
	ts := time.Date(2026, 9, 21, 15, 0, 0, 0, time.UTC)
	compositeKey := "position\u0000MSCU0001\u0000" + fmt.Sprintf("%019d", ts.UnixNano())

	container := &chaincode.Container{ID: "MSCU0001", Status: "Loaded", Location: "CNSHA"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	chaincodeStub.GetStateReturns(bytes, nil)
	chaincodeStub.CreateCompositeKeyReturns(compositeKey, nil)

	cc := chaincode.SmartContract{}
	err = cc.UpdatePosition(transactionContext, "MSCU0001", "31.2304", "121.4737", "2026-09-21T15:00:00Z", "12.5", "134", "GPS", "21.5")
	require.NoError(t, err)

	// PutState called for BOTH the container key and the composite key
	require.Equal(t, 2, chaincodeStub.PutStateCallCount())
	containerKey, containerJSON := chaincodeStub.PutStateArgsForCall(0)
	require.Equal(t, "MSCU0001", containerKey)
	compositePutKey, compositePayload := chaincodeStub.PutStateArgsForCall(1)
	require.Equal(t, compositeKey, compositePutKey)

	// Container state: Position stored with original strings, Loaded -> InTransit with one event
	var updated chaincode.Container
	err = json.Unmarshal(containerJSON, &updated)
	require.NoError(t, err)
	require.Equal(t, "InTransit", updated.Status)
	require.NotNil(t, updated.Position)
	require.Equal(t, "31.2304", updated.Position.Lat)
	require.Equal(t, "121.4737", updated.Position.Lon)
	require.Equal(t, "2026-09-21T15:00:00Z", updated.Position.Timestamp)
	require.Equal(t, "12.5", updated.Position.Speed)
	require.Equal(t, "134", updated.Position.Heading)
	require.Equal(t, "GPS", updated.Position.Source)
	require.Equal(t, "21.5", updated.Position.Temperature)
	require.Len(t, updated.Journey, 1)
	require.Equal(t, "InTransit", updated.Journey[0].Action)
	require.Equal(t, "GPS-Auto", updated.Journey[0].By)
	require.Equal(t, "Auto-updated: vessel underway (GPS fix received)", updated.Journey[0].Notes)

	// Composite-key track record: Position carries the temperature too
	var trackPos chaincode.Position
	err = json.Unmarshal(compositePayload, &trackPos)
	require.NoError(t, err)
	require.Equal(t, "31.2304", trackPos.Lat)
	require.Equal(t, "121.4737", trackPos.Lon)
	require.Equal(t, "2026-09-21T15:00:00Z", trackPos.Timestamp)
	require.Equal(t, "12.5", trackPos.Speed)
	require.Equal(t, "134", trackPos.Heading)
	require.Equal(t, "GPS", trackPos.Source)
	require.Equal(t, "21.5", trackPos.Temperature)

	// Composite key built with object type "position" and id + zero-padded nanos
	objectType, attrs := chaincodeStub.CreateCompositeKeyArgsForCall(0)
	require.Equal(t, "position", objectType)
	require.Equal(t, []string{"MSCU0001", fmt.Sprintf("%019d", ts.UnixNano())}, attrs)

	// Event emitted with the right payload
	require.Equal(t, 1, chaincodeStub.SetEventCallCount())
	eventName, eventPayload := chaincodeStub.SetEventArgsForCall(0)
	require.Equal(t, "UpdatePosition", eventName)
	var event map[string]string
	err = json.Unmarshal(eventPayload, &event)
	require.NoError(t, err)
	require.Equal(t, "MSCU0001", event["id"])
	require.Equal(t, "31.2304", event["lat"])
	require.Equal(t, "121.4737", event["lon"])
	require.Equal(t, "2026-09-21T15:00:00Z", event["timestamp"])
	require.Equal(t, "InTransit", event["status"])
	require.Equal(t, "21.5", event["temperature"])
}

func TestUpdatePositionContainerNotFound(t *testing.T) {
	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	chaincodeStub.GetStateReturns(nil, nil)
	cc := chaincode.SmartContract{}
	err := cc.UpdatePosition(transactionContext, "MSCU0000", "31.2304", "121.4737", "2026-09-21T15:00:00Z", "12.5", "134", "GPS", "21.5")
	require.EqualError(t, err, "container MSCU0000 does not exist")
}

func TestUpdatePositionInvalidInputs(t *testing.T) {
	container := &chaincode.Container{ID: "MSCU0001", Status: "Loaded", Location: "CNSHA"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	cc := chaincode.SmartContract{}
	cases := []struct {
		name      string
		lat       string
		lon       string
		timestamp string
		err       string
	}{
		{name: "invalid lat", lat: "abc", lon: "121.4737", timestamp: "2026-09-21T15:00:00Z", err: `invalid latitude "abc"`},
		{name: "lat out of range", lat: "91", lon: "121.4737", timestamp: "2026-09-21T15:00:00Z", err: "latitude 91 out of range (-90 to 90)"},
		{name: "invalid lon", lat: "31.2304", lon: "abc", timestamp: "2026-09-21T15:00:00Z", err: `invalid longitude "abc"`},
		{name: "lon out of range", lat: "31.2304", lon: "181", timestamp: "2026-09-21T15:00:00Z", err: "longitude 181 out of range (-180 to 180)"},
		{name: "invalid timestamp", lat: "31.2304", lon: "121.4737", timestamp: "not-a-time", err: `invalid timestamp "not-a-time"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			chaincodeStub := &mocks.ChaincodeStub{}
			transactionContext := &mocks.TransactionContext{}
			transactionContext.GetStubReturns(chaincodeStub)
			chaincodeStub.GetStateReturns(bytes, nil)

			err := cc.UpdatePosition(transactionContext, "MSCU0001", tc.lat, tc.lon, tc.timestamp, "12.5", "134", "GPS", "21.5")
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.err)
			require.Equal(t, 0, chaincodeStub.PutStateCallCount())
		})
	}
}

func TestUpdatePositionAlreadyInTransit(t *testing.T) {
	container := &chaincode.Container{
		ID: "MSCU0001", Status: "InTransit", Location: "OnBoard:MSC_SVEVA",
		Journey: []chaincode.ContainerEvent{
			{Timestamp: "2026-09-21T14:30:00Z", Action: "Loaded", Location: "CNSHA", By: "Terminal-Ops", Notes: "Loaded aboard MSC SVEVA"},
		},
	}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)

	chaincodeStub.GetStateReturns(bytes, nil)
	chaincodeStub.CreateCompositeKeyReturns("position\u0000MSCU0001\u00001789898496000000000", nil)

	cc := chaincode.SmartContract{}
	err = cc.UpdatePosition(transactionContext, "MSCU0001", "32.0", "122.0", "2026-09-21T16:00:00Z", "13.2", "140", "GPS", "")
	require.NoError(t, err)

	require.Equal(t, 2, chaincodeStub.PutStateCallCount())
	_, containerJSON := chaincodeStub.PutStateArgsForCall(0)
	var updated chaincode.Container
	err = json.Unmarshal(containerJSON, &updated)
	require.NoError(t, err)
	require.Equal(t, "InTransit", updated.Status)
	// optional temperature may be empty; not required, not validated
	require.Equal(t, "", updated.Position.Temperature)
	// no new transition event for already-InTransit containers
	require.Len(t, updated.Journey, 1)
	require.Equal(t, "Loaded", updated.Journey[0].Action)
}

func TestGetPositionHistory(t *testing.T) {
	positions := []*chaincode.Position{
		{Lat: "31.2304", Lon: "121.4737", Timestamp: "2026-09-21T15:00:00Z", Speed: "12.5", Heading: "134", Source: "GPS"},
		{Lat: "32.0", Lon: "122.0", Timestamp: "2026-09-21T16:00:00Z", Speed: "13.2", Heading: "140", Source: "GPS"},
		{Lat: "33.1", Lon: "123.2", Timestamp: "2026-09-21T17:00:00Z", Speed: "14.0", Heading: "145", Source: "GPS"},
	}
	var positionBytes [][]byte
	for _, p := range positions {
		b, err := json.Marshal(p)
		require.NoError(t, err)
		positionBytes = append(positionBytes, b)
	}

	iterator := &mocks.StateQueryIterator{}
	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, true)
	iterator.HasNextReturnsOnCall(2, true)
	iterator.HasNextReturnsOnCall(3, false)
	iterator.NextReturnsOnCall(0, &queryresult.KV{Value: positionBytes[0]}, nil)
	iterator.NextReturnsOnCall(1, &queryresult.KV{Value: positionBytes[1]}, nil)
	iterator.NextReturnsOnCall(2, &queryresult.KV{Value: positionBytes[2]}, nil)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)
	chaincodeStub.GetStateByPartialCompositeKeyReturns(iterator, nil)

	cc := chaincode.SmartContract{}
	history, err := cc.GetPositionHistory(transactionContext, "MSCU0001")
	require.NoError(t, err)
	require.Len(t, history, 3)
	require.Equal(t, []chaincode.Position{*positions[0], *positions[1], *positions[2]}, history)

	// partial composite key queried with the container id
	objectType, attrs := chaincodeStub.GetStateByPartialCompositeKeyArgsForCall(0)
	require.Equal(t, "position", objectType)
	require.Equal(t, []string{"MSCU0001"}, attrs)

	// iterator closed after iteration
	require.Equal(t, 1, iterator.CloseCallCount())
}

func TestGetAllContainersSkipsCompositeKeys(t *testing.T) {
	container := &chaincode.Container{ID: "MSCU0001"}
	bytes, err := json.Marshal(container)
	require.NoError(t, err)

	// A position composite key contains the \x00 separator and must be skipped,
	// otherwise unmarshalling it as a Container would produce a phantom empty-ID entry.
	iterator := &mocks.StateQueryIterator{}
	iterator.HasNextReturnsOnCall(0, true)
	iterator.HasNextReturnsOnCall(1, true)
	iterator.HasNextReturnsOnCall(2, false)
	iterator.NextReturnsOnCall(0, &queryresult.KV{Key: "MSCU0001", Value: bytes}, nil)
	iterator.NextReturnsOnCall(1, &queryresult.KV{Key: "position\u0000MSCU0001\u00001789898400000000000", Value: []byte(`{"Lat":"31.2304","Lon":"121.4737","Timestamp":"2026-09-21T15:00:00Z"}`)}, nil)

	chaincodeStub := &mocks.ChaincodeStub{}
	transactionContext := &mocks.TransactionContext{}
	transactionContext.GetStubReturns(chaincodeStub)
	chaincodeStub.GetStateByRangeReturns(iterator, nil)

	cc := &chaincode.SmartContract{}
	containers, err := cc.GetAllContainers(transactionContext)
	require.NoError(t, err)
	require.Equal(t, []*chaincode.Container{container}, containers)
}
