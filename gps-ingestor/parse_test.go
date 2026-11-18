package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func TestDetectFormat(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"json array", "[{\"lat\":1}]", "json"},
		{"json object", " { \"lat\": 1 } ", "json"},
		{"nmea", "$GPRMC,080000.00,A,4807.038,N,00427.402,W,12.5,180.0,200926,,*1E", "nmea"},
		{"nmea gn", "$GNGGA,080000.00,4807.038,N,00427.402,W,1,08,1.0,0.0,M,,M,,*60", "nmea"},
		{"gpx", "<gpx></gpx>", "gpx"},
		{"csv", "lat,lon\ntime,x\n1,2\n", "csv"},
		{"empty", "", "csv"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectFormat([]byte(c.input)); got != c.want {
				t.Fatalf("detectFormat(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	recs, errs, err := Parse(fixture(t, "sample.json"), Options{Format: "json"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	r0 := recs[0]
	if r0.Lat != 48.1173 || r0.Lon != -4.4567 {
		t.Fatalf("record 0 coords = %v,%v", r0.Lat, r0.Lon)
	}
	if r0.ContainerID != "MSCU1234567" {
		t.Fatalf("record 0 container = %q", r0.ContainerID)
	}
	if !r0.HasTime || !r0.Time.Equal(time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("record 0 time = %v", r0.Time)
	}
	if r0.Speed != "12.5" || r0.Heading != "180" {
		t.Fatalf("record 0 speed/heading = %q/%q", r0.Speed, r0.Heading)
	}
	if recs[1].ContainerID != "MAEU7890123" {
		t.Fatalf("record 1 container = %q (fuzzy latitude/longitude/containerId)", recs[1].ContainerID)
	}
}

func TestParseCSV(t *testing.T) {
	recs, errs, err := Parse(fixture(t, "sample.csv"), Options{Format: "csv"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].ContainerID != "MSCU1234567" || recs[1].ContainerID != "MAEU7890123" {
		t.Fatalf("csv container ids = %q, %q", recs[0].ContainerID, recs[1].ContainerID)
	}
}

func TestParseNMEA(t *testing.T) {
	recs, errs, err := Parse(fixture(t, "sample.nmea"), Options{Format: "nmea"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	// sample.nmea has RMC+GGA for epoch 08:00 and RMC+GGA for epoch 09:00;
	// each epoch collapses to a single merged record -> 2 records.
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (one per epoch)", len(recs))
	}
	// Epoch 08:00 (RMC authoritative): 48 deg 07.038 N / 004 deg 27.402 W.
	if got := recs[0].Lat; got < 48.117 || got > 48.118 {
		t.Fatalf("epoch 0 lat = %v", got)
	}
	if got := recs[0].Lon; got < -4.4569 || got > -4.4566 {
		t.Fatalf("epoch 0 lon = %v", got)
	}
	// Speed/heading must be retained from RMC, never blanked by the GGA copy.
	if recs[0].Speed != "12.5" || recs[0].Heading != "180.0" {
		t.Fatalf("epoch 0 speed/heading = %q/%q", recs[0].Speed, recs[0].Heading)
	}
	if !recs[0].HasTime || !recs[0].Time.Equal(time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("epoch 0 time = %v", recs[0].Time)
	}
	// Epoch 09:00: N hemisphere positive, W hemisphere negative.
	if !recs[1].HasTime || !recs[1].Time.Equal(time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("epoch 1 time = %v", recs[1].Time)
	}
	if recs[1].Lat <= 0 || recs[1].Lon >= 0 {
		t.Fatalf("expected +lat/-lon for 4812.000 N/00420.000 W, got %v,%v", recs[1].Lat, recs[1].Lon)
	}
	if recs[1].Speed != "11.0" || recs[1].Heading != "175.0" {
		t.Fatalf("epoch 1 speed/heading = %q/%q", recs[1].Speed, recs[1].Heading)
	}
}

func TestParseNMEAMergeAndGGAOnly(t *testing.T) {
	// merge.nmea: RMC+GGA share epoch 10:00; a GGA-only epoch 11:00 follows.
	recs, errs, err := Parse(fixture(t, "merge.nmea"), Options{Format: "nmea", Date: "2026-09-20"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	// Exactly one record per epoch -> 2 records.
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2 (one per epoch)", len(recs))
	}
	// Epoch 10:00 is the merged RMC+GGA; RMC speed/heading retained.
	if !recs[0].Time.Equal(time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("epoch 0 time = %v", recs[0].Time)
	}
	if recs[0].Speed != "10.0" || recs[0].Heading != "170.0" {
		t.Fatalf("epoch 0 speed/heading = %q/%q (RMC must be retained)", recs[0].Speed, recs[0].Heading)
	}
	// Epoch 11:00 is GGA-only; distinct time -> separate record.
	if !recs[1].Time.Equal(time.Date(2026, 9, 20, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("epoch 1 time = %v", recs[1].Time)
	}
	if recs[1].Speed != "" || recs[1].Heading != "" {
		t.Fatalf("epoch 1 (GGA-only) speed/heading should be empty, got %q/%q", recs[1].Speed, recs[1].Heading)
	}
	// Chronological order preserved.
	if recs[0].Time.After(recs[1].Time) {
		t.Fatal("records not in chronological order")
	}
}

func TestParseGGARequiresDate(t *testing.T) {
	ggaOnly := "$GPGGA,080000.00,4807.038,N,00427.402,W,1,08,1.0,0.0,M,,M,,*66\n"
	f, _ := os.CreateTemp(t.TempDir(), "*.nmea")
	f.WriteString(ggaOnly)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "nmea"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected a date-missing error for GGA-only file")
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 records without date, got %d", len(recs))
	}
	// With -date the record parses.
	recs, errs, err = Parse(f.Name(), Options{Format: "nmea", Date: "2026-09-20"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if !recs[0].Time.Equal(time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("GGA time = %v", recs[0].Time)
	}
}

func TestParseMalformedNMEA(t *testing.T) {
	// sample? use malformed.nmea with a bad checksum.
	recs, errs, err := Parse(fixture(t, "malformed.nmea"), Options{Format: "nmea"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected checksum-mismatch error")
	}
	if !strings.Contains(errs[0].Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", errs[0])
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 records from malformed line, got %d", len(recs))
	}
}

func TestParseMalformedCSV(t *testing.T) {
	recs, errs, err := Parse(fixture(t, "malformed.csv"), Options{Format: "csv"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) == 0 {
		t.Fatal("expected out-of-range coordinate errors")
	}
	if len(recs) != 0 {
		t.Fatalf("expected 0 valid records, got %d", len(recs))
	}
}

func TestParseGPX(t *testing.T) {
	recs, errs, err := Parse(fixture(t, "sample.gpx"), Options{Format: "gpx"})
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if !recs[0].HasTime || !recs[0].Time.Equal(time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC)) {
		t.Fatalf("gpx time = %v", recs[0].Time)
	}
}

// FIX D: the GPX track <name> heuristic extracts a single unambiguous
// container id and applies it to every point.
func TestParseGPXAutoContainer(t *testing.T) {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk>
    <name>Container CONT-UHF-98231A Route</name>
    <trkseg>
      <trkpt lat="31.2304" lon="121.4737"><time>2026-11-01T12:00:00Z</time></trkpt>
      <trkpt lat="30.5" lon="124.0"><time>2026-11-03T00:00:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`
	f, _ := os.CreateTemp(t.TempDir(), "*.gpx")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "gpx"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	for _, r := range recs {
		if r.ContainerID != "CONT-UHF-98231A" {
			t.Fatalf("container id = %q, want CONT-UHF-98231A", r.ContainerID)
		}
	}
}

// FIX D: an ambiguous track name (multiple container-like tokens) leaves
// ContainerID empty, requiring -container.
func TestParseGPXAutoContainerAmbiguous(t *testing.T) {
	content := `<gpx version="1.1" creator="test">
  <trk>
    <name>Route MSCU1234567 and MAEU7890123</name>
    <trkseg>
      <trkpt lat="31.2304" lon="121.4737"><time>2026-11-01T12:00:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`
	f, _ := os.CreateTemp(t.TempDir(), "*.gpx")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "gpx"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].ContainerID != "" {
		t.Fatalf("ambiguous name must leave ContainerID empty, got %q", recs[0].ContainerID)
	}
}

// FIX D: -container overrides the auto-detected track-name container id.
func TestGPXAutoContainerOverriddenByCLI(t *testing.T) {
	content := `<gpx version="1.1" creator="test">
  <trk>
    <name>Container CONT-UHF-98231A Route</name>
    <trkseg>
      <trkpt lat="31.2304" lon="121.4737"><time>2026-11-01T12:00:00Z</time></trkpt>
    </trkseg>
  </trk>
</gpx>`
	f, _ := os.CreateTemp(t.TempDir(), "*.gpx")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "gpx"})
	if err != nil || len(errs) != 0 {
		t.Fatalf("parse: err=%v errs=%v", err, errs)
	}
	plan := Plan(recs, Options{Container: "MSCU1234567"}, nil)
	if plan[0].Container != "MSCU1234567" || plan[0].Status != "planned" {
		t.Fatalf("override failed: %+v", plan[0])
	}
}

// FIX D: unit test of the track-name heuristic, covering extraction and the
// ambiguous / no-match cases.
func TestContainerIDFromTrackName(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"Container CONT-UHF-98231A Route", "CONT-UHF-98231A"},
		{"MSCU1234567", "MSCU1234567"},
		{"Route MSCU1234567 and MAEU7890123", ""}, // ambiguous
		{"Container Route", ""},                   // no container-like token
		{"", ""},
	}
	for _, c := range cases {
		if got := containerIDFromTrackName(c.name); got != c.want {
			t.Fatalf("containerIDFromTrackName(%q) = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestPlanContainerOverride(t *testing.T) {
	recs, _, err := Parse(fixture(t, "sample.json"), Options{Format: "json"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	opts := Options{Container: "MSCU1234567", Source: "GPS:sample.json"}
	plan := Plan(recs, opts, nil)
	if len(plan) != 2 {
		t.Fatalf("plan length = %d", len(plan))
	}
	for _, fix := range plan {
		if fix.Container != "MSCU1234567" || fix.Status != "planned" {
			t.Fatalf("fix = %+v", fix)
		}
	}
}

func TestPlanContainerID(t *testing.T) {
	recs, _, err := Parse(fixture(t, "sample.csv"), Options{Format: "csv"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// No -container; records carry their own container ids (valid ISO 6346).
	plan := Plan(recs, Options{}, nil)
	if len(plan) != 2 {
		t.Fatalf("plan length = %d", len(plan))
	}
	if plan[0].Container != "MSCU1234567" || plan[1].Container != "MAEU7890123" {
		t.Fatalf("containers = %q, %q", plan[0].Container, plan[1].Container)
	}
	for _, fix := range plan {
		if fix.Status != "planned" {
			t.Fatalf("expected planned, got %+v", fix)
		}
	}
}

// FIX B: a non-ISO 6346 container id is accepted as-is (with an advisory
// note) rather than hard-rejected.
func TestPlanNonISOContainerIDAccepted(t *testing.T) {
	recs := []RawRecord{
		{Lat: 1, Lon: 2, ContainerID: "CONT-UHF-98231A", RawLine: "row1"},
	}
	plan := Plan(recs, Options{}, nil)
	if len(plan) != 1 || plan[0].Status != "planned" {
		t.Fatalf("non-ISO id should be accepted as-is, got %+v", plan)
	}
	if plan[0].Container != "CONT-UHF-98231A" {
		t.Fatalf("container = %q", plan[0].Container)
	}
}

// FIX B: in live mode the ledger index still validates a non-ISO id.
func TestPlanNonISOContainerIDLedgerCheck(t *testing.T) {
	index := map[string]ContainerInfo{
		"CONT-UHF-98231A": {ID: "CONT-UHF-98231A", Location: "AtPort:USLAX"},
	}
	recs := []RawRecord{
		{Lat: 1, Lon: 2, ContainerID: "CONT-UHF-98231A", RawLine: "r1"}, // on ledger -> planned
		{Lat: 3, Lon: 4, ContainerID: "CONT-UHF-99999X", RawLine: "r2"}, // not on ledger -> unmapped
	}
	plan := Plan(recs, Options{}, index)
	if plan[0].Status != "planned" || plan[0].Container != "CONT-UHF-98231A" {
		t.Fatalf("existing non-ISO id should plan: %+v", plan[0])
	}
	if plan[1].Status != "unmapped" || !strings.Contains(plan[1].Reason, "not found on ledger") {
		t.Fatalf("non-ISO id missing on ledger should be unmapped: %+v", plan[1])
	}
}

// FIX A: nested `location` object supplies lat/lon.
func TestParseJSONNestedLocation(t *testing.T) {
	content := `[{"containerId":"CONT-UHF-98231A","timestamp":"2026-11-01T12:00:00Z","location":{"latitude":31.2304,"longitude":121.4737}}]`
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "json"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if recs[0].Lat != 31.2304 || recs[0].Lon != 121.4737 {
		t.Fatalf("coords = %v,%v", recs[0].Lat, recs[0].Lon)
	}
	if recs[0].ContainerID != "CONT-UHF-98231A" {
		t.Fatalf("container = %q", recs[0].ContainerID)
	}
}

// FIX A: top-level lat/lon take precedence over a nested location object.
func TestParseJSONLocationPrecedence(t *testing.T) {
	content := `[{"lat":10,"lon":20,"location":{"latitude":1,"longitude":2}}]`
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "json"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if recs[0].Lat != 10 || recs[0].Lon != 20 {
		t.Fatalf("top-level coords should win, got %v,%v", recs[0].Lat, recs[0].Lon)
	}
}

// FIX A: `location` given as a "lat,lon" string is parsed best-effort.
func TestParseJSONLocationString(t *testing.T) {
	content := `[{"location":"31.2304,121.4737","containerId":"CONT-UHF-98231A"}]`
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "json"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if recs[0].Lat != 31.2304 || recs[0].Lon != 121.4737 {
		t.Fatalf("string location coords = %v,%v", recs[0].Lat, recs[0].Lon)
	}
}

// FIX C: unconsumed fields are reported once per record as an informational
// note (never an error), and remain visible in -dry-run.
func TestParseJSONIgnoredFields(t *testing.T) {
	var notes []string
	content := `[{"lat":1,"lon":2,"containerId":"X1","temperatureCelsius":4.5,"status":"in transit","txId":"tx_1"}]`
	f, _ := os.CreateTemp(t.TempDir(), "*.json")
	f.WriteString(content)
	f.Close()
	recs, errs, err := Parse(f.Name(), Options{Format: "json", Info: func(format string, args ...any) { notes = append(notes, fmt.Sprintf(format, args...)) }})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("ignored fields must not be an error, got %v", errs)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	if len(notes) != 1 {
		t.Fatalf("expected one info note, got %v", notes)
	}
	if !strings.Contains(notes[0], "ignoring fields [status txId]") {
		t.Fatalf("info note = %q", notes[0])
	}
}

// FIX: temperatureCelsius is now consumed into the record (and passed through
// to Fixes), so it must be populated and no longer reported as an ignored field.
func TestParseJSONTemperature(t *testing.T) {
	var notes []string
	recs, errs, err := Parse(filepath.Join("..", "container_gps_simulation.json"), Options{
		Format: "json",
		Info:   func(format string, args ...any) { notes = append(notes, fmt.Sprintf(format, args...)) },
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 9 {
		t.Fatalf("got %d records, want 9", len(recs))
	}
	// First fix has temperatureCelsius 4.5; last has 5.0.
	if recs[0].Temperature != "4.5" {
		t.Fatalf("record 0 temperature = %q, want 4.5", recs[0].Temperature)
	}
	if recs[8].Temperature != "5.0" {
		t.Fatalf("record 8 temperature = %q, want 5.0", recs[8].Temperature)
	}
	// Fixes must carry Temperature through to the planned JSON output.
	plan := Plan(recs, Options{}, nil)
	if plan[0].Temperature != "4.5" || plan[0].Status != "planned" {
		t.Fatalf("plan[0] temperature = %q (status %q)", plan[0].Temperature, plan[0].Status)
	}
	// temperatureCelsius must no longer be listed among ignored fields.
	for _, n := range notes {
		if strings.Contains(n, "temperatureCelsius") {
			t.Fatalf("temperatureCelsius still reported as ignored: %q", n)
		}
		if strings.Contains(n, "ignoring fields") && !strings.Contains(n, "[status txId]") {
			t.Fatalf("unexpected ignored fields note: %q", n)
		}
	}
}

// FIX: GPX records carry no temperature, so Temperature must be the empty string.
func TestParseGPXTemperatureEmpty(t *testing.T) {
	recs, errs, err := Parse(filepath.Join("..", "container_route.gpx"), Options{Format: "gpx"})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(errs) != 0 {
		t.Fatalf("unexpected parse errors: %v", errs)
	}
	if len(recs) != 9 {
		t.Fatalf("got %d records, want 9", len(recs))
	}
	for _, r := range recs {
		if r.Temperature != "" {
			t.Fatalf("gpx temperature = %q, want empty", r.Temperature)
		}
	}
}

func TestPlanVesselMapping(t *testing.T) {
	// A record carrying a vessel name, mapped against a synthetic ledger index.
	index := map[string]ContainerInfo{
		"MSCU1234567": {ID: "MSCU1234567", Location: "OnBoard:MSC_SVEVA"},
		"MAEU7890123": {ID: "MAEU7890123", Location: "AtPort:NLRTM"},
	}
	recs := []RawRecord{
		{Lat: 1, Lon: 2, Vessel: "MSC SVEVA", RawLine: "r1"}, // '_' and ' ' equivalent
		{Lat: 3, Lon: 4, Vessel: "maersk", RawLine: "r2"},    // no OnBoard match
	}
	plan := Plan(recs, Options{}, index)
	if plan[0].Status != "planned" || plan[0].Container != "MSCU1234567" {
		t.Fatalf("vessel map failed: %+v", plan[0])
	}
	if plan[1].Status != "unmapped" {
		t.Fatalf("expected unmapped for non-OnBoard vessel, got %+v", plan[1])
	}
}

func TestPlanVesselMappingRequiresIndex(t *testing.T) {
	recs := []RawRecord{
		{Lat: 1, Lon: 2, Vessel: "MSC SVEVA", RawLine: "r1"},
	}
	// nil index (dry-run): vessel mapping cannot resolve.
	plan := Plan(recs, Options{}, nil)
	if len(plan) != 1 || plan[0].Status != "unmapped" {
		t.Fatalf("expected unmapped in dry-run, got %+v", plan)
	}
	if !strings.Contains(plan[0].Reason, "ledger index") {
		t.Fatalf("reason = %q", plan[0].Reason)
	}
}

func TestPlanVesselOverride(t *testing.T) {
	index := map[string]ContainerInfo{
		"MSCU1234567": {ID: "MSCU1234567", Location: "OnBoard:MSC_SVEVA"},
	}
	// -vessel overrides the record's own (empty) vessel.
	recs := []RawRecord{{Lat: 1, Lon: 2, RawLine: "r1"}}
	plan := Plan(recs, Options{Vessel: "MSC SVEVA"}, index)
	if len(plan) != 1 || plan[0].Container != "MSCU1234567" || plan[0].Status != "planned" {
		t.Fatalf("vessel override failed: %+v", plan)
	}
}

func TestPlanSkipWithoutMapping(t *testing.T) {
	recs := []RawRecord{{Lat: 1, Lon: 2, RawLine: "r1"}}
	plan := Plan(recs, Options{}, nil)
	if len(plan) != 1 || plan[0].Status != "unmapped" {
		t.Fatalf("expected unmapped, got %+v", plan)
	}
	if !strings.Contains(plan[0].Reason, "no container id or vessel") {
		t.Fatalf("reason = %q", plan[0].Reason)
	}
}

func TestPlanTimestampRFC3339(t *testing.T) {
	recs := []RawRecord{
		{Lat: 1, Lon: 2, Time: time.Date(2026, 9, 20, 8, 0, 0, 0, time.UTC), HasTime: true, ContainerID: "MSCU1234567"},
	}
	plan := Plan(recs, Options{}, nil)
	if plan[0].Timestamp != "2026-09-20T08:00:00Z" {
		t.Fatalf("timestamp = %q", plan[0].Timestamp)
	}
}

func TestPlanRoundsCoordinates(t *testing.T) {
	// NMEA-derived coords carry float noise (e.g. 48.11729999999999);
	// Plan must round them to 7 decimal places so submitted args and the
	// dry-run JSON are clean.
	recs := []RawRecord{
		{Lat: 48.11729999999999, Lon: -4.456700000000001, ContainerID: "MSCU1234567"},
	}
	plan := Plan(recs, Options{}, nil)
	if plan[0].Lat != 48.1173 {
		t.Fatalf("lat not rounded: got %v, want 48.1173", plan[0].Lat)
	}
	if plan[0].Lon != -4.4567 {
		t.Fatalf("lon not rounded: got %v, want -4.4567", plan[0].Lon)
	}
	if got := strconv.FormatFloat(plan[0].Lat, 'f', -1, 64); got != "48.1173" {
		t.Fatalf("formatted lat = %q, want no float noise", got)
	}
}

func TestNeedsIndex(t *testing.T) {
	if !needsIndex([]RawRecord{{ContainerID: "MSCU1234567"}}, Options{}) {
		t.Fatal("container id should require index for ledger cross-check")
	}
	if !needsIndex([]RawRecord{{Vessel: "MSC SVEVA"}}, Options{}) {
		t.Fatal("vessel mapping should require index")
	}
	if needsIndex([]RawRecord{}, Options{Container: "X"}) {
		t.Fatal("container override should not require index")
	}
}

func TestBuildIndex(t *testing.T) {
	result := []byte(`[{"ID":"MSCU1234567","Location":"OnBoard:MSC_SVEVA"},{"ID":"MAEU7890123","Location":"AtPort:NLRTM"}]`)
	idx := buildIndex(result)
	if len(idx) != 2 {
		t.Fatalf("index size = %d", len(idx))
	}
	if idx["MSCU1234567"].Location != "OnBoard:MSC_SVEVA" {
		t.Fatalf("index location = %q", idx["MSCU1234567"].Location)
	}
}

func TestSortFixesByTimestamp(t *testing.T) {
	fixes := []Fix{
		{Container: "A", Timestamp: "2026-09-20T09:00:00Z"},
		{Container: "B", Timestamp: "2026-09-20T08:00:00Z"},
		{Container: "C", Timestamp: "2026-09-20T08:30:00Z"},
	}
	sortFixesByTimestamp(fixes)
	if fixes[0].Container != "B" || fixes[2].Container != "A" {
		t.Fatalf("sort order = %q, %q, %q", fixes[0].Container, fixes[1].Container, fixes[2].Container)
	}
}
