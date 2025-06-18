# Fastlane – The Minimalist Speedtest CLI

> Fastlane ▸ Your network. Deconstructed.
*“Built for `developers` who **hate lag**.”*
---

### ⚡ What is Fastlane?

**Fastlane** is a snappy, developer-focused CLI tool to benchmark network performance — ping, download, upload — with zero fluff.  
Built to replace bloated GUIs with clean terminal output and modular commands.

---

<img src="./Images/Banner.png" alt="Fastlane Banner">

### 🔧 Features

- Modular commands: `ping`, `download`, `upload`, `full`, `xray`, `live`, `report`
- Latency breakdown: DNS, TCP, TLS, TTFB
- Real-time progress bars and ASCII graphs
- Geo-based server selection
- Session-based reporting
- `--json` output for automation
- Clean UI with box-drawn layouts and techy vibes

---

### 🧪 Usage

```bash
fastlane ping
fastlane download
fastlane upload
fastlane full
````

Optional flags:

```bash
--server <hostname>      # Override auto-selected server
--json                   # Output results in JSON
--verbose                # Show detailed logs
```

---

### 🗂️ Project Structure

```
fastlane/
├── cmd/                      # All Cobra commands go here
│   ├── root.go               # Root command (fastlane)
│   ├── ping.go               # `fastlane ping`
│   ├── download.go           # `fastlane download`
│   ├── upload.go             # `fastlane upload`
│   ├── full.go               # `fastlane full`
│   ├── live.go               # `fastlane live`
│   ├── xray.go               # `fastlane xray`
│   ├── report.go             # `fastlane report`
│   └── version.go            # `fastlane version`
│
├── internal/                 # Core logic lives here, testable independently
│   ├── nettest/              # Network testing logic
│   │   ├── ping.go
│   │   ├── download.go
│   │   ├── upload.go
│   │   └── full.go
│   ├── ui/                   # ASCII UI rendering, spinners, color layouts
│   │   ├── printer.go
│   │   ├── progress.go
│   │   └── graph.go
│   ├── report/               # Report saving/loading
│   │   ├── store.go
│   │   └── types.go
│   └── utils/                # Helper funcs, JSON handling, formatting, etc.
│       └── helpers.go
│
├── assets/                   # Optional: JSON test files, ASCII art, flags
│   └── servers.json
│
├── go.mod
├── go.sum
└── main.go                   # Entrypoint → sets up CLI and runs rootCmd
```

---

### 🛠 Build & Run

```bash
go build -o fastlane
./fastlane ping
```

---

### 🧑‍💻 Author

Made with latency hate by **Atharv Singh**
*“Built for `developers` who **hate lag**.”*

---
