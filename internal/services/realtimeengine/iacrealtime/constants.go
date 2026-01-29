package iacrealtime

const (
	ContainerPath            = "/path"
	ContainerFormat          = "json"
	ContainerTempDirPattern  = "iac-realtime"
	KicsContainerPrefix      = "cli-iac-realtime-"
	ContainerResultsFileName = "results.json"
	InfoSeverity             = "info"
	IacEnginePath            = "/usr/local/bin"

	// Container engine names
	engineDocker = "docker"
	enginePodman = "podman"

	// engineVerifyTimeout is the timeout in seconds for verifying container engine availability
	engineVerifyTimeout = 5
)

// macOSDockerFallbackPaths contains additional paths to check for Docker on macOS
// These paths cover various Docker installation methods:
// - /usr/local/bin/docker: Standard location (Intel Macs)
// - /opt/homebrew/bin/docker: Homebrew on Apple Silicon
// - /Applications/Docker.app/Contents/Resources/bin/docker: Docker Desktop app bundle
// - ~/.docker/bin/docker: Docker Desktop CLI tools (resolved at runtime)
// - ~/.rd/bin/docker: Rancher Desktop (resolved at runtime)
var macOSDockerFallbackPaths = []string{
	"/usr/local/bin",
	"/opt/homebrew/bin",
	"/Applications/Docker.app/Contents/Resources/bin",
}

// macOSPodmanFallbackPaths contains additional paths to check for Podman on macOS
var macOSPodmanFallbackPaths = []string{
	"/usr/local/bin",
	"/opt/homebrew/bin",
}

var KicsErrorCodes = []string{"60", "50", "40", "30", "20"}

type LineIndex struct {
	Start int
	End   int
}

var Severities = map[string]string{
	"critical": "Critical",
	"high":     "High",
	"medium":   "Medium",
	"low":      "Low",
	"info":     "Info",
	"unknown":  "Unknown",
}
