package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/v2/contractapi"
)

type SmartContract struct {
	contractapi.Contract
}

type ContainerEvent struct {
	Timestamp string `json:"Timestamp"`
	Action    string `json:"Action"`
	Location  string `json:"Location"`
	By        string `json:"By"`
	Notes     string `json:"Notes"`
}

type Container struct {
	ID             string           `json:"ID"`
	Type           string           `json:"Type"`
	Status         string           `json:"Status"`
	Location       string           `json:"Location"`
	Owner          string           `json:"Owner"`
	CargoDesc      string           `json:"CargoDesc"`
	GrossWeight    int              `json:"GrossWeight"`
	TareWeight     int              `json:"TareWeight"`
	NetWeight      int              `json:"NetWeight"`
	SealNumber     string           `json:"SealNumber"`
	Destination    string           `json:"Destination"`
	LastInspection string           `json:"LastInspection"`
	TempSetpoint   int              `json:"TempSetpoint"`
	BillOfLading   string           `json:"BillOfLading"`
	CustomsStatus  string           `json:"CustomsStatus"`
	Journey        []ContainerEvent `json:"Journey"`
}

func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	containers := []Container{
		{
			ID: "MSCU1234567", Type: "40GP", Status: "InTransit",
			Location: "OnBoard:MSC_SVEVA", Owner: "MSC",
			CargoDesc: "Electronics & machinery", GrossWeight: 28500, TareWeight: 3700,
			SealNumber: "MS123456", Destination: "USNYC",
			LastInspection: "2026-09-15", CustomsStatus: "Clear",
			Journey: []ContainerEvent{
				{Timestamp: "2026-09-20T08:00:00Z", Action: "GateIn", Location: "CNSHA", By: "Depot-Ops", Notes: "Gate in at Shanghai depot"},
				{Timestamp: "2026-09-21T14:30:00Z", Action: "Loaded", Location: "CNSHA", By: "Terminal-Ops", Notes: "Loaded aboard MSC SVEVA"},
			},
		},
		{
			ID: "MAEU7890123", Type: "40RH", Status: "Loaded",
			Location: "AtPort:NLRTM", Owner: "Maersk",
			CargoDesc: "Frozen seafood", GrossWeight: 26500, TareWeight: 3800,
			SealNumber: "MK789012", Destination: "NLRTM",
			LastInspection: "2026-09-22", TempSetpoint: -18, CustomsStatus: "Clear",
			Journey: []ContainerEvent{
				{Timestamp: "2026-09-25T06:00:00Z", Action: "GateIn", Location: "NLRTM", By: "Depot-Ops", Notes: "Reefer pre-cooled to -18C"},
			},
		},
		{
			ID: "COSU4567890", Type: "20GP", Status: "AtPort",
			Location: "CNSGH", Owner: "COSCO",
			CargoDesc: "Textiles & apparel", GrossWeight: 22000, TareWeight: 2200,
			SealNumber: "CS456789", Destination: "DEHAM",
			LastInspection: "2026-09-18", CustomsStatus: "Clear",
			Journey: []ContainerEvent{
				{Timestamp: "2026-09-19T10:00:00Z", Action: "GateIn", Location: "CNSGH", By: "Depot-Ops", Notes: "Gate in at Shanghai terminal"},
				{Timestamp: "2026-09-20T16:00:00Z", Action: "Loaded", Location: "CNSGH", By: "Terminal-Ops", Notes: "Loaded aboard COSCO PRIDE"},
				{Timestamp: "2026-09-28T07:00:00Z", Action: "Discharged", Location: "DEHAM", By: "Terminal-Ops", Notes: "Discharged at Hamburg"},
			},
		},
		{
			ID: "HLCU3210987", Type: "20OT", Status: "UnderCustoms",
			Location: "USLAX", Owner: "Hapag-Lloyd",
			CargoDesc: "Oversized machinery parts", GrossWeight: 19500, TareWeight: 2300,
			SealNumber: "HL321098", Destination: "USLAX",
			LastInspection: "2026-09-10", CustomsStatus: "Hold",
			Journey: []ContainerEvent{
				{Timestamp: "2026-09-12T09:00:00Z", Action: "GateIn", Location: "DEHAM", By: "Depot-Ops", Notes: "Open top container arrived"},
				{Timestamp: "2026-09-14T11:00:00Z", Action: "Loaded", Location: "DEHAM", By: "Terminal-Ops", Notes: "Loaded aboard HAPAG EXPRESS"},
				{Timestamp: "2026-09-26T13:00:00Z", Action: "Discharged", Location: "USLAX", By: "Terminal-Ops", Notes: "Discharged at Los Angeles"},
				{Timestamp: "2026-09-27T10:00:00Z", Action: "CustomsHold", Location: "USLAX", By: "CBP-Officer", Notes: "Random inspection flagged"},
			},
		},
		{
			ID: "ONEU6543210", Type: "40FR", Status: "Empty",
			Location: "SGSIN", Owner: "ONE",
			CargoDesc: "", GrossWeight: 0, TareWeight: 5200,
			Destination: "SGSIN",
			LastInspection: "2026-09-28", CustomsStatus: "Clear",
			Journey: []ContainerEvent{
				{Timestamp: "2026-09-28T15:00:00Z", Action: "GateOut", Location: "SGSIN", By: "Depot-Ops", Notes: "Discharged empty, returned to depot"},
			},
		},
		{
			ID: "EVERU9876543", Type: "40GP", Status: "Damaged",
			Location: "AEDXB", Owner: "Evergreen",
			CargoDesc: "Furniture & home goods", GrossWeight: 27300, TareWeight: 3700,
			SealNumber: "EV987654", Destination: "AEDXB",
			LastInspection: "2026-09-25", CustomsStatus: "Clear",
			Journey: []ContainerEvent{
				{Timestamp: "2026-09-22T07:00:00Z", Action: "GateIn", Location: "CNSHA", By: "Depot-Ops", Notes: "Gate in at Shanghai"},
				{Timestamp: "2026-09-23T12:00:00Z", Action: "Loaded", Location: "CNSHA", By: "Terminal-Ops", Notes: "Loaded aboard EVERGREEN ACE"},
				{Timestamp: "2026-09-29T06:00:00Z", Action: "Discharged", Location: "AEDXB", By: "Terminal-Ops", Notes: "Discharged at Dubai — dent found on right wall"},
				{Timestamp: "2026-09-29T06:30:00Z", Action: "DamageReported", Location: "AEDXB", By: "Surveyor", Notes: "Structural dent 30cm x 20cm on right wall panel"},
			},
		},
	}

	for _, c := range containers {
		cJSON, err := json.Marshal(c)
		if err != nil {
			return err
		}
		err = ctx.GetStub().PutState(c.ID, cJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state: %w", err)
		}
	}
	return nil
}

