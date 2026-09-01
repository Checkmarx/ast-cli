package governance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModelDetectionResult holds the result of model detection
type ModelDetectionResult struct {
	Model      string    `json:"model"`
	Vendor     string    `json:"vendor"`
	Confidence float64   `json:"confidence"`
	Source     string    `json:"source"`
	SourcePath string    `json:"sourcePath,omitempty"`
	DetectedAt time.Time `json:"detectedAt"`
	Errors     []string  `json:"errors,omitempty"`
}

// DetectionContext holds context for model detection
type DetectionContext struct {
	UserID   string
	AgentID  string
	DBClient interface{}
	// HookModel is the model name Claude Code reports directly in the
	// SessionStart hook payload ("model" field). This is the actual model
	// running the session — including per-session overrides made via
	// `/model` — and takes priority over every other detection layer.
	HookModel string
}

// DetectModel orchestrates fallback strategy for Claude Code model detection.
func DetectModel(ctx DetectionContext) ModelDetectionResult {
	result := ModelDetectionResult{
		DetectedAt: time.Now().UTC(),
		Vendor:     "anthropic",
	}

	// Layer 0: Model reported directly by Claude Code in the hook payload.
	// This reflects the model actually running the session, including
	// `/model` overrides, so it is preferred over static config sources.
	if ctx.HookModel != "" {
		return ModelDetectionResult{
			Model:      normalizeModelName(ctx.HookModel),
			Vendor:     "anthropic",
			Confidence: 1.0,
			Source:     "hook_payload",
			DetectedAt: time.Now().UTC(),
		}
	}

	// Layer 1: Settings file (~/.claude/settings.json)
	if model, path, err := detectFromSettingsFile(); err == nil && model != "" {
		return ModelDetectionResult{
			Model:      normalizeModelName(model),
			Vendor:     "anthropic",
			Confidence: 0.95,
			Source:     "settings_file",
			SourcePath: path,
			DetectedAt: time.Now().UTC(),
		}
	} else if err != nil {
		result.Errors = append(result.Errors, "settings_file: "+err.Error())
	}

	// Layer 2: Environment variable
	if model := detectFromEnv(); model != "" {
		return ModelDetectionResult{
			Model:      normalizeModelName(model),
			Vendor:     "anthropic",
			Confidence: 0.80,
			Source:     "env_var",
			DetectedAt: time.Now().UTC(),
		}
	}

	// Layer 3: Default fallback — no signal was found. Report "unknown" rather
	// than guessing a specific model, since that would misrepresent the session.
	result.Errors = append(result.Errors, "all_detection_layers_exhausted")
	return ModelDetectionResult{
		Model:      "unknown",
		Vendor:     "anthropic",
		Confidence: 0.0,
		Source:     "default",
		DetectedAt: time.Now().UTC(),
		Errors:     result.Errors,
	}
}

// normalizeModelName maps Claude Code's raw model identifiers (e.g.
// "claude-sonnet-4-5-20250929", "claude-haiku-4-5-20251001") to the short
// family names used by the ai_models catalog. Unrecognized names are
// returned unchanged so newer models are still recorded rather than dropped.
func normalizeModelName(raw string) string {
	lower := strings.ToLower(raw)
	switch {
	case strings.Contains(lower, "opus"):
		return "claude-opus-5"
	case strings.Contains(lower, "sonnet"):
		return "claude-sonnet-5"
	case strings.Contains(lower, "haiku"):
		return "claude-haiku-4"
	default:
		return raw
	}
}

// detectFromSettingsFile reads ~/.claude/settings.json and extracts the model
func detectFromSettingsFile() (model, path string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("cannot resolve home: %w", err)
	}

	path = filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot read: %w", err)
	}

	var cfg struct {
		Model       string `json:"model"`
		ModelConfig struct {
			DefaultModel string `json:"defaultModel"`
		} `json:"modelConfig"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", "", fmt.Errorf("cannot parse: %w", err)
	}

	if cfg.Model != "" {
		return cfg.Model, path, nil
	}
	if cfg.ModelConfig.DefaultModel != "" {
		return cfg.ModelConfig.DefaultModel, path, nil
	}

	return "", path, fmt.Errorf("model field not found in settings")
}

// detectFromEnv checks environment variables for model configuration
func detectFromEnv() string {
	varNames := []string{
		"CLAUDE_MODEL",
		"CLAUDE_DEFAULT_MODEL",
	}
	for _, varName := range varNames {
		if model := os.Getenv(varName); model != "" {
			return model
		}
	}
	return ""
}
