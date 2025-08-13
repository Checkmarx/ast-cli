package util

import (
	"fmt"
	"net/url"
	"strings"
)

// SCMConfig holds configuration for different SCM providers
type SCMConfig struct {
	Provider string
	BaseURL  string
	Port     string
	IsCloud  bool
}

// SCMURLBuilder provides dynamic URL construction for different SCM providers
type SCMURLBuilder struct {
	config *SCMConfig
}

// NewSCMURLBuilder creates a new URL builder based on the provided configuration
func NewSCMURLBuilder(baseURL string, provider string) *SCMURLBuilder {
	config := detectSCMConfiguration(baseURL, provider)
	return &SCMURLBuilder{config: config}
}

// detectSCMConfiguration analyzes the provided URL to determine SCM configuration
func detectSCMConfiguration(apiURL, provider string) *SCMConfig {
	config := &SCMConfig{
		Provider: strings.ToLower(provider),
	}

	// If no URL provided, assume cloud service
	if apiURL == "" {
		config.IsCloud = true
		config.BaseURL = getDefaultCloudURL(provider)
		config.Port = getDefaultCloudPort(provider)
		return config
	}

	// Parse the provided URL
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		// If URL parsing fails, treat as cloud
		config.IsCloud = true
		config.BaseURL = getDefaultCloudURL(provider)
		config.Port = getDefaultCloudPort(provider)
		return config
	}

	// Determine if it's a cloud service based on hostname
	config.IsCloud = isCloudService(parsedURL.Host, provider)

	if config.IsCloud {
		config.BaseURL = getDefaultCloudURL(provider)
		config.Port = getDefaultCloudPort(provider)
	} else {
		// On-premise configuration
		config.BaseURL = fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)
		config.Port = parsedURL.Port()
		if config.Port == "" {
			config.Port = getDefaultOnPremPort(provider, parsedURL.Scheme)
		}
	}

	return config
}

// isCloudService determines if the hostname belongs to a cloud service
func isCloudService(hostname, provider string) bool {
	cloudHosts := map[string][]string{
		"github":    {"github.com", "api.github.com"},
		"gitlab":    {"gitlab.com"},
		"bitbucket": {"bitbucket.org", "api.bitbucket.org"},
		"azure":     {"dev.azure.com", "visualstudio.com"},
	}

	hosts, exists := cloudHosts[strings.ToLower(provider)]
	if !exists {
		return false
	}

	for _, host := range hosts {
		if strings.Contains(hostname, host) {
			return true
		}
	}
	return false
}

// getDefaultCloudURL returns the default cloud URL for each provider
func getDefaultCloudURL(provider string) string {
	cloudURLs := map[string]string{
		"github":    "https://api.github.com/repos/",
		"gitlab":    "https://gitlab.com/api/v4/",
		"bitbucket": "https://api.bitbucket.org/2.0/",
		"azure":     "https://dev.azure.com/",
	}
	return cloudURLs[strings.ToLower(provider)]
}

// getDefaultCloudPort returns the default port for cloud services (usually 443 for HTTPS)
func getDefaultCloudPort(provider string) string {
	return "443"
}

// getDefaultOnPremPort returns common default ports for on-premise installations
func getDefaultOnPremPort(provider, scheme string) string {
	if scheme == "https" {
		return "443"
	}

	defaultPorts := map[string]string{
		"github":    "80",   // GitHub Enterprise often uses 80/443
		"gitlab":    "80",   // GitLab CE/EE often uses 80/443
		"bitbucket": "7990", // Bitbucket Server default
		"azure":     "8080", // Azure DevOps Server default
	}

	if port, exists := defaultPorts[strings.ToLower(provider)]; exists {
		return port
	}
	return "80"
}

// BuildAPIURL constructs the complete API URL based on the SCM configuration
func (b *SCMURLBuilder) BuildAPIURL() string {
	if b.config.IsCloud {
		return b.config.BaseURL
	}

	// For on-premise, append the appropriate API suffix
	switch b.config.Provider {
	case "github":
		return b.config.BaseURL + "/api/v3/repos/"
	case "gitlab":
		return b.config.BaseURL + "/api/v4/"
	case "bitbucket":
		return b.config.BaseURL + "/rest/api/1.0/"
	case "azure":
		return b.config.BaseURL
	default:
		return b.config.BaseURL
	}
}

// GetFullURL returns the complete URL including port if non-standard
func (b *SCMURLBuilder) GetFullURL() string {
	baseURL := b.BuildAPIURL()

	// Only append port if it's not standard (80 for HTTP, 443 for HTTPS)
	if b.config.Port != "" && b.config.Port != "80" && b.config.Port != "443" {
		parsedURL, err := url.Parse(baseURL)
		if err == nil && parsedURL.Port() == "" {
			parsedURL.Host = fmt.Sprintf("%s:%s", parsedURL.Hostname(), b.config.Port)
			return parsedURL.String()
		}
	}

	return baseURL
}

// IsCloud returns whether this is a cloud service
func (b *SCMURLBuilder) IsCloud() bool {
	return b.config.IsCloud
}

// GetProvider returns the SCM provider name
func (b *SCMURLBuilder) GetProvider() string {
	return b.config.Provider
}

// GetPort returns the configured port
func (b *SCMURLBuilder) GetPort() string {
	return b.config.Port
}

// GetBaseURL returns the base URL without API suffixes
func (b *SCMURLBuilder) GetBaseURL() string {
	return b.config.BaseURL
}
