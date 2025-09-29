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
	OAuthMode     string // OAuth operational mode: "native" or "proxy"
	OAuthProvider string // OAuth provider: "hmac", "okta", "google", "azure"
	JWTSecret     string // JWT signing secret for HMAC provider

	// OIDC provider configuration
	OIDCIssuer       string // OIDC issuer URL
	OIDCAudience     string // OIDC audience
	OIDCClientID     string // OIDC client ID
	OIDCClientSecret       string // OIDC client secret
	OAuthRedirectURIs      string // OAuth redirect URIs - single URI or comma-separated list

	// Allowlist configuration for filtering catalogs, schemas, and tables
	AllowedCatalogs []string // List of allowed catalogs (empty means no filtering)
	AllowedSchemas  []string // List of allowed schemas in catalog.schema format
	AllowedTables   []string // List of allowed tables in catalog.schema.table format
}

// NewTrinoConfig creates a new TrinoConfig with values from environment variables or defaults
func NewTrinoConfig() (*TrinoConfig, error) {
	port, _ := strconv.Atoi(getEnv("TRINO_PORT", "8080"))
	ssl, _ := strconv.ParseBool(getEnv("TRINO_SSL", "true"))
	sslInsecure, _ := strconv.ParseBool(getEnv("TRINO_SSL_INSECURE", "true"))
	scheme := getEnv("TRINO_SCHEME", "https")
	allowWriteQueries, _ := strconv.ParseBool(getEnv("TRINO_ALLOW_WRITE_QUERIES", "false"))

	// OAuth mode configuration: native (default) or proxy
	oauthMode := strings.ToLower(getEnv("OAUTH_MODE", "native"))

	// Smart OAuth detection: if OAUTH_PROVIDER is explicitly set, enable OAuth
	oauthProvider := strings.ToLower(getEnv("OAUTH_PROVIDER", ""))
	oauthEnabled := false

	if oauthProvider != "" {
		// User explicitly set OAUTH_PROVIDER, enable OAuth
		oauthEnabled = true
		log.Printf("INFO: OAuth automatically enabled because OAUTH_PROVIDER=%s is set", oauthProvider)

		// Allow explicit override if needed
		if explicitEnabled := os.Getenv("OAUTH_ENABLED"); explicitEnabled != "" {
			oauthEnabled, _ = strconv.ParseBool(explicitEnabled)
			if !oauthEnabled {
				log.Printf("WARNING: OAuth explicitly disabled via OAUTH_ENABLED=false despite OAUTH_PROVIDER being set")
			}
		}
	} else {
		// No OAUTH_PROVIDER set, check OAUTH_ENABLED (defaults to false)
		oauthEnabled, _ = strconv.ParseBool(getEnv("OAUTH_ENABLED", "false"))
		oauthProvider = "hmac" // Default provider when OAuth is enabled without explicit provider
	}
	jwtSecret := getEnv("JWT_SECRET", "")

	// OIDC configuration with secure defaults
	oidcIssuer := getEnv("OIDC_ISSUER", "")
	oidcAudience := getEnv("OIDC_AUDIENCE", "") // No default - must be explicitly configured
	oidcClientID := getEnv("OIDC_CLIENT_ID", "")
	oidcClientSecret := getEnv("OIDC_CLIENT_SECRET", "")
	oauthRedirectURIs := getEnv("OAUTH_REDIRECT_URI", "")
	oauthAllowedRedirects := getEnv("OAUTH_ALLOWED_REDIRECT_URIS", "")

	// Prioritize OAUTH_REDIRECT_URI, but warn if both are set with different values
	if oauthRedirectURIs != "" && oauthAllowedRedirects != "" && oauthRedirectURIs != oauthAllowedRedirects {
		log.Printf("WARNING: Both OAUTH_REDIRECT_URI and OAUTH_ALLOWED_REDIRECT_URIS are set with different values. Using OAUTH_REDIRECT_URI: %s", oauthRedirectURIs)
	} else if oauthRedirectURIs == "" {
		oauthRedirectURIs = oauthAllowedRedirects
	}

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

	// Parse allowlist configuration
	allowedCatalogs := parseAllowlist(getEnv("TRINO_ALLOWED_CATALOGS", ""))
	allowedSchemas := parseAllowlist(getEnv("TRINO_ALLOWED_SCHEMAS", ""))
	allowedTables := parseAllowlist(getEnv("TRINO_ALLOWED_TABLES", ""))

	// Validate allowlist formats
	if err := validateAllowlist("TRINO_ALLOWED_SCHEMAS", allowedSchemas, 1); err != nil { // Must have catalog.schema format
		return nil, err
	}
	if err := validateAllowlist("TRINO_ALLOWED_TABLES", allowedTables, 2); err != nil { // Must have catalog.schema.table format
		return nil, err
	}

	// If using HTTPS, force SSL to true
	if strings.EqualFold(scheme, "https") {
		ssl = true
	}

	// Log a warning if write queries are allowed
	if allowWriteQueries {
		log.Println("WARNING: Write queries are enabled (TRINO_ALLOW_WRITE_QUERIES=true). SQL injection protection is bypassed.")
	}

	// Validate OAuth mode
	validModes := map[string]bool{"native": true, "proxy": true}
	if !validModes[oauthMode] {
		return nil, fmt.Errorf("invalid OAuth mode '%s'. Supported modes: native, proxy", oauthMode)
	}

	// Validate and log OAuth mode status
	if oauthEnabled {
		// Validate OAuth provider
		validProviders := map[string]bool{"hmac": true, "okta": true, "google": true, "azure": true}
		if !validProviders[oauthProvider] {
			return nil, fmt.Errorf("invalid OAuth provider '%s'. Supported providers: hmac, okta, google, azure", oauthProvider)
		}

		log.Printf("INFO: OAuth 2.1 authentication enabled (mode: %s, provider: %s)", oauthMode, oauthProvider)
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

		// Validate proxy mode specific requirements
		if oauthMode == "proxy" {
			if oauthProvider != "hmac" && oidcClientSecret == "" {
				log.Printf("WARNING: OIDC_CLIENT_SECRET not set for proxy mode with %s provider. OAuth flow may fail.", oauthProvider)
			}
			if oauthRedirectURIs == "" {
				log.Printf("WARNING: No OAuth redirect URIs configured for proxy mode. All redirects will be rejected for security.")
			}
		}

		if oauthProvider != "hmac" {
			log.Printf("INFO: JWT audience validation enabled for: %s", oidcAudience)
		}
		if oauthRedirectURIs != "" {
			log.Printf("INFO: OAuth redirect URIs configured: %s", oauthRedirectURIs)
		}
	} else {
		log.Println("INFO: OAuth authentication is disabled (OAUTH_ENABLED=false). Enable OAuth for production deployments.")
	}

	// Log allowlist configuration
	logAllowlistConfiguration(allowedCatalogs, allowedSchemas, allowedTables)

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
		OAuthMode:         oauthMode,
		OAuthProvider:     oauthProvider,
		JWTSecret:         jwtSecret,
		OIDCIssuer:        oidcIssuer,
		OIDCAudience:      oidcAudience,
		OIDCClientID:      oidcClientID,
		OIDCClientSecret:     oidcClientSecret,
		OAuthRedirectURIs:    oauthRedirectURIs,
		AllowedCatalogs:   allowedCatalogs,
		AllowedSchemas:    allowedSchemas,
		AllowedTables:     allowedTables,
	}, nil
}

