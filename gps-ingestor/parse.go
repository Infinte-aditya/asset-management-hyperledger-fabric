// Package main implements the gps-ingestor CLI: it parses a GPS/nav file
// (JSON, CSV, NMEA or GPX), maps each reading to a container, and submits
// UpdatePosition transactions to the container chaincode via the Fabric Gateway.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// containerIDRe matches the ISO 6346 container number pattern:
// 4 owner letters followed by 7 digits (e.g. MSCU1234567). It is advisory
// only: a container id that does not match is accepted as-is (with a one-time
// informational note) rather than being rejected.
var containerIDRe = regexp.MustCompile(`^[A-Z]{4}[0-9]{7}$`)

// containerishTokenRe matches a single token that plausibly contains a
// container id: at least two letters, optionally followed by letters/digits/
// hyphens, containing at least one digit. Used only for the advisory GPX
// <trk><name> heuristic; never to reject a record.
var containerishTokenRe = regexp.MustCompile(`^[A-Za-z]{2,}[A-Za-z0-9-]*[0-9][A-Za-z0-9-]*$`)

// trackNameStopWords are words commonly found in GPX track names that must
// not be mistaken for a container id.
var trackNameStopWords = map[string]bool{
	"container": true,
	"route":     true,
	"track":     true,
	"name":      true,
	"id":        true,
	"vessel":    true,
	"ship":      true,
}

// Fix is a normalized GPS reading mapped (or not) to a container.
type Fix struct {
	Container   string  `json:"Container"`
	Lat         float64 `json:"Lat"`
	Lon         float64 `json:"Lon"`
	Timestamp   string  `json:"Timestamp"`
	Speed       string  `json:"Speed"`
	Heading     string  `json:"Heading"`
	Source      string  `json:"Source"`
	Temperature string  `json:"Temperature"`
	Status      string  `json:"Status,omitempty"` // "planned" | "unmapped"
	Reason      string  `json:"Reason,omitempty"`
	RawLine     string  `json:"RawLine,omitempty"`
	ContainerID string  `json:"ContainerID,omitempty"`
	Vessel      string  `json:"Vessel,omitempty"`
}

// RawRecord is a parsed GPS reading before container mapping.
type RawRecord struct {
	Lat         float64
	Lon         float64
	Time        time.Time
	HasTime     bool
	Speed       string
	Heading     string
	Temperature string
	ContainerID string
	Vessel      string
	RawLine     string
	Warn        string // non-fatal per-record warning (e.g. unparsable time)
}

// ContainerInfo is a container as returned by GetAllContainers.
type ContainerInfo struct {
	ID       string
	Location string
}

// Options carries CLI-derived settings used while parsing and mapping.
type Options struct {
	Format    string // auto | csv | json | nmea | gpx
	Container string // -container override (applies to all records)
	Vessel    string // -vessel override for vessel mapping
	Date      string // YYYY-MM-DD needed for GGA-only files
	Delimiter rune   // CSV field delimiter
	Source    string // "GPS:<base name of file>"

	// Info, when set, receives informational notes (ignored fields, advisory
	// container-id checks). When nil the notes are written to stderr so they
	// remain visible in -dry-run.
	Info func(format string, args ...any)
}

// detectFormat sniffs the file format from its leading bytes.
func detectFormat(data []byte) string {
	t := strings.TrimLeft(string(data), " \t\r\n")
	if t == "" {
		return "csv"
	}
	switch t[0] {
	case '{', '[':
		return "json"
	case '<':
		return "gpx"
	}
	first := strings.SplitN(t, "\n", 2)[0]
	if strings.HasPrefix(first, "$GP") || strings.HasPrefix(first, "$GN") {
		return "nmea"
	}
	return "csv"
}

