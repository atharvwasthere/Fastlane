package stats

import (
	"io"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// CountingReader adds every byte read to *N atomically, so a sampler sees
// bytes as they arrive instead of only when a request completes.
type CountingReader struct {
	R io.Reader
	N *int64
}

func (c CountingReader) Read(p []byte) (int, error) {
	n, err := c.R.Read(p)
	atomic.AddInt64(c.N, int64(n))
	return n, err
}

// Sampler turns a growing byte counter into fixed-interval Mbps samples.
// Samples inside the warm-up window (TCP slow start, TLS handshakes) feed
// only the EWMA used for live display; the Welford stats, the convergence
// check and the final rate use steady-state samples only.
type Sampler struct {
	mu        sync.RWMutex
	warmUp    time.Duration
	minN      int64
	threshold float64

	start       time.Time
	lastAt      time.Time
	lastBytes   int64
	steadyAt    time.Time
	steadyBytes int64

	welford   *Welford
	ewma      *EWMA
	converged bool
}

// SamplerStats is a point-in-time view of a Sampler.
type SamplerStats struct {
	Mean, EWMA, StdDev, Min, Max, CV float64
	Samples                          int
	Converged                        bool
}

// NewSampler stops once the relative standard error of the steady-state
// mean (CV/√n) drops below threshold, after at least minSamples samples.
func NewSampler(warmUp time.Duration, minSamples int, threshold float64) *Sampler {
	return &Sampler{
		warmUp:    warmUp,
		minN:      int64(minSamples),
		threshold: threshold,
		welford:   NewWelford(),
		ewma:      NewEWMA(0.2),
	}
}

// Start marks t as the beginning of the run, when the byte counter is 0.
func (s *Sampler) Start(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.start, s.lastAt = t, t
}

// Observe records the byte counter at now and reports whether the run has
// converged. Once converged it stays converged.
func (s *Sampler) Observe(now time.Time, bytes int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	dt := now.Sub(s.lastAt).Seconds()
	delta := bytes - s.lastBytes
	s.lastAt, s.lastBytes = now, bytes
	// A negative delta is an upload rolling back a rejected request: no sample.
	if dt <= 0 || delta < 0 {
		return s.converged
	}
	mbps := float64(delta*8) / 1e6 / dt
	s.ewma.Add(mbps)

	if now.Sub(s.start) < s.warmUp {
		return false
	}
	// The first tick past warm-up straddles the boundary; it only marks the steady origin.
	if s.steadyAt.IsZero() {
		s.steadyAt, s.steadyBytes = now, bytes
		return false
	}

	s.welford.Add(mbps)
	n := s.welford.Count()
	if !s.converged && n >= s.minN && s.welford.Mean() > 0 {
		s.converged = s.welford.CoefficientOfVariation()/math.Sqrt(float64(n)) < s.threshold
	}
	return s.converged
}

// Mbps is the steady-state rate from the end of warm-up to now. If the run
// never left warm-up it falls back to the whole-run average.
func (s *Sampler) Mbps(now time.Time, bytes int64) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	from, base := s.start, int64(0)
	if !s.steadyAt.IsZero() && now.After(s.steadyAt) {
		from, base = s.steadyAt, s.steadyBytes
	}
	secs := now.Sub(from).Seconds()
	// An upload rollback can take the counter below the steady origin.
	if secs <= 0 || bytes <= base {
		return 0
	}
	return float64((bytes-base)*8) / 1e6 / secs
}

// Stats returns the current steady-state statistics.
func (s *Sampler) Stats() SamplerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return SamplerStats{
		Mean:      s.welford.Mean(),
		EWMA:      s.ewma.Value(),
		StdDev:    s.welford.SampleStdDev(),
		Min:       s.welford.Min(),
		Max:       s.welford.Max(),
		CV:        s.welford.CoefficientOfVariation(),
		Samples:   int(s.welford.Count()),
		Converged: s.converged,
	}
}
