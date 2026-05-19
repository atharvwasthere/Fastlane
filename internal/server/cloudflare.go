package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const traceURL = "https://speed.cloudflare.com/cdn-cgi/trace"

type Colo struct {
	IP   string
	Loc  string
	Colo string
}

// String returns a compact human-readable form like "colo=DEL loc=IN".
// Empty fields are omitted so partial parses still render.
func (c Colo) String() string {
	parts := make([]string, 0, 2)
	if c.Colo != "" {
		parts = append(parts, "colo="+c.Colo)
	}
	if c.Loc != "" {
		parts = append(parts, "loc="+c.Loc)
	}
	return strings.Join(parts, " ")
}

// DetectCloudflareColo fetches /cdn-cgi/trace and parses the colo+loc fields.
// Network failures return a zero Colo and the error; callers should treat the
// colo as informational only and proceed.
func DetectCloudflareColo(ctx context.Context) (Colo, error) {
	return detectColoFrom(ctx, traceURL)
}

func detectColoFrom(ctx context.Context, url string) (Colo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Colo{}, err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Colo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Colo{}, fmt.Errorf("cloudflare trace: status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return Colo{}, err
	}
	return parseTrace(string(body)), nil
}

func parseTrace(body string) Colo {
	var c Colo
	for _, line := range strings.Split(body, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "ip":
			c.IP = strings.TrimSpace(v)
		case "loc":
			c.Loc = strings.TrimSpace(v)
		case "colo":
			c.Colo = strings.TrimSpace(v)
		}
	}
	return c
}