// parseAllowlist parses a comma-separated allowlist from an environment variable
func parseAllowlist(value string) []string {
	if value == "" {
		return nil
	}

	// Split by comma and clean up entries
	items := strings.Split(value, ",")
	var result []string
	for _, item := range items {
		cleaned := strings.TrimSpace(item)
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

// validateAllowlist validates the format of allowlist entries
func validateAllowlist(envVar string, allowlist []string, expectedDots int) error {
	for _, item := range allowlist {
		dots := strings.Count(item, ".")
		if dots != expectedDots {
			return fmt.Errorf("invalid format in %s: '%s' (expected %d dots, found %d)",
				envVar, item, expectedDots, dots)
		}
	}
	return nil
}

// logAllowlistConfiguration logs the current allowlist configuration
func logAllowlistConfiguration(catalogs, schemas, tables []string) {
	if len(catalogs) > 0 || len(schemas) > 0 || len(tables) > 0 {
		log.Println("INFO: Trino allowlist configuration:")
		if len(catalogs) > 0 {
			log.Printf("  - Allowed catalogs: %s (%d configured)", strings.Join(catalogs, ", "), len(catalogs))
		}
		if len(schemas) > 0 {
			log.Printf("  - Allowed schemas: %s (%d configured)", strings.Join(schemas, ", "), len(schemas))
		}
		if len(tables) > 0 {
			log.Printf("  - Allowed tables: %s (%d configured)", strings.Join(tables, ", "), len(tables))
		}
	} else {
		log.Println("INFO: No Trino allowlists configured - all catalogs, schemas, and tables are accessible")
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