// Parse reads the file at path and returns parsed raw records plus any
// per-line errors. opts.Format of "auto" (or empty) triggers detection.
func Parse(path string, opts Options) ([]RawRecord, []error, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	format := opts.Format
	if format == "" || format == "auto" {
		format = detectFormat(data)
	}
	opts.Source = "GPS:" + filepath.Base(path)

	switch format {
	case "json":
		recs, errs := parseJSON(data, opts)
		return recs, errs, nil
	case "csv":
		recs, errs := parseCSV(data, opts)
		return recs, errs, nil
	case "nmea":
		recs, errs := parseNMEA(data, opts)
		return recs, errs, nil
	case "gpx":
		recs, errs := parseGPX(data, opts)
		return recs, errs, nil
	default:
		return nil, nil, fmt.Errorf("unknown format %q", format)
	}
}

// --- JSON ---

func parseJSON(data []byte, opts Options) ([]RawRecord, []error) {
	// UseNumber preserves the literal form of numeric values (e.g. "5.0" stays
	// "5.0" rather than collapsing to "5") so stringified fields like
	// temperature round-trip exactly as authored.
	decode := func(b []byte, v any) error {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.UseNumber()
		return dec.Decode(v)
	}
	var objs []map[string]any
	if err := decode(data, &objs); err != nil {
		var one map[string]any
		if err2 := decode(data, &one); err2 != nil {
			return nil, []error{fmt.Errorf("json: %v", err)}
		}
		objs = []map[string]any{one}
	}
	var records []RawRecord
	var errs []error
	for _, obj := range objs {
		ignored := ignoredFields(obj)
		rec, werr := extractRecord(lowerMap(obj), "json record", opts)
		noteIgnoredFields(opts, "json record", ignored)
		if werr != nil {
			errs = append(errs, werr)
			continue
		}
		if rec.Warn != "" {
			errs = append(errs, fmt.Errorf("%s", rec.Warn))
		}
		records = append(records, rec)
	}
	return records, errs
}

// lowerMap lowercases every top-level key so fuzzy matching is case-insensitive.
func lowerMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[strings.ToLower(k)] = v
	}
	return out
}

// --- CSV ---

func parseCSV(data []byte, opts Options) ([]RawRecord, []error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	r.Comma = opts.Delimiter
	if r.Comma == 0 {
		r.Comma = ','
	}
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, []error{fmt.Errorf("csv: no header: %v", err)}
	}
	names := make([]string, len(header))
	rawNames := make([]string, len(header))
	for i, h := range header {
		names[i] = strings.ToLower(strings.TrimSpace(h))
		rawNames[i] = strings.TrimSpace(h)
	}

	var records []RawRecord
	var errs []error
	line := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		line++
		if err != nil {
			errs = append(errs, fmt.Errorf("csv line %d: %v", line, err))
			continue
		}
		m := make(map[string]any, len(names))
		rawMap := make(map[string]any, len(rawNames))
		for i, name := range names {
			if i < len(row) {
				m[name] = row[i]
				rawMap[rawNames[i]] = row[i]
			}
		}
		ignored := ignoredFields(rawMap)
		rec, werr := extractRecord(m, fmt.Sprintf("csv line %d", line), opts)
		noteIgnoredFields(opts, fmt.Sprintf("csv line %d", line), ignored)
		if werr != nil {
			errs = append(errs, werr)
			continue
		}
		if rec.Warn != "" {
			errs = append(errs, fmt.Errorf("%s", rec.Warn))
		}
		records = append(records, rec)
	}
	return records, errs
}

// --- NMEA ---

func parseNMEA(data []byte, opts Options) ([]RawRecord, []error) {
	lines := strings.Split(string(data), "\n")
	var errs []error

	// First pass: find a date (ddmmyy) from the first RMC sentence so that
	// GGA sentences (which carry time only) can form a full timestamp.
	fileDate := ""
	for _, l := range lines {
		s := strings.TrimSpace(l)
		if !isRMC(s) {
			continue
		}
		f := splitNMEA(s)
		if len(f) >= 10 {
			fileDate = strings.TrimSpace(f[9])
		}
		break
	}

	var parsed []RawRecord
	for _, l := range lines {
		s := strings.TrimSpace(l)
		if s == "" || !(strings.HasPrefix(s, "$GP") || strings.HasPrefix(s, "$GN")) {
			continue
		}
		if err := checkNMEA(s); err != nil {
			errs = append(errs, err)
			continue
		}
		f := splitNMEA(s)
		switch {
		case isRMC(s):
			rec, err := parseRMC(f, opts)
			if err != nil {
				errs = append(errs, fmt.Errorf("nmea: %s", err))
				continue
			}
			parsed = append(parsed, rec)
		case isGGA(s):
			rec, err := parseGGA(f, fileDate, opts)
			if err != nil {
				errs = append(errs, fmt.Errorf("nmea: %s", err))
				continue
			}
			parsed = append(parsed, rec)
		}
	}
	// $GPRMC and $GPGGA sentences share the same fix time within an epoch;
	// collapse each epoch into a single record so identical (containerID,
	// fixTimeNanos) ledger keys do not collide. RMC is authoritative.
	return mergeNMEARecords(parsed), errs
}

