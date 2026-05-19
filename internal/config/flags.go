package config

// GlobalFlags holds all persistent flags available across the CLI
type GlobalFlags struct {
	JSON    bool
	Verbose bool
	Timeout uint32
	Debug   bool
	NoColor bool
}

// CommandFlags holds command-specific flags
type CommandFlags struct {
	Threads        int
	SaveReport     bool
	Server         string
	AutoServer     bool
	AdaptiveThreads bool
	PreferLatency  bool
	City           string
	Coords         string
	Keyword        string
	ServerID       string
	Count          int    // For loss test: number of packets
	Rate           int    // For loss test: packets per second
}

// DefaultGlobalFlags returns sensible defaults for global flags
func DefaultGlobalFlags() GlobalFlags {
	return GlobalFlags{
		JSON:    false,
		Verbose: false,
		Timeout: 30,
		Debug:   false,
		NoColor: false,
	}
}

// DefaultCommandFlags returns sensible defaults for command flags
func DefaultCommandFlags() CommandFlags {
	return CommandFlags{
		Threads:       4,
		SaveReport:    true,
		Server:        "",
		PreferLatency: false,
		City:          "",
		Coords:        "",
		Keyword:       "",
		ServerID:      "",
		Count:         100,
		Rate:          10,
	}
}