func (s *SmartContract) ContainerExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	cJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %w", err)
	}
	return cJSON != nil, nil
}

func (s *SmartContract) RegisterContainer(ctx contractapi.TransactionContextInterface, id string, containerType string, owner string, tareWeight int, destination string) error {
	exists, err := s.ContainerExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("container %s already exists", id)
	}

	container := Container{
		ID:           id,
		Type:         containerType,
		Status:       "Empty",
		Location:     destination,
		Owner:        owner,
		TareWeight:   tareWeight,
		Destination:  destination,
		CustomsStatus: "Clear",
		Journey: []ContainerEvent{
			{Timestamp: time.Now().UTC().Format(time.RFC3339), Action: "Registered", Location: destination, By: owner, Notes: "Container registered on ledger"},
		},
	}

	cJSON, err := json.Marshal(container)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, cJSON)
	if err != nil {
		return err
	}

	err = ctx.GetStub().SetEvent("RegisterContainer", cJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *SmartContract) ReadContainer(ctx contractapi.TransactionContextInterface, id string) (*Container, error) {
	cJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %w", err)
	}
	if cJSON == nil {
		return nil, fmt.Errorf("container %s does not exist", id)
	}

	var container Container
	err = json.Unmarshal(cJSON, &container)
	if err != nil {
		return nil, err
	}
	return &container, nil
}

func (s *SmartContract) UpdateContainer(ctx contractapi.TransactionContextInterface, id string, containerType string, status string, location string, owner string, cargoDesc string, grossWeight int, tareWeight int, sealNumber string, destination string, lastInspection string, tempSetpoint int, billOfLading string, customsStatus string) error {
	exists, err := s.ContainerExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container %s does not exist", id)
	}

	container := Container{
		ID:             id,
		Type:           containerType,
		Status:         status,
		Location:       location,
		Owner:          owner,
		CargoDesc:      cargoDesc,
		GrossWeight:    grossWeight,
		TareWeight:     tareWeight,
		NetWeight:      grossWeight - tareWeight,
		SealNumber:     sealNumber,
		Destination:    destination,
		LastInspection: lastInspection,
		TempSetpoint:   tempSetpoint,
		BillOfLading:   billOfLading,
		CustomsStatus:  customsStatus,
	}

	cJSON, err := json.Marshal(container)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, cJSON)
}

