package internal

import (
	"time"

	"github.com/atharvwasthere/Fastlane/internal/bench"
	"github.com/atharvwasthere/Fastlane/internal/download"
	"github.com/atharvwasthere/Fastlane/internal/loss"
	"github.com/atharvwasthere/Fastlane/internal/ping"
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
}

func DownloadEngine(p DownloadParams) bench.Engine {
	return download.NewBenchEngine(download.Config{
		URL:            p.URL,
		Threads:        p.Threads,
		Timeout:        p.Timeout,
		TestDuration:   p.TestDuration,
		CVThreshold:    0.03,
		UpdateInterval: 100 * time.Millisecond,
		MinSamples:     5,
	})
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
	Host         string
	Port         int
	Count        int
	PacketSize   int
	Rate         int
	Timeout      time.Duration
	EnableJitter bool
}

func LossEngine(p LossParams) bench.Engine {
	return loss.NewBenchEngine(loss.Config{
		Host:         p.Host,
		Port:         p.Port,
		Count:        p.Count,
		PacketSize:   p.PacketSize,
		Rate:         p.Rate,
		Timeout:      p.Timeout,
		EnableJitter: p.EnableJitter,
	})
}