// mergeNMEARecords collapses NMEA records that share the same fix time into a
// single record per epoch. RMC is the authoritative base (it carries speed,
// heading and the date); empty Speed/Heading are filled from the other
// sentence, but a non-empty value is never overwritten by an empty one.
// Records without a time, or with distinct times, stay separate. The merged
// timed records are returned in chronological order.
func mergeNMEARecords(parsed []RawRecord) []RawRecord {
	idx := map[time.Time]int{}
	var timed []RawRecord
	var untimed []RawRecord
	for _, r := range parsed {
		if !r.HasTime {
			untimed = append(untimed, r)
			continue
		}
		if i, ok := idx[r.Time]; ok {
			timed[i] = mergeFixFields(timed[i], r)
			continue
		}
		idx[r.Time] = len(timed)
		timed = append(timed, r)
	}
	sort.SliceStable(timed, func(i, j int) bool {
		return timed[i].Time.Before(timed[j].Time)
	})
	return append(timed, untimed...)
}

// mergeFixFields fills empty Speed/Heading on a from b without ever letting an
// empty value on b overwrite a non-empty value on a.
func mergeFixFields(a, b RawRecord) RawRecord {
	a.Speed = firstNonEmpty(a.Speed, b.Speed)
	a.Heading = firstNonEmpty(a.Heading, b.Heading)
	return a
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func isRMC(s string) bool {
	return strings.HasPrefix(s, "$GPRMC") || strings.HasPrefix(s, "$GNRMC")
}

func isGGA(s string) bool {
	return strings.HasPrefix(s, "$GPGGA") || strings.HasPrefix(s, "$GNGGA")
}

func splitNMEA(s string) []string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '*'); i >= 0 {
		s = s[:i]
	}
	return strings.Split(s, ",")
}

func checkNMEA(s string) error {
	star := strings.IndexByte(s, '*')
	if star < 0 {
		return nil // no checksum present; skip validation
	}
	var x byte
	for i := 1; i < star; i++ {
		x ^= s[i]
	}
	computed := fmt.Sprintf("%02X", x)
	file := strings.ToUpper(s[star+1:])
	if file != computed {
		return fmt.Errorf("line %q: checksum mismatch (computed %s, line has %s)", s, computed, file)
	}
	return nil
}

func parseRMC(f []string, opts Options) (RawRecord, error) {
	var rec RawRecord
	if len(f) < 10 {
		return rec, fmt.Errorf("malformed RMC sentence")
	}
	if strings.TrimSpace(f[2]) != "A" {
		return rec, fmt.Errorf("RMC data invalid (status %q)", f[2])
	}
	lat, err := parseNMEACoord(f[3], f[4])
	if err != nil {
		return rec, err
	}
	lon, err := parseNMEACoord(f[5], f[6])
	if err != nil {
		return rec, err
	}
	rec.Lat, rec.Lon = lat, lon
	t := strings.TrimSpace(f[1])
	d := strings.TrimSpace(f[9])
	if t != "" && d != "" {
		if tm, ok := nmeaTime(t, d); ok {
			rec.Time, rec.HasTime = tm, true
		}
	}
	rec.Speed = strings.TrimSpace(f[7])
	rec.Heading = strings.TrimSpace(f[8])
	return rec, nil
}

