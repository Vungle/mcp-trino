package oauth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HandleMetadata handles the legacy OAuth metadata endpoint for MCP compliance
func (h *OAuth2Handler) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	remoteAddr := r.RemoteAddr
	userAgent := r.UserAgent()

	log.Printf("OAuth2: Metadata request from %s (User-Agent: %s)", remoteAddr, userAgent)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if r.Method != "GET" {
		log.Printf("OAuth2: Invalid method %s for metadata endpoint from %s", r.Method, remoteAddr)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Return OAuth metadata based on configuration
	if !h.config.Enabled {
		log.Printf("OAuth2: OAuth disabled, returning disabled metadata to %s", remoteAddr)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{
			"oauth_enabled": false,
			"authentication_methods": ["none"],
			"mcp_version": "1.0.0"
		}`)
		return
	}

	// Create provider-specific metadata
	metadata := map[string]interface{}{
		"oauth_enabled":          true,
		"authentication_methods": []string{"bearer_token"},
		"token_types":            []string{"JWT"},
		"token_validation":       "server_side",
		"supported_flows":        []string{"claude_code", "mcp_remote"},
		"mcp_version":            "1.0.0",
		"server_version":         h.config.Version,
		"provider":               h.config.Provider,
		"authorization_endpoint": fmt.Sprintf("%s://%s:%s/oauth/authorize", h.config.Scheme, h.config.MCPHost, h.config.MCPPort),
		"token_endpoint":         h.oauth2Config.Endpoint.TokenURL,
	}

	// Add provider-specific metadata
	switch h.config.Provider {
	case "hmac":
		metadata["validation_method"] = "hmac_sha256"
		metadata["signature_algorithm"] = "HS256"
		metadata["requires_secret"] = true
	case "okta", "google", "azure":
		metadata["validation_method"] = "oidc_jwks"
		metadata["signature_algorithm"] = "RS256"
		metadata["requires_secret"] = false
		if h.config.Issuer != "" {
			metadata["issuer"] = h.config.Issuer
			metadata["jwks_uri"] = h.config.Issuer + "/.well-known/jwks.json"
		}
		if h.config.Audience != "" {
			metadata["audience"] = h.config.Audience
		}
	}

	// Encode and send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		log.Printf("OAuth2: Error encoding metadata for %s: %v", remoteAddr, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	responseTime := time.Since(start).Milliseconds()
	log.Printf("OAuth2: Metadata response sent to %s in %dms", remoteAddr, responseTime)
}

// HandleAuthorizationServerMetadata handles the standard OAuth 2.0 Authorization Server Metadata endpoint
func (h *OAuth2Handler) HandleAuthorizationServerMetadata(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	remoteAddr := r.RemoteAddr
	userAgent := r.UserAgent()

	log.Printf("OAuth2: Authorization Server Metadata request from %s (User-Agent: %s)", remoteAddr, userAgent)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if r.Method != "GET" {
		log.Printf("OAuth2: Invalid method %s for authorization server metadata endpoint from %s", r.Method, remoteAddr)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Return OAuth 2.0 Authorization Server Metadata (RFC 8414)
	metadata := map[string]interface{}{
		"issuer":                                h.config.Issuer,
		"authorization_endpoint":                fmt.Sprintf("%s/oauth2/v1/authorize", h.config.Issuer),
		"token_endpoint":                        fmt.Sprintf("%s/oauth2/v1/token", h.config.Issuer),
		"registration_endpoint":                 fmt.Sprintf("%s/oauth2/v1/clients", h.config.Issuer),
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"code_challenge_methods_supported":      []string{"plain", "S256"},
		"revocation_endpoint":                   fmt.Sprintf("%s/oauth/revoke", h.config.Issuer),
	}

	// Encode and send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		log.Printf("OAuth2: Error encoding Authorization Server metadata for %s: %v", remoteAddr, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	responseTime := time.Since(start).Milliseconds()
	log.Printf("OAuth2: Authorization Server Metadata response sent to %s in %dms", remoteAddr, responseTime)
}

// HandleProtectedResourceMetadata handles the OAuth 2.0 Protected Resource Metadata endpoint
func (h *OAuth2Handler) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	remoteAddr := r.RemoteAddr
	userAgent := r.UserAgent()

	log.Printf("OAuth2: Protected Resource Metadata request from %s (User-Agent: %s)", remoteAddr, userAgent)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes

	if r.Method != "GET" {
		log.Printf("OAuth2: Invalid method %s for protected resource metadata endpoint from %s", r.Method, remoteAddr)
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Return OAuth 2.0 Protected Resource Metadata (RFC 9728)
	metadata := map[string]interface{}{
		"resource":                              h.config.MCPURL,
		"authorization_servers":                 []string{fmt.Sprintf("%s://%s:%s", h.config.Scheme, h.config.MCPHost, h.config.MCPPort)},
		"bearer_methods_supported":              []string{"header"},
		"resource_signing_alg_values_supported": []string{"RS256"},
		"resource_documentation":                fmt.Sprintf("%s://%s:%s/docs", h.config.Scheme, h.config.MCPHost, h.config.MCPPort),
		"resource_policy_uri":                   fmt.Sprintf("%s://%s:%s/policy", h.config.Scheme, h.config.MCPHost, h.config.MCPPort),
		"resource_tos_uri":                      fmt.Sprintf("%s://%s:%s/tos", h.config.Scheme, h.config.MCPHost, h.config.MCPPort),
	}

	// Encode and send response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(metadata); err != nil {
		log.Printf("OAuth2: Error encoding Protected Resource metadata for %s: %v", remoteAddr, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	responseTime := time.Since(start).Milliseconds()
	log.Printf("OAuth2: Protected Resource Metadata response sent to %s in %dms", remoteAddr, responseTime)
}

// HandleRegister handles OAuth dynamic client registration for mcp-remote
func (h *OAuth2Handler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	remoteAddr := r.RemoteAddr
	userAgent := r.UserAgent()

	log.Printf("OAuth2: Client registration request from %s (User-Agent: %s)", remoteAddr, userAgent)

	if r.Method != "POST" {
		log.Printf("OAuth2: Invalid method %s for registration endpoint from %s", r.Method, remoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the registration request
	var regRequest map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&regRequest); err != nil {
		log.Printf("OAuth2: Failed to parse registration request from %s: %v", remoteAddr, err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("OAuth2: Registration request from %s: %+v", remoteAddr, regRequest)

	// Accept any client registration from mcp-remote
	// Return our pre-configured client_id
	response := map[string]interface{}{
		"client_id":                  h.config.ClientID,
		"client_secret":              "", // Public client, no secret
		"client_id_issued_at":        time.Now().Unix(),
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
		"application_type":           "native",
		"client_name":                regRequest["client_name"],
	}

	// Use fixed redirect URI if configured, otherwise use client's redirect URIs
	if h.config.RedirectURI != "" {
		response["redirect_uris"] = []string{h.config.RedirectURI}
		log.Printf("OAuth2: Registration response using fixed redirect URI for %s: %s", remoteAddr, h.config.RedirectURI)
	} else {
		response["redirect_uris"] = regRequest["redirect_uris"]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("OAuth2: Failed to encode registration response for %s: %v", remoteAddr, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	responseTime := time.Since(start).Milliseconds()
	log.Printf("OAuth2: Client registration response sent to %s in %dms", remoteAddr, responseTime)
}

// HandleCallbackRedirect handles the /callback redirect for Claude Code compatibility
func (h *OAuth2Handler) HandleCallbackRedirect(w http.ResponseWriter, r *http.Request) {
	remoteAddr := r.RemoteAddr
	userAgent := r.UserAgent()

	log.Printf("OAuth2: Callback redirect request from %s (User-Agent: %s) - Query: %s", remoteAddr, userAgent, r.URL.RawQuery)

	// Preserve all query parameters when redirecting
	redirectURL := "/oauth/callback"
	if r.URL.RawQuery != "" {
		redirectURL += "?" + r.URL.RawQuery
	}

	log.Printf("OAuth2: Redirecting %s from /callback to %s", remoteAddr, redirectURL)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
