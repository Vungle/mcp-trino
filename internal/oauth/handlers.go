package oauth

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/tuannvm/mcp-trino/internal/config"
)

// OAuth2Handler handles OAuth2 flows using the standard library
type OAuth2Handler struct {
	config       *OAuth2Config
	oauth2Config *oauth2.Config
}

// GetConfig returns the OAuth2 configuration
func (h *OAuth2Handler) GetConfig() *OAuth2Config {
	return h.config
}

// OAuth2Config holds OAuth2 configuration
type OAuth2Config struct {
	Enabled     bool
	Provider    string
	RedirectURI string

	// OIDC configuration
	Issuer       string
	Audience     string
	ClientID     string
	ClientSecret string

	// Server configuration
	MCPHost string
	MCPPort string
	Scheme  string

	// MCPURL is the full URL of the MCP server, used for the resource endpoint in the OAuth 2.0 Protected Resource Metadata endpoint
	MCPURL string

	// Server version
	Version string
}

// NewOAuth2Handler creates a new OAuth2 handler using the standard library
func NewOAuth2Handler(cfg *OAuth2Config) *OAuth2Handler {
	var endpoint oauth2.Endpoint

	// Use OIDC discovery for supported providers, fallback to hardcoded for others
	switch cfg.Provider {
	case "okta", "google", "azure":
		// Use OIDC discovery to get correct endpoints
		if discoveredEndpoint, err := discoverOIDCEndpoints(cfg.Issuer); err != nil {
			log.Printf("Warning: OIDC discovery failed for %s, using fallback endpoints: %v", cfg.Provider, err)
			// Fallback to Okta-style endpoints as they're most common
			endpoint = oauth2.Endpoint{
				AuthURL:  cfg.Issuer + "/oauth2/v1/authorize",
				TokenURL: cfg.Issuer + "/oauth2/v1/token",
			}
		} else {
			endpoint = discoveredEndpoint
		}
	default:
		// For HMAC and unknown providers, use hardcoded endpoints
		endpoint = oauth2.Endpoint{
			AuthURL:  cfg.Issuer + "/oauth2/v1/authorize",
			TokenURL: cfg.Issuer + "/oauth2/v1/token",
		}
	}

	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     endpoint,
		Scopes:       []string{"openid", "profile", "email"},
	}

	return &OAuth2Handler{
		config:       cfg,
		oauth2Config: oauth2Config,
	}
}

// discoverOIDCEndpoints uses OIDC discovery to get the correct authorization and token endpoints
func discoverOIDCEndpoints(issuer string) (oauth2.Endpoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure HTTP client with appropriate timeouts and TLS settings
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Verify TLS certificates
				MinVersion:         tls.VersionTLS12,
			},
			IdleConnTimeout:     30 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
		},
	}

	// Create OIDC provider with custom HTTP client
	provider, err := oidc.NewProvider(
		oidc.ClientContext(ctx, httpClient),
		issuer,
	)
	if err != nil {
		return oauth2.Endpoint{}, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	// Return the discovered endpoint
	return provider.Endpoint(), nil
}

// NewOAuth2ConfigFromTrinoConfig creates OAuth2 config from Trino config
func NewOAuth2ConfigFromTrinoConfig(trinoConfig *config.TrinoConfig, version string) *OAuth2Config {
	mcpHost := getEnv("MCP_HOST", "localhost")
	mcpPort := getEnv("MCP_PORT", "8080")

	// Determine scheme based on HTTPS configuration
	scheme := "http"
	if getEnv("HTTPS_CERT_FILE", "") != "" && getEnv("HTTPS_KEY_FILE", "") != "" {
		scheme = "https"
	}

	mcpURL := getEnv("MCP_URL", fmt.Sprintf("%s://%s:%s", scheme, mcpHost, mcpPort))

	return &OAuth2Config{
		Enabled:      trinoConfig.OAuthEnabled,
		Provider:     trinoConfig.OAuthProvider,
		RedirectURI:  trinoConfig.OAuthRedirectURI,
		Issuer:       trinoConfig.OIDCIssuer,
		Audience:     trinoConfig.OIDCAudience,
		ClientID:     trinoConfig.OIDCClientID,
		ClientSecret: trinoConfig.OIDCClientSecret,
		MCPHost:      mcpHost,
		MCPPort:      mcpPort,
		MCPURL:       mcpURL,
		Scheme:       scheme,
		Version:      version,
	}
}