func parseGGA(f []string, fileDate string, opts Options) (RawRecord, error) {
	var rec RawRecord
	if len(f) < 6 {
		return rec, fmt.Errorf("malformed GGA sentence")
	}
	lat, err := parseNMEACoord(f[2], f[3])
	if err != nil {
		return rec, err
	}
	lon, err := parseNMEACoord(f[4], f[5])
	if err != nil {
		return rec, err
	}
	rec.Lat, rec.Lon = lat, lon
	t := strings.TrimSpace(f[1])
	d := fileDate
	if d == "" {
		d = strings.TrimSpace(opts.Date)
	}
	if d == "" {
		return rec, fmt.Errorf("GGA has no date; provide -date YYYY-MM-DD")
	}
	if tm, ok := nmeaTimeWithDate(t, d); ok {
		rec.Time, rec.HasTime = tm, true
	}
	return rec, nil
}

func parseNMEACoord(raw, hemi string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, fmt.Errorf("missing coordinate")
	}
	hemi = strings.ToUpper(strings.TrimSpace(hemi))
	degMin, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("bad coordinate %q", raw)
	}
	deg := math.Floor(degMin / 100)
	min := degMin - deg*100
	val := deg + min/60.0
	if hemi == "S" || hemi == "W" {
		val = -val
	}
	return val, nil
}

func nmeaTime(t, ddmmyy string) (time.Time, bool) {
	hh, mm, sec, nanos := parseHMS(t)
	if len(ddmmyy) != 6 {
		return time.Time{}, false
	}
	dd := atoi(ddmmyy[0:2])
	mo := atoi(ddmmyy[2:4])
	yy := 2000 + atoi(ddmmyy[4:6])
	if mo < 1 || mo > 12 || dd < 1 || dd > 31 {
		return time.Time{}, false
	}
	return time.Date(yy, time.Month(mo), dd, hh, mm, sec, nanos, time.UTC), true
}

func nmeaTimeWithDate(t, dateStr string) (time.Time, bool) {
	if len(dateStr) == 6 {
		return nmeaTime(t, dateStr)
	}
	hh, mm, sec, nanos := parseHMS(t)
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(d.Year(), d.Month(), d.Day(), hh, mm, sec, nanos, time.UTC), true
}

func parseHMS(t string) (hh, mm, sec, nanos int) {
	parts := strings.SplitN(t, ".", 2)
	whole := parts[0]
	hh, mm, sec = 0, 0, 0
	if len(whole) >= 6 {
		hh, mm, sec = atoi(whole[0:2]), atoi(whole[2:4]), atoi(whole[4:6])
	} else if len(whole) >= 4 {
		hh, mm = atoi(whole[0:2]), atoi(whole[2:4])
	}
	if len(parts) == 2 && parts[1] != "" {
		frac := parts[1]
		for len(frac) < 9 {
			frac += "0"
		}
		if len(frac) > 9 {
			frac = frac[:9]
		}
		nanos = atoi(frac)
	}
	return hh, mm, sec, nanos
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// --- GPX ---

type gpx struct {
	XMLName xml.Name `xml:"gpx"`
	Trk     struct {
		Name   string `xml:"name"`
		Trkseg []struct {
			Trkpt []struct {
				Lat  string `xml:"lat,attr"`
				Lon  string `xml:"lon,attr"`
				Time string `xml:"time"`
			} `xml:"trkpt"`
		} `xml:"trkseg"`
	} `xml:"trk"`
}

func parseGPX(data []byte, opts Options) ([]RawRecord, []error) {
	var g gpx
	if err := xml.Unmarshal(data, &g); err != nil {
		return nil, []error{fmt.Errorf("gpx: %v", err)}
	}
	// Best-effort container detection from the track <name>: used only when the
	// name contains exactly one unambiguous container-like token. -container
	// always overrides it (resolveContainer priority 1).
	autoContainer := ""
	if opts.Container == "" {
		autoContainer = containerIDFromTrackName(g.Trk.Name)
	}
	var records []RawRecord
	var errs []error
	var pts []struct {
		Lat  string `xml:"lat,attr"`
		Lon  string `xml:"lon,attr"`
		Time string `xml:"time"`
	}
	for _, seg := range g.Trk.Trkseg {
		pts = append(pts, seg.Trkpt...)
	}
	for i, p := range pts {
		lat, err := strconv.ParseFloat(strings.TrimSpace(p.Lat), 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("gpx trkpt %d: bad latitude %q", i, p.Lat))
			continue
		}
		lon, err := strconv.ParseFloat(strings.TrimSpace(p.Lon), 64)
		if err != nil {
			errs = append(errs, fmt.Errorf("gpx trkpt %d: bad longitude %q", i, p.Lon))
			continue
		}
		rec := RawRecord{Lat: lat, Lon: lon, RawLine: fmt.Sprintf("gpx trkpt %d", i)}
		if autoContainer != "" {
			rec.ContainerID = autoContainer
		}
		if t := strings.TrimSpace(p.Time); t != "" {
			if tm, err := time.Parse(time.RFC3339, t); err == nil {
				rec.Time, rec.HasTime = tm.UTC(), true
			}
		}
		records = append(records, rec)
	}
	return records, errs
}

