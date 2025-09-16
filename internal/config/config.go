package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// TrinoConfig holds Trino connection parameters
type TrinoConfig struct {
	// Basic connection parameters
	Host              string
	Port              int
	User              string
	Password          string
	Catalog           string
	Schema            string
	Scheme            string
	SSL               bool
	SSLInsecure       bool
	AllowWriteQueries bool          // Controls whether non-read-only SQL queries are allowed
	QueryTimeout      time.Duration // Query execution timeout

	// OAuth mode configuration
	OAuthEnabled  bool   // Enable OAuth 2.1 authentication
	OAuthProvider string // OAuth provider: "hmac", "okta", "google", "azure"
	JWTSecret     string // JWT signing secret for HMAC provider

	// OIDC provider configuration
	OIDCIssuer       string // OIDC issuer URL
	OIDCAudience     string // OIDC audience
	OIDCClientID     string // OIDC client ID
	OIDCClientSecret string // OIDC client secret
	OAuthRedirectURI string // Fixed OAuth redirect URI (overrides dynamic callback)
}

// NewTrinoConfig creates a new TrinoConfig with values from environment variables or defaults
func NewTrinoConfig() (*TrinoConfig, error) {
	port, _ := strconv.Atoi(getEnv("TRINO_PORT", "8080"))
	ssl, _ := strconv.ParseBool(getEnv("TRINO_SSL", "true"))
	sslInsecure, _ := strconv.ParseBool(getEnv("TRINO_SSL_INSECURE", "true"))
	scheme := getEnv("TRINO_SCHEME", "https")
	allowWriteQueries, _ := strconv.ParseBool(getEnv("TRINO_ALLOW_WRITE_QUERIES", "false"))

	// Smart OAuth detection: if OAUTH_PROVIDER is explicitly set, enable OAuth
	oauthProvider := strings.ToLower(getEnv("OAUTH_PROVIDER", ""))
	oauthEnabled := false

	if oauthProvider != "" {
		// User explicitly set OAUTH_PROVIDER, enable OAuth
		oauthEnabled = true
		log.Printf("INFO: OAuth automatically enabled because OAUTH_PROVIDER=%s is set", oauthProvider)

		// Allow explicit override if needed
		if explicitEnabled := os.Getenv("TRINO_OAUTH_ENABLED"); explicitEnabled != "" {
			oauthEnabled, _ = strconv.ParseBool(explicitEnabled)
			if !oauthEnabled {
				log.Printf("WARNING: OAuth explicitly disabled via TRINO_OAUTH_ENABLED=false despite OAUTH_PROVIDER being set")
			}
		}
	} else {
		// No OAUTH_PROVIDER set, check TRINO_OAUTH_ENABLED (defaults to false)
		oauthEnabled, _ = strconv.ParseBool(getEnv("TRINO_OAUTH_ENABLED", "false"))
		oauthProvider = "hmac" // Default provider when OAuth is enabled without explicit provider
	}
	jwtSecret := getEnv("JWT_SECRET", "")

	// OIDC configuration with secure defaults
	oidcIssuer := getEnv("OIDC_ISSUER", "")
	oidcAudience := getEnv("OIDC_AUDIENCE", "") // No default - must be explicitly configured
	oidcClientID := getEnv("OIDC_CLIENT_ID", "")
	oidcClientSecret := getEnv("OIDC_CLIENT_SECRET", "")
	oauthRedirectURI := getEnv("OAUTH_REDIRECT_URI", "")

	// Parse query timeout from environment variable
	const defaultTimeout = 30
	timeoutStr := getEnv("TRINO_QUERY_TIMEOUT", strconv.Itoa(defaultTimeout))
	timeoutInt, err := strconv.Atoi(timeoutStr)

	// Validate timeout value
	switch {
	case err != nil:
		log.Printf("WARNING: Invalid TRINO_QUERY_TIMEOUT '%s': not an integer. Using default of %d seconds", timeoutStr, defaultTimeout)
		timeoutInt = defaultTimeout
	case timeoutInt <= 0:
		log.Printf("WARNING: Invalid TRINO_QUERY_TIMEOUT '%d': must be positive. Using default of %d seconds", timeoutInt, defaultTimeout)
		timeoutInt = defaultTimeout
	}

	queryTimeout := time.Duration(timeoutInt) * time.Second

	// If using HTTPS, force SSL to true
	if strings.EqualFold(scheme, "https") {
		ssl = true
	}

	// Log a warning if write queries are allowed
	if allowWriteQueries {
		log.Println("WARNING: Write queries are enabled (TRINO_ALLOW_WRITE_QUERIES=true). SQL injection protection is bypassed.")
	}

	// Validate and log OAuth mode status
	if oauthEnabled {
		// Validate OAuth provider
		validProviders := map[string]bool{"hmac": true, "okta": true, "google": true, "azure": true}
		if !validProviders[oauthProvider] {
			return nil, fmt.Errorf("invalid OAuth provider '%s'. Supported providers: hmac, okta, google, azure", oauthProvider)
		}

		log.Printf("INFO: OAuth 2.1 authentication enabled (TRINO_OAUTH_ENABLED=true) with provider: %s", oauthProvider)
		if oauthProvider == "hmac" && jwtSecret == "" {
			return nil, fmt.Errorf("security error: JWT_SECRET is required when using HMAC provider. Set JWT_SECRET environment variable")
		}
		if oauthProvider != "hmac" && oidcIssuer == "" {
			log.Printf("WARNING: OIDC_ISSUER not set for %s provider. OAuth authentication may fail.", oauthProvider)
		}

		// Validate audience configuration for security
		if oauthProvider != "hmac" && oidcAudience == "" {
			return nil, fmt.Errorf("security error: OIDC_AUDIENCE is required for %s provider. Set OIDC_AUDIENCE environment variable to your service-specific audience", oauthProvider)
		}

		if oauthProvider != "hmac" {
			log.Printf("INFO: JWT audience validation enabled for: %s", oidcAudience)
		}
		if oauthRedirectURI != "" {
			log.Printf("INFO: Fixed OAuth redirect URI configured: %s", oauthRedirectURI)
		}
	} else {
		log.Println("INFO: OAuth authentication is disabled (TRINO_OAUTH_ENABLED=false). Enable OAuth for production deployments.")
	}

	return &TrinoConfig{
		Host:              getEnv("TRINO_HOST", "localhost"),
		Port:              port,
		User:              getEnv("TRINO_USER", "trino"),
		Password:          getEnv("TRINO_PASSWORD", ""),
		Catalog:           getEnv("TRINO_CATALOG", "memory"),
		Schema:            getEnv("TRINO_SCHEMA", "default"),
		Scheme:            scheme,
		SSL:               ssl,
		SSLInsecure:       sslInsecure,
		AllowWriteQueries: allowWriteQueries,
		QueryTimeout:      queryTimeout,
		OAuthEnabled:      oauthEnabled,
		OAuthProvider:     oauthProvider,
		JWTSecret:         jwtSecret,
		OIDCIssuer:        oidcIssuer,
		OIDCAudience:      oidcAudience,
		OIDCClientID:      oidcClientID,
		OIDCClientSecret:  oidcClientSecret,
		OAuthRedirectURI:  oauthRedirectURI,
	}, nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
