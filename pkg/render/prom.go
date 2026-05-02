package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

// promRenderer hand-writes the Prometheus exposition format. One label —
// `server` — keeps cardinality low; the metric name encodes everything else.
type promRenderer struct {
	w io.Writer
}

func newPromRenderer(w io.Writer, _ Options) Renderer {
	return &promRenderer{w: w}
}

func (p *promRenderer) Header(_, _ string)    {}
func (p *promRenderer) Live(_ bench.Snapshot) {}

func (p *promRenderer) Final(r bench.Result) {
	server := escapeLabel(r.Server)
	emit := func(name, help, typ string, value any) {
		fmt.Fprintf(p.w, "# HELP %s %s\n", name, help)
		fmt.Fprintf(p.w, "# TYPE %s %s\n", name, typ)
		fmt.Fprintf(p.w, "%s{server=\"%s\"} %v\n", name, server, value)
	}

	keys := sortedKeys(r.Metrics)
	for _, k := range keys {
		emit(
			fmt.Sprintf("fastlane_%s_%s", r.Kind, k),
			fmt.Sprintf("%s %s", r.Kind, k),
			"gauge",
			r.Metrics[k],
		)
	}
	cKeys := sortedKeysI(r.Counters)
	for _, k := range cKeys {
		emit(
			fmt.Sprintf("fastlane_%s_%s", r.Kind, k),
			fmt.Sprintf("%s %s count", r.Kind, k),
			"counter",
			r.Counters[k],
		)
	}
	fKeys := sortedKeysB(r.Flags)
	for _, k := range fKeys {
		v := 0
		if r.Flags[k] {
			v = 1
		}
		emit(
			fmt.Sprintf("fastlane_%s_%s", r.Kind, k),
			fmt.Sprintf("%s %s flag", r.Kind, k),
			"gauge",
			v,
		)
	}
}

func (p *promRenderer) Error(err error) {
	fmt.Fprintf(p.w, "# ERROR %s\n", err.Error())
}

func escapeLabel(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' || c == '\\' {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return string(out)
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func sortedKeysI(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func sortedKeysB(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