// HandleAuthorize handles OAuth2 authorization requests with PKCE
func (h *OAuth2Handler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract query parameters
	query := r.URL.Query()

	// PKCE parameters from client
	codeChallenge := query.Get("code_challenge")
	codeChallengeMethod := query.Get("code_challenge_method")
	clientRedirectURI := query.Get("redirect_uri")
	state := query.Get("state")
	clientID := query.Get("client_id")

	log.Printf("OAuth2: Authorization request - client_id: %s, redirect_uri: %s, code_challenge: %s",
		clientID, clientRedirectURI, truncateString(codeChallenge, 10))

	// Set redirect URI - use fixed URI if configured, otherwise use client's URI
	redirectURI := clientRedirectURI
	if h.config.RedirectURI != "" {
		redirectURI = h.config.RedirectURI
		log.Printf("OAuth2: Using fixed redirect URI: %s (overriding client's %s)", redirectURI, clientRedirectURI)
	}

	// Update OAuth2 config with redirect URI
	h.oauth2Config.RedirectURL = redirectURI

	// Create authorization URL with PKCE
	authURL := h.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)

	// Add PKCE parameters to the URL
	if codeChallenge != "" {
		parsedURL, err := url.Parse(authURL)
		if err != nil {
			log.Printf("OAuth2: Failed to parse auth URL: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		query := parsedURL.Query()
		query.Set("code_challenge", codeChallenge)
		query.Set("code_challenge_method", codeChallengeMethod)

		// If using fixed redirect URI, encode original client URI in state
		if h.config.RedirectURI != "" {
			// Encode state and redirect URI safely using JSON + base64
			stateData := map[string]string{
				"state":    state,
				"redirect": clientRedirectURI,
			}
			jsonData, err := json.Marshal(stateData)
			if err != nil {
				log.Printf("OAuth2: Failed to encode state data: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			encodedState := base64.URLEncoding.EncodeToString(jsonData)
			query.Set("state", encodedState)
			log.Printf("OAuth2: Encoded state for proxy callback (length: %d)", len(encodedState))
		}

		parsedURL.RawQuery = query.Encode()
		authURL = parsedURL.String()
	}

	log.Printf("OAuth2: Redirecting to authorization URL: %s", authURL)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// HandleCallback handles OAuth2 callback
func (h *OAuth2Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")

	log.Printf("OAuth2: Callback received - code: %s, state: %s, error: %s",
		truncateString(code, 10), state, errorParam)

	// Handle OAuth errors
	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		log.Printf("OAuth2: Authorization error: %s - %s", errorParam, errorDesc)
		http.Error(w, fmt.Sprintf("Authorization failed: %s", errorDesc), http.StatusBadRequest)
		return
	}

	if code == "" {
		log.Printf("OAuth2: No authorization code received")
		http.Error(w, "No authorization code received", http.StatusBadRequest)
		return
	}

	// If using fixed redirect URI, handle proxy callback
	if h.config.RedirectURI != "" {
		// Try to decode the state parameter
		jsonData, err := base64.URLEncoding.DecodeString(state)
		if err == nil {
			var stateData map[string]string
			if err := json.Unmarshal(jsonData, &stateData); err == nil {
				if originalState, ok := stateData["state"]; ok {
					if originalRedirectURI, ok := stateData["redirect"]; ok {
						log.Printf("OAuth2: Proxying callback to original client: %s", originalRedirectURI)

						// Build proxy callback URL
						proxyURL := fmt.Sprintf("%s?code=%s&state=%s", originalRedirectURI, code, originalState)
						http.Redirect(w, r, proxyURL, http.StatusFound)
						return
					}
				}
			}
		}

		// Fallback: try legacy pipe-delimited format for backward compatibility
		if strings.Contains(state, "|") {
			parts := strings.SplitN(state, "|", 2)
			if len(parts) == 2 {
				// Validate that neither part contains additional pipes
				if strings.Contains(parts[0], "|") || strings.Contains(parts[1], "|") {
					log.Printf("OAuth2: Invalid legacy state format detected")
					h.showSuccessPage(w, code, state)
					return
				}
				originalState := parts[0]
				originalRedirectURI := parts[1]

				log.Printf("OAuth2: Proxying callback to original client (legacy format): %s", originalRedirectURI)

				// Build proxy callback URL
				proxyURL := fmt.Sprintf("%s?code=%s&state=%s", originalRedirectURI, code, originalState)
				http.Redirect(w, r, proxyURL, http.StatusFound)
				return
			}
		}
	}

	// Fallback: show success page
	h.showSuccessPage(w, code, state)
}

// HandleToken handles OAuth2 token exchange
func (h *OAuth2Handler) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("OAuth2: Token exchange request from %s", r.RemoteAddr)

	// Parse form data
	if err := r.ParseForm(); err != nil {
		log.Printf("OAuth2: Failed to parse form: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Extract parameters
	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	clientRedirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeVerifier := r.FormValue("code_verifier")

	log.Printf("OAuth2: Token request - grant_type: %s, client_id: %s, redirect_uri: %s, code: %s",
		grantType, clientID, clientRedirectURI, truncateString(code, 10))

	// Validate parameters
	if code == "" {
		log.Printf("OAuth2: Missing authorization code")
		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	if grantType != "authorization_code" {
		log.Printf("OAuth2: Unsupported grant type: %s", grantType)
		http.Error(w, "Unsupported grant type", http.StatusBadRequest)
		return
	}

	// Set redirect URI for token exchange
	redirectURI := clientRedirectURI
	if h.config.RedirectURI != "" {
		redirectURI = h.config.RedirectURI
		log.Printf("OAuth2: Token exchange using fixed redirect URI: %s", redirectURI)
	}

	h.oauth2Config.RedirectURL = redirectURI

	// For PKCE, we need to manually add the code_verifier to the token exchange
	// Since oauth2 library doesn't support PKCE directly, we'll use a custom approach
	ctx := context.Background()

	// Create custom HTTP client for token exchange with PKCE
	if codeVerifier != "" {
		// Create a custom client that adds code_verifier to the token request
		customClient := &http.Client{
			Transport: &pkceTransport{
				base:         http.DefaultTransport,
				codeVerifier: codeVerifier,
			},
		}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, customClient)
	}

	// Exchange code for tokens
	token, err := h.oauth2Config.Exchange(ctx, code)
	if err != nil {
		log.Printf("OAuth2: Token exchange failed: %v", err)
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	log.Printf("OAuth2: Token exchange successful")

	// Build response
	response := map[string]interface{}{
		"access_token": token.AccessToken,
		"token_type":   token.TokenType,
		"expires_in":   int(time.Until(token.Expiry).Seconds()),
	}

	// Add optional fields
	if token.RefreshToken != "" {
		response["refresh_token"] = token.RefreshToken
	}

	// Add ID token if present
	if idToken, ok := token.Extra("id_token").(string); ok {
		response["id_token"] = idToken
	}

	// Add scope if present
	if scope, ok := token.Extra("scope").(string); ok {
		response["scope"] = scope
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("OAuth2: Failed to encode token response: %v", err)
	}
}

// showSuccessPage displays a success page after OAuth completion
func (h *OAuth2Handler) showSuccessPage(w http.ResponseWriter, code, state string) {
	// Log authorization details server-side (truncated for security)
	log.Printf("OAuth2: Authorization successful - code: %s, state: %s",
		truncateString(code, 10), truncateString(state, 10))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, `<!DOCTYPE html>
		<html lang="en">
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<title>OAuth2 Success</title>
		</head>
		<body>
			<h2>Authentication Successful!</h2>
			<p>You have been successfully authenticated.</p>
			<p>You can now close this window and return to your application.</p>
		</body>
		</html>`)
}

// truncateString safely truncates a string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// pkceTransport adds PKCE code_verifier to token exchange requests
type pkceTransport struct {
	base         http.RoundTripper
	codeVerifier string
}

// RoundTrip implements the RoundTripper interface
func (p *pkceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Only modify POST requests to token endpoint
	if req.Method == "POST" && strings.Contains(req.URL.Path, "/token") {
		// Read the existing body
		defer func() {
			if closeErr := req.Body.Close(); closeErr != nil {
				log.Printf("Warning: failed to close request body: %v", closeErr)
			}
		}()
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		// Parse the form data
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}

		// Add code_verifier if not already present
		if values.Get("code_verifier") == "" && p.codeVerifier != "" {
			values.Set("code_verifier", p.codeVerifier)
		}

		// Create new body with code_verifier
		newBody := strings.NewReader(values.Encode())
		req.Body = io.NopCloser(newBody)
		req.ContentLength = int64(len(values.Encode()))
	}

	return p.base.RoundTrip(req)
}

// getEnv gets environment variable with default value
func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}
