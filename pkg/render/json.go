package render

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// jsonRenderer emits the schema-locked v2.0.0 envelope. Adding a metric to
// the card without adding it to JSON is a bug — both consume the same
// bench.Result map keys, so they stay in lockstep automatically.
type jsonRenderer struct {
	w   io.Writer
	enc *json.Encoder
}

func newJSONRenderer(w io.Writer, _ Options) Renderer {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return &jsonRenderer{w: w, enc: enc}
}

// Header / Live are no-ops — JSON callers want one envelope only.
func (j *jsonRenderer) Header(_, _ string)      {}
func (j *jsonRenderer) Live(_ bench.Snapshot)   {}

func (j *jsonRenderer) Final(r bench.Result) {
	envelope := struct {
		Version         string             `json:"version"`
		Command         string             `json:"command"`
		Timestamp       string             `json:"timestamp"`
		Server          string             `json:"server"`
		DurationSeconds float64            `json:"duration_seconds"`
		Metrics         map[string]float64 `json:"metrics"`
		Counters        map[string]int64   `json:"counters"`
		Flags           map[string]bool    `json:"flags"`
		Layers          map[string]float64 `json:"layers,omitempty"`
		Notes           []string           `json:"notes"`
	}{
		Version:         "2.0.0",
		Command:         string(r.Kind),
		Timestamp:       r.StartedAt.UTC().Format(time.RFC3339),
		Server:          r.Server,
		DurationSeconds: r.Duration.Seconds(),
		Metrics:         orMap(r.Metrics),
		Counters:        orMapI(r.Counters),
		Flags:           orMapB(r.Flags),
		Layers:          r.Layers,
		Notes:           orSlice(r.Notes),
	}
	_ = j.enc.Encode(envelope)
}

func (j *jsonRenderer) Error(err error) {
	_ = j.enc.Encode(struct {
		Version string `json:"version"`
		Error   string `json:"error"`
	}{Version: "2.0.0", Error: err.Error()})
	fmt.Fprintln(j.w)
}

func orMap(m map[string]float64) map[string]float64 {
	if m == nil {
		return map[string]float64{}
	}
	return m
}
func orMapI(m map[string]int64) map[string]int64 {
	if m == nil {
		return map[string]int64{}
	}
	return m
}
func orMapB(m map[string]bool) map[string]bool {
	if m == nil {
		return map[string]bool{}
	}
	return m
}
func orSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