// containerIDFromTrackName scans a GPX <trk><name> string for a single
// unambiguous container-id-like token. It returns the uppercased id, or ""
// when there is no match or more than one candidate (callers must then rely
// on -container). Common words such as "Container" and "Route" are ignored.
func containerIDFromTrackName(name string) string {
	tokens := strings.FieldsFunc(name, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-')
	})
	var candidates []string
	for _, tok := range tokens {
		if !containerishTokenRe.MatchString(tok) {
			continue
		}
		if trackNameStopWords[strings.ToLower(tok)] {
			continue
		}
		candidates = append(candidates, tok)
	}
	if len(candidates) != 1 {
		return ""
	}
	return normalizeContainerID(candidates[0])
}

// --- Generic field extraction ---

// fixedFieldKeys are the recognized (consumed) record field names, lowercase.
// Any other key present in a JSON/CSV record is reported as an ignored field.
var fixedFieldKeys = map[string]bool{
	"lat": true, "latitude": true,
	"lon": true, "lng": true, "longitude": true,
	"time": true, "timestamp": true, "ts": true,
	"speed": true, "sog": true,
	"heading": true, "cog": true, "course": true,
	"temperature": true, "temperaturecelsius": true, "temp": true, "tempc": true,
	"container": true, "containerid": true, "container_id": true, "id": true,
	"vessel": true, "mmsi": true, "ship": true,
	"location": true,
}

// ignoredFields returns the keys of m that are not among the recognized GPS
// fields (matched case-insensitively). Keys are returned sorted for stable
// output.
func ignoredFields(m map[string]any) []string {
	var out []string
	for k := range m {
		if !fixedFieldKeys[strings.ToLower(k)] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// noteIgnoredFields reports, once per record, the fields that were present but
// not consumed. Informational only; never an error.
func noteIgnoredFields(opts Options, rawLine string, fields []string) {
	if len(fields) == 0 {
		return
	}
	infoNote(opts, "%s: ignoring fields %v", rawLine, fields)
}

// infoNote reports an informational (non-error) note. If opts.Info is set it
// receives the note; otherwise the note is written to stderr with an "info:"
// prefix so it stays visible in -dry-run.
func infoNote(opts Options, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if opts.Info != nil {
		opts.Info(msg)
		return
	}
	fmt.Fprintf(os.Stderr, "info: %s\n", msg)
}

// locationCoords extracts latitude/longitude from a `location` value. It
// accepts a nested object carrying lat/latitude and lon/lng/longitude (keys
// matched case-insensitively) or a "lat,lon" string. Returns "" for any part
// that is missing; callers decide how to fall back or error.
func locationCoords(v any) (lat, lon string) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			switch strings.ToLower(k) {
			case "lat", "latitude":
				if lat == "" {
					lat = strings.TrimSpace(stringVal(val))
				}
			case "lon", "lng", "longitude":
				if lon == "" {
					lon = strings.TrimSpace(stringVal(val))
				}
			}
		}
	case string:
		s := strings.TrimSpace(x)
		if i := strings.IndexByte(s, ','); i > 0 {
			lat = strings.TrimSpace(s[:i])
			lon = strings.TrimSpace(s[i+1:])
		}
	}
	return lat, lon
}

