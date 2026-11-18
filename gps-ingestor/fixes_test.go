package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// FIX 1: sub-second fixes must produce distinct Timestamp strings that
// round-trip through time.Parse(time.RFC3339, ...).
func TestSubSecondTimestampsDistinct(t *testing.T) {
	t1 := time.Date(2026, 9, 20, 8, 0, 0, 500000000, time.UTC) // .5s
	t2 := time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)         // whole second
	recs := []RawRecord{
		{Lat: 1, Lon: 2, Time: t1, HasTime: true, ContainerID: "MSCU1234567"},
		{Lat: 1, Lon: 2, Time: t2, HasTime: true, ContainerID: "MSCU1234567"},
	}
	plan := Plan(recs, Options{}, nil)
	if len(plan) != 2 {
		t.Fatalf("plan length = %d", len(plan))
	}
	if plan[0].Timestamp == plan[1].Timestamp {
		t.Fatalf("sub-second fixes collapsed to same timestamp %q", plan[0].Timestamp)
	}
	if plan[0].Timestamp != "2026-09-20T08:00:00.5Z" {
		t.Fatalf("sub-second precision lost: %q", plan[0].Timestamp)
	}
	if plan[1].Timestamp != "2026-09-20T08:00:00Z" {
		t.Fatalf("whole-second should have no fractional part: %q", plan[1].Timestamp)
	}
	for _, f := range plan {
		if _, err := time.Parse(time.RFC3339, f.Timestamp); err != nil {
			t.Fatalf("timestamp %q fails RFC3339 parse: %v", f.Timestamp, err)
		}
	}
}

// FIX 3: a present but unparsable time key must emit a warning and leave the
// timestamp empty while still planning the record.
func TestUnparsableTimeWarns(t *testing.T) {
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(`[{"lat":48.1,"lon":-4.4,"time":"25/12/2026","container":"MSCU1234567"}]`)
	f.Close()

	recs, errs, err := Parse(f.Name(), Options{Format: "json"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("record should still be planned, got %d", len(recs))
	}
	if recs[0].HasTime {
		t.Fatal("unparsable time should not be parsed")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "unparsable time 25/12/2026") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unparsable time warning, got %v", errs)
	}
}

// FIX 2: runSubmissions must report submitted/failed counts and notFound, and
// the caller can then exit non-zero on any failure.
func TestRunSubmissionsFailureCounts(t *testing.T) {
	fixes := []Fix{
		{Container: "A", Timestamp: "t1"},
		{Container: "B", Timestamp: "t2"},
		{Container: "C", Timestamp: "t3"},
	}
	submit := func(f Fix) error {
		switch f.Container {
		case "A":
			return nil
		case "B":
			return &notFoundError{id: "B"}
		default:
			return fmt.Errorf("boom")
		}
	}
	submitted, failed, notFound := runSubmissions(submit, fixes)
	if submitted != 1 {
		t.Fatalf("submitted = %d, want 1", submitted)
	}
	if failed != 2 {
		t.Fatalf("failed = %d, want 2", failed)
	}
	if !notFound {
		t.Fatal("expected notFound = true")
	}
}

func TestRunSubmissionsAbortsOnConnError(t *testing.T) {
	fixes := []Fix{
		{Container: "A", Timestamp: "t1"},
		{Container: "B", Timestamp: "t2"},
	}
	submit := func(f Fix) error {
		if f.Container == "A" {
			return fmt.Errorf("deadline exceeded")
		}
		return nil
	}
	submitted, failed, notFound := runSubmissions(submit, fixes)
	if submitted != 0 || failed != 1 || notFound {
		t.Fatalf("got submitted=%d failed=%d notFound=%v", submitted, failed, notFound)
	}
}

func TestParseNMESubSecondPreserved(t *testing.T) {
	// RMC at 10:00:00.00 and GGA at 10:00:00.50 are distinct fix times, so they
	// stay separate and the .5s fraction must be preserved in the timestamp.
	content := "$GPRMC,100000.00,A,4815.000,N,00410.000,W,10.0,170.0,200926,,*15\n" +
		"$GPGGA,100000.50,4815.000,N,00410.000,W,1,08,1.0,0.0,M,,M,,*60\n"
	f, _ := os.CreateTemp(t.TempDir(), "*.nmea")
	f.WriteString(content)
	f.Close()

	recs, errs, err := Parse(f.Name(), Options{Format: "nmea"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (distinct sub-second fix times)", len(recs))
	}
	plan := Plan(recs, Options{Container: "MSCU1234567"}, nil)
	if plan[0].Timestamp == plan[1].Timestamp {
		t.Fatalf("sub-second fixes collapsed to %q", plan[0].Timestamp)
	}
	// One timestamp carries the .5 fraction.
	hasFraction := plan[0].Timestamp == "2026-09-20T10:00:00.5Z" || plan[1].Timestamp == "2026-09-20T10:00:00.5Z"
	if !hasFraction {
		t.Fatalf("sub-second fraction lost: %q, %q", plan[0].Timestamp, plan[1].Timestamp)
	}
	for _, f := range plan {
		if _, err := time.Parse(time.RFC3339, f.Timestamp); err != nil {
			t.Fatalf("timestamp %q fails RFC3339 parse: %v", f.Timestamp, err)
		}
	}
}

func TestNotFoundErrorAs(t *testing.T) {
	err := &notFoundError{id: "MSCU0000000"}
	var nf *notFoundError
	if !errors.As(err, &nf) || nf.id != "MSCU0000000" {
		t.Fatalf("errors.As failed for %v", err)
	}
}
