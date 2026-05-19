package internal

import (
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/download"
	"github.com/atharvwasthere/Fastlane/internal/loss"
	"github.com/atharvwasthere/Fastlane/internal/ping"
	"github.com/atharvwasthere/Fastlane/internal/testsuite"
	"github.com/atharvwasthere/Fastlane/internal/upload"
)

// This file is the ONLY place cmd-side code talks to concrete engine
// packages. Per the architecture skill, cmd/<name>.go files import this
// package and call PingEngine/DownloadEngine/UploadEngine/LossEngine — they
// never see download.Config, upload.Result, etc.

// PingParams is the cmd-facing knob set for ping.
type PingParams struct {
	Host       string
	Iterations int
	Timeout    time.Duration
}

// PingEngine returns a bench.Engine for the configured ping run.
func PingEngine(p PingParams) bench.Engine {
	return ping.NewBenchEngine(ping.BenchConfig{
		Host:       p.Host,
		Iterations: p.Iterations,
		Timeout:    p.Timeout,
	})
}

// DownloadParams mirrors download.Config but stays in the cmd layer's vocabulary.
type DownloadParams struct {
	URL          string
	Threads      int
	Timeout      time.Duration
	TestDuration time.Duration
	Adaptive     bool // hill-climb thread count before recording
}

func DownloadEngine(p DownloadParams) bench.Engine {
	base := download.Config{
		URL:            p.URL,
		Threads:        p.Threads,
		Timeout:        p.Timeout,
		TestDuration:   p.TestDuration,
		CVThreshold:    0.03,
		UpdateInterval: 100 * time.Millisecond,
		MinSamples:     5,
	}
	if p.Adaptive {
		return download.NewAdaptiveBenchEngine(download.DefaultAdaptive(base))
	}
	return download.NewBenchEngine(base)
}

type UploadParams struct {
	URL          string
	Threads      int
	Timeout      time.Duration
	TestDuration time.Duration
}

func UploadEngine(p UploadParams) bench.Engine {
	return upload.NewBenchEngine(upload.Config{
		URL:            p.URL,
		Threads:        p.Threads,
		Timeout:        p.Timeout,
		TestDuration:   p.TestDuration,
		CVThreshold:    0.03,
		UpdateInterval: 100 * time.Millisecond,
		MinSamples:     5,
		ChunkSize:      1024 * 1024,
	})
}

type LossParams struct {
	Mode         string // "http" (default) | "udp"
	Host         string
	Port         int
	Count        int
	PacketSize   int
	Rate         int
	Timeout      time.Duration
	EnableJitter bool
}

// TestParams configures the four-step test suite. URLs/hosts are resolved by
// cmd/test.go before factory construction so the factory itself stays pure.
type TestParams struct {
	Host         string
	DownloadURL  string
	UploadURL    string
	Threads      int
	Timeout      time.Duration
	TestDuration time.Duration
	LossCount    int
	LossRate     int
}

func TestEngine(p TestParams) bench.Engine {
	return testsuite.NewEngine(testsuite.Config{
		Server: p.Host,
		Ping: PingEngine(PingParams{
			Host:       p.Host,
			Iterations: 5,
			Timeout:    p.Timeout,
		}),
		Download: DownloadEngine(DownloadParams{
			URL:          p.DownloadURL,
			Threads:      p.Threads,
			Timeout:      p.Timeout,
			TestDuration: p.TestDuration,
		}),
		Upload: UploadEngine(UploadParams{
			URL:          p.UploadURL,
			Threads:      p.Threads,
			Timeout:      p.Timeout,
			TestDuration: p.TestDuration,
		}),
		Loss: LossEngine(LossParams{
			Mode:    "http",
			Host:    p.DownloadURL,
			Count:   p.LossCount,
			Rate:    p.LossRate,
			Timeout: p.Timeout,
		}),
	})
}

func LossEngine(p LossParams) bench.Engine {
	mode := loss.Mode(p.Mode)
	if mode == "" {
		mode = loss.ModeHTTP
	}
	return loss.NewBenchEngine(loss.Config{
		Mode:         mode,
		Host:         p.Host,
		Port:         p.Port,
		Count:        p.Count,
		PacketSize:   p.PacketSize,
		Rate:         p.Rate,
		Timeout:      p.Timeout,
		EnableJitter: p.EnableJitter,
	})
}