func extractRecord(m map[string]any, rawLine string, opts Options) (RawRecord, error) {
	var rec RawRecord
	rec.RawLine = rawLine

	// Coordinate extraction: top-level explicit lat/lon keys win when present;
	// otherwise a nested `location` object (or "lat,lon" string) is used.
	latStr := firstString(m, "lat", "latitude")
	lonStr := firstString(m, "lon", "lng", "longitude")
	if latStr == "" || lonStr == "" {
		if loc, ok := m["location"]; ok {
			llat, llon := locationCoords(loc)
			if latStr == "" {
				latStr = llat
			}
			if lonStr == "" {
				lonStr = llon
			}
		}
	}
	if latStr == "" {
		return rec, fmt.Errorf("%s: missing latitude", rawLine)
	}
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return rec, fmt.Errorf("%s: bad latitude %q", rawLine, latStr)
	}
	if lat < -90 || lat > 90 {
		return rec, fmt.Errorf("%s: latitude %v out of range", rawLine, lat)
	}

	if lonStr == "" {
		return rec, fmt.Errorf("%s: missing longitude", rawLine)
	}
	lon, err := strconv.ParseFloat(lonStr, 64)
	if err != nil {
		return rec, fmt.Errorf("%s: bad longitude %q", rawLine, lonStr)
	}
	if lon < -180 || lon > 180 {
		return rec, fmt.Errorf("%s: longitude %v out of range", rawLine, lon)
	}
	rec.Lat, rec.Lon = lat, lon

	if v, ok := firstAny(m, "time", "timestamp", "ts"); ok {
		if tm, has := parseTime(v); has {
			rec.Time, rec.HasTime = tm, true
		} else {
			rec.Warn = fmt.Sprintf("%s: unparsable time %s, timestamp left empty", rawLine, stringVal(v))
		}
	}
	rec.Speed = firstString(m, "speed", "sog")
	rec.Heading = firstString(m, "heading", "cog", "course")
	rec.Temperature = firstString(m, "temperaturecelsius", "temperature", "temp", "tempc")
	rec.ContainerID = normalizeContainerID(firstString(m, "container", "containerid", "container_id", "id"))
	rec.Vessel = strings.TrimSpace(firstString(m, "vessel", "mmsi", "ship"))
	return rec, nil
}

func normalizeContainerID(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

// roundCoord rounds a coordinate to 7 decimal places (~1 cm) to avoid float
// noise (e.g. 48.11729999999999) in submitted args and dry-run output.
func roundCoord(v float64) float64 {
	return math.Round(v*1e7) / 1e7
}

// formatTimestamp renders a fix time as RFC3339. RFC3339Nano preserves
// sub-second precision (dropping trailing zeros) so distinct fixes within the
// same second keep distinct chaincode track keys; whole-second times render
// without a fractional part. The result is still valid RFC3339 (Go's
// time.Parse(time.RFC3339, ...) accepts a fractional second).
func formatTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s := strings.TrimSpace(stringVal(v)); s != "" {
				return s
			}
		}
	}
	return ""
}

func firstAny(m map[string]any, keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, true
		}
	}
	return nil, false
}

func stringVal(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	case float64:
		if x == math.Trunc(x) {
			return strconv.FormatFloat(x, 'f', -1, 64)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(v)
	}
}

func parseTime(v any) (time.Time, bool) {
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
			if tm, err := time.Parse(layout, s); err == nil {
				return tm.UTC(), true
			}
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return unixFrom(f), true
		}
	case float64:
		return unixFrom(x), true
	case json.Number:
		if f, err := x.Float64(); err == nil {
			return unixFrom(f), true
		}
	}
	return time.Time{}, false
}

func unixFrom(v float64) time.Time {
	if v > 1e12 {
		return time.UnixMilli(int64(v)).UTC()
	}
	return time.Unix(int64(v), 0).UTC()
}

// --- Mapping ---

