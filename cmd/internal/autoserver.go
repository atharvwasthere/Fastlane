package internal

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/atharvwasthere/Fastlane/internal/server"
)

// AutoServerURL runs server.AutoSelect and returns a URL suitable for the
// named command kind. Per-server `download_path` / `upload_path` come from
// the embedded servers.json; if a server lacks the path for the requested
// kind, AutoServerURL falls back to known-good defaults (Cloudflare's
// __down/__up). On any error a single stderr notice is printed and the
// caller's `fallback` is returned untouched.
//
// URL conventions:
//   - download → https://<host><download_path>
//   - upload   → https://<host><upload_path>
//   - ping     → <host>          (ping accepts bare host)
func AutoServerURL(ctx context.Context, kind, fallback string) string {
	s, err := server.AutoSelect(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "auto-server: %v — using default\n", err)
		return fallback
	}
	fmt.Fprintf(os.Stderr, "auto-server: selected %s (%s, %s)\n", s.Name, s.Location, s.Country)
	switch kind {
	case "download":
		return buildURL(s.Host, s.Port, pickPath(s.DownloadPath, "/__down?bytes=104857600"))
	case "upload":
		return buildURL(s.Host, s.Port, pickPath(s.UploadPath, "/__up"))
	case "ping":
		return s.Host
	default:
		return s.Host
	}
}

func pickPath(p, fallback string) string {
	if p == "" {
		return fallback
	}
	if !strings.HasPrefix(p, "/") {
		return "/" + p
	}
	return p
}

func buildURL(host string, port int, path string) string {
	scheme := "https"
	hostPart := host
	if port != 0 && port != 443 {
		hostPart = fmt.Sprintf("%s:%d", host, port)
	}
	return fmt.Sprintf("%s://%s%s", scheme, hostPart, path)
}
