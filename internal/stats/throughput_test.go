package stats

import (
	"math"
	"testing"
	"time"
)

// feed ticks every 100ms, adding rate(i) Mbps worth of bytes per tick.
func feed(s *Sampler, t0 time.Time, ticks int, rate func(i int) float64) (time.Time, int64) {
	var bytes int64
	now := t0
	for i := 1; i <= ticks; i++ {
		now = t0.Add(time.Duration(i) * 100 * time.Millisecond)
		bytes += int64(rate(i) * 1e6 / 8 / 10)
		s.Observe(now, bytes)
	}
	return now, bytes
}

func TestSamplerExcludesWarmUp(t *testing.T) {
	t0 := time.Unix(0, 0)
	s := NewSampler(time.Second, 1000, 0.03) // minN high: never stop early
	s.Start(t0)
	// Slow start: 5 Mbps for the first second, then a steady 100 Mbps.
	now, bytes := feed(s, t0, 50, func(i int) float64 {
		if i <= 10 {
			return 5
		}
		return 100
	})

	if got := s.Mbps(now, bytes); math.Abs(got-100) > 0.5 {
		t.Fatalf("steady-state Mbps = %.2f, want ~100 (warm-up must be excluded)", got)
	}
	if st := s.Stats(); math.Abs(st.Mean-100) > 0.5 || st.Min < 99 {
		t.Fatalf("steady stats include warm-up: mean=%.2f min=%.2f", st.Mean, st.Min)
	}
}

func TestSamplerConvergesOnSteadyRate(t *testing.T) {
	t0 := time.Unix(0, 0)
	s := NewSampler(time.Second, 5, 0.03)
	s.Start(t0)
	// A perfectly steady link (CV = 0) used to never converge under the old "cv > 0" guard.
	feed(s, t0, 17, func(int) float64 { return 50 })
	if !s.Stats().Converged {
		t.Fatalf("expected convergence on a steady rate, stats=%+v", s.Stats())
	}
}

func TestSamplerConvergedIsSticky(t *testing.T) {
	t0 := time.Unix(0, 0)
	s := NewSampler(0, 2, 0.03)
	s.Start(t0)
	now, bytes := feed(s, t0, 5, func(int) float64 { return 50 })
	if !s.Stats().Converged {
		t.Fatal("expected convergence")
	}
	// A wild sample after convergence must not flip it back.
	s.Observe(now.Add(100*time.Millisecond), bytes+100_000_000)
	if !s.Stats().Converged {
		t.Fatal("converged flipped back to false")
	}
}

func TestSamplerSkipsNegativeDelta(t *testing.T) {
	t0 := time.Unix(0, 0)
	s := NewSampler(0, 1000, 0.03)
	s.Start(t0)
	now, bytes := feed(s, t0, 5, func(int) float64 { return 50 })
	before := s.Stats().Samples
	s.Observe(now.Add(100*time.Millisecond), bytes-1000) // an upload rolled back
	if got := s.Stats(); got.Samples != before || got.Min < 0 {
		t.Fatalf("negative delta produced a sample: %+v", got)
	}
}

func TestSamplerFallsBackWithinWarmUp(t *testing.T) {
	t0 := time.Unix(0, 0)
	s := NewSampler(10*time.Second, 5, 0.03)
	s.Start(t0)
	now, bytes := feed(s, t0, 10, func(int) float64 { return 40 })
	if got := s.Mbps(now, bytes); math.Abs(got-40) > 0.5 {
		t.Fatalf("whole-run fallback = %.2f, want ~40", got)
	}
	if s.Stats().Samples != 0 {
		t.Fatal("warm-up ticks must not become steady samples")
	}
}