// Plan maps parsed records to containers and returns the planned fixes.
// index (map of ledger containers) may be nil in dry-run mode.
func Plan(records []RawRecord, opts Options, index map[string]ContainerInfo) []Fix {
	fixes := make([]Fix, 0, len(records))
	for _, r := range records {
		f := Fix{
			Lat:         roundCoord(r.Lat),
			Lon:         roundCoord(r.Lon),
			Speed:       r.Speed,
			Heading:     r.Heading,
			Temperature: r.Temperature,
			Source:      opts.Source,
			ContainerID: r.ContainerID,
			Vessel:      r.Vessel,
			RawLine:     r.RawLine,
		}
		if r.HasTime {
			f.Timestamp = formatTimestamp(r.Time)
		}
		container, reason, ok := resolveContainer(r, opts, index)
		if !ok {
			f.Status = "unmapped"
			f.Reason = reason
			fixes = append(fixes, f)
			continue
		}
		f.Container = container
		f.Status = "planned"
		fixes = append(fixes, f)
	}
	return fixes
}

// resolveContainer applies the mapping priority (never guesses).
func resolveContainer(r RawRecord, opts Options, index map[string]ContainerInfo) (string, string, bool) {
	// 1) -container applies to all records.
	if opts.Container != "" {
		return opts.Container, "", true
	}
	// 2) record carries a container id. Any non-empty id is accepted as-is:
	// a non-ISO 6346 id only earns a one-time informational note, never a hard
	// rejection. In live mode the ledger-existence check still gates it.
	if r.ContainerID != "" {
		if !containerIDRe.MatchString(r.ContainerID) {
			noteNonISOCountainerID(opts, r.ContainerID)
		}
		if index != nil {
			if _, ok := index[r.ContainerID]; !ok {
				return "", fmt.Sprintf("container %q not found on ledger", r.ContainerID), false
			}
		}
		return r.ContainerID, "", true
	}
	// 3) vessel mapping.
	vessel := r.Vessel
	if opts.Vessel != "" {
		vessel = opts.Vessel
	}
	if vessel != "" {
		if index == nil {
			return "", fmt.Sprintf("vessel %q requires ledger index (live mode)", vessel), false
		}
		if id, ok := matchVessel(index, vessel); ok {
			return id, "", true
		}
		return "", fmt.Sprintf("no unique container aboard vessel %q", vessel), false
	}
	// 4) otherwise skip loudly.
	return "", "no container id or vessel in record", false
}

// nonISOContainerNotes tracks container ids for which the advisory non-ISO
// note was already emitted, so the message appears exactly once per id.
var nonISOContainerNotes = map[string]bool{}

// noteNonISOCountainerID emits a one-time informational note when a container
// id does not match the ISO 6346 pattern. The id is still accepted as-is; the
// note is purely advisory.
func noteNonISOCountainerID(opts Options, id string) {
	if nonISOContainerNotes[id] {
		return
	}
	nonISOContainerNotes[id] = true
	infoNote(opts, "container id %q does not match ISO 6346 (%s); accepted as-is, ledger check applies in live mode",
		id, containerIDRe.String())
}

// matchVessel returns the single container whose Location is
// "OnBoard:<vessel>" (case-insensitive, '_' == ' ').
func matchVessel(index map[string]ContainerInfo, vessel string) (string, bool) {
	want := normalizeVessel(vessel)
	matches := []string{}
	for id, c := range index {
		loc := normalizeVessel(c.Location)
		onboard := strings.TrimPrefix(loc, "onboard:")
		if onboard != "" && onboard == want {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return "", false
}

func normalizeVessel(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

// buildIndex converts the JSON returned by GetAllContainers into an index.
func buildIndex(result []byte) map[string]ContainerInfo {
	var list []struct {
		ID       string `json:"ID"`
		Location string `json:"Location"`
	}
	_ = json.Unmarshal(result, &list)
	index := make(map[string]ContainerInfo, len(list))
	for _, c := range list {
		index[c.ID] = ContainerInfo{ID: c.ID, Location: c.Location}
	}
	return index
}

// needsIndex reports whether live mapping requires the ledger container index.
func needsIndex(records []RawRecord, opts Options) bool {
	if opts.Container != "" {
		return false
	}
	if opts.Vessel != "" {
		return true
	}
	for _, r := range records {
		if r.ContainerID != "" || r.Vessel != "" {
			return true
		}
	}
	return false
}
