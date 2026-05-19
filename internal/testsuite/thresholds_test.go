package testsuite

import (
	"testing"

	"github.com/atharvwasthere/Fastlane/internal/bench"
)

func TestThresholds_AllPass(t *testing.T) {
	r := bench.Result{Metrics: map[string]float64{
		"download_mbps": 200, "upload_mbps": 50, "latency_ms": 20, "jitter_ms": 2, "loss_percent": 0,
	}}
	th := Thresholds{MinDownloadMbps: 100, MinUploadMbps: 20, MaxLatencyMS: 50, MaxJitterMS: 10, MaxLossPercent: 1}
	if v := th.Check(r); len(v) != 0 {
		t.Fatalf("expected no violations, got %v", v)
	}
}

func TestThresholds_EachField(t *testing.T) {
	cases := []struct {
		name   string
		th     Thresholds
		r      bench.Result
		want   string
	}{
		{
			name: "download_below",
			th:   Thresholds{MinDownloadMbps: 100},
			r:    bench.Result{Metrics: map[string]float64{"download_mbps": 50}},
			want: "download_mbps",
		},
		{
			name: "upload_below",
			th:   Thresholds{MinUploadMbps: 10},
			r:    bench.Result{Metrics: map[string]float64{"upload_mbps": 2}},
			want: "upload_mbps",
		},
		{
			name: "latency_above",
			th:   Thresholds{MaxLatencyMS: 50},
			r:    bench.Result{Metrics: map[string]float64{"latency_ms": 200}},
			want: "latency_ms",
		},
		{
			name: "jitter_above",
			th:   Thresholds{MaxJitterMS: 5},
			r:    bench.Result{Metrics: map[string]float64{"jitter_ms": 20}},
			want: "jitter_ms",
		},
		{
			name: "loss_above",
			th:   Thresholds{MaxLossPercent: 0.5},
			r:    bench.Result{Metrics: map[string]float64{"loss_percent": 3}},
			want: "loss_percent",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := c.th.Check(c.r)
			if len(v) != 1 || v[0].Metric != c.want {
				t.Fatalf("got %v, want one violation on %s", v, c.want)
			}
		})
	}
}

func TestThresholds_ZeroDisables(t *testing.T) {
	r := bench.Result{Metrics: map[string]float64{"download_mbps": 1, "latency_ms": 5000}}
	th := Thresholds{} // all zero → no checks active
	if v := th.Check(r); len(v) != 0 {
		t.Fatalf("expected no checks active, got %v", v)
	}
}
