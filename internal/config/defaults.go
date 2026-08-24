package config

const (
	// DefaultFrontendPort is the single TCP port used by the WebUI, REST API,
	// event WebSocket, and browser audio stream unless the administrator
	// chooses another port during installation.
	DefaultFrontendPort = 8989

	// DefaultListen is intentionally loopback-only as a safe daemon fallback.
	// The appliance installer will write an explicit LAN bind after first-run
	// security is configured and the selected port has passed its preflight.
	DefaultListen = "127.0.0.1:8989"
)