func (s *SmartContract) DeleteContainer(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.ContainerExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("container %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

func (s *SmartContract) TransferContainer(ctx contractapi.TransactionContextInterface, id string, newOwner string) (string, error) {
	container, err := s.ReadContainer(ctx, id)
	if err != nil {
		return "", err
	}

	oldOwner := container.Owner
	container.Owner = newOwner

	container.Journey = append(container.Journey, ContainerEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "Transferred",
		Location:  container.Location,
		By:        oldOwner,
		Notes:     fmt.Sprintf("Ownership transferred from %s to %s", oldOwner, newOwner),
	})

	cJSON, err := json.Marshal(container)
	if err != nil {
		return "", err
	}

	err = ctx.GetStub().PutState(id, cJSON)
	if err != nil {
		return "", err
	}

	eventJSON, _ := json.Marshal(map[string]string{"id": id, "from": oldOwner, "to": newOwner})
	_ = ctx.GetStub().SetEvent("TransferContainer", eventJSON)

	return oldOwner, nil
}

func (s *SmartContract) ReportEvent(ctx contractapi.TransactionContextInterface, id string, action string, location string, by string, notes string) error {
	container, err := s.ReadContainer(ctx, id)
	if err != nil {
		return err
	}

	statusMap := map[string]string{
		"GateIn":         "AtPort",
		"GateOut":        "Empty",
		"Loaded":         "Loaded",
		"Discharged":     "AtPort",
		"DamageReported": "Damaged",
		"Repaired":       "Empty",
		"CustomsHold":    "UnderCustoms",
		"CustomsRelease": "AtPort",
		"InTransit":      "InTransit",
	}

	if newStatus, ok := statusMap[action]; ok {
		container.Status = newStatus
	}
	if location != "" {
		container.Location = location
	}

	container.Journey = append(container.Journey, ContainerEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    action,
		Location:  location,
		By:        by,
		Notes:     notes,
	})

	cJSON, err := json.Marshal(container)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, cJSON)
}

func (s *SmartContract) ReportDamage(ctx contractapi.TransactionContextInterface, id string, damageType string, description string, reportedBy string) error {
	container, err := s.ReadContainer(ctx, id)
	if err != nil {
		return err
	}

	container.Status = "Damaged"

	container.Journey = append(container.Journey, ContainerEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "DamageReported",
		Location:  container.Location,
		By:        reportedBy,
		Notes:     fmt.Sprintf("Damage type: %s — %s", damageType, description),
	})

	cJSON, err := json.Marshal(container)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, cJSON)
	if err != nil {
		return err
	}

	eventJSON, _ := json.Marshal(map[string]string{"id": id, "damageType": damageType})
	_ = ctx.GetStub().SetEvent("ReportDamage", eventJSON)

	return nil
}

func (s *SmartContract) UpdateCustoms(ctx contractapi.TransactionContextInterface, id string, customsStatus string, by string, notes string) error {
	container, err := s.ReadContainer(ctx, id)
	if err != nil {
		return err
	}

	container.CustomsStatus = customsStatus
	if customsStatus == "Hold" {
		container.Status = "UnderCustoms"
	} else if customsStatus == "Clear" || customsStatus == "Release" {
		if container.Status == "UnderCustoms" {
			if container.Location != "" && container.Location != "" {
				container.Status = "AtPort"
			}
		}
	}

	container.Journey = append(container.Journey, ContainerEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "Customs:" + customsStatus,
		Location:  container.Location,
		By:        by,
		Notes:     notes,
	})

	cJSON, err := json.Marshal(container)
	if err != nil {
		return err
	}

	err = ctx.GetStub().PutState(id, cJSON)
	if err != nil {
		return err
	}

	eventJSON, _ := json.Marshal(map[string]string{"id": id, "customsStatus": customsStatus})
	_ = ctx.GetStub().SetEvent("CustomsAction", eventJSON)

	return nil
}

func (s *SmartContract) SetTemperature(ctx contractapi.TransactionContextInterface, id string, tempSetpoint int, by string) error {
	container, err := s.ReadContainer(ctx, id)
	if err != nil {
		return err
	}

	container.TempSetpoint = tempSetpoint

	container.Journey = append(container.Journey, ContainerEvent{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Action:    "TempSet",
		Location:  container.Location,
		By:        by,
		Notes:     fmt.Sprintf("Temperature setpoint changed to %d°C", tempSetpoint),
	})

	cJSON, err := json.Marshal(container)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, cJSON)
}

func (s *SmartContract) GetContainerHistory(ctx contractapi.TransactionContextInterface, id string) ([]ContainerEvent, error) {
	container, err := s.ReadContainer(ctx, id)
	if err != nil {
		return nil, err
	}
	return container.Journey, nil
}

func (s *SmartContract) GetAllContainers(ctx contractapi.TransactionContextInterface) ([]*Container, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var containers []*Container
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var container Container
		err = json.Unmarshal(queryResponse.GetValue(), &container)
		if err != nil {
			return nil, err
		}
		containers = append(containers, &container)
	}
	return containers, nil
}

func (s *SmartContract) QueryContainers(ctx contractapi.TransactionContextInterface, queryJSON string) ([]*Container, error) {
	var filters struct {
		Status  string `json:"status"`
		Type    string `json:"type"`
		Owner   string `json:"owner"`
		Location string `json:"location"`
	}
	err := json.Unmarshal([]byte(queryJSON), &filters)
	if err != nil {
		return nil, fmt.Errorf("invalid query JSON: %w", err)
	}

	all, err := s.GetAllContainers(ctx)
	if err != nil {
		return nil, err
	}

	var result []*Container
	for _, c := range all {
		if filters.Status != "" && c.Status != filters.Status {
			continue
		}
		if filters.Type != "" && c.Type != filters.Type {
			continue
		}
		if filters.Owner != "" && c.Owner != filters.Owner {
			continue
		}
		if filters.Location != "" && c.Location != filters.Location {
			continue
		}
		result = append(result, c)
	}
	return result, nil
}