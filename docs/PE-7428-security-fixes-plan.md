# Security Fixes Implementation Plan - PE-7428

## Overview
Address security vulnerabilities in the mcp-trino codebase identified during security review. This document outlines the implementation plan for fixing critical security issues.

**Jira Ticket:** PE-7428  
**Epic:** PE-6985 (Q3 2025 - Automation Workflow)  
**Split from:** PE-7320 (Security Assessment for Remote MCP Access)

## Security Issues Identified

### 1. JWT Token Logging (middleware.go:129)
**Issue:** JWTs contain user info in cleartext (email, subject identifier, name) and are logged in preview format.

**Current Code:**
```go
// Log token for debugging (first 50 chars)
tokenPreview := tokenString
if len(tokenString) > 50 {
    tokenPreview = tokenString[:50] + "..."
}
log.Printf("OAuth: Validating token for tool %s: %s", req.Params.Name, tokenPreview)
```

**Risk:** Sensitive user information exposure in logs.

**Fix:** Replace token preview with hash/truncated hash for debugging while maintaining security.

### 2. Missing jwtSecret Configuration (config.go:94)
**Issue:** Server starts with warning when jwtSecret is not configured in HMAC mode.

**Current Code:**
```go
if oauthProvider == "hmac" && jwtSecret == "" {
    log.Println("WARNING: JWT_SECRET not set for HMAC provider. Using insecure default for development only.")
}
```

**Risk:** Easy to deploy to production without proper JWT secret, creating security vulnerability.

**Fix:** Fail hard and refuse to start server if jwtSecret isn't set in HMAC mode.

### 3. Trino Password Logging (client.go:42)
**Issue:** Password included in DSN construction could be exposed in connection failure logs.

**Current Code:**
```go
dsn := fmt.Sprintf("%s://%s:%s@%s:%d?catalog=%s&schema=%s&SSL=%t&SSLInsecure=%t",
    cfg.Scheme,
    url.QueryEscape(cfg.User),
    url.QueryEscape(cfg.Password), // <- Password in DSN
    cfg.Host,
    cfg.Port,
    url.QueryEscape(cfg.Catalog),
    url.QueryEscape(cfg.Schema),
    cfg.SSL,
    cfg.SSLInsecure)
```

**Risk:** Possible password exposure in connection error messages.

**Fix:** Sanitize connection error messages to remove sensitive information.

### 4. OAuth Default Configuration (config.go:47)
**Issue:** TRINO_OAUTH_ENABLED defaults to false (insecure mode).

**Current Code:**
```go
oauthEnabled, _ := strconv.ParseBool(getEnv("TRINO_OAUTH_ENABLED", "false"))
```

**Risk:** Default insecure configuration makes it easy to deploy without proper authentication.

**Fix:** Make OAuth the default, require explicit flag/env var to enable insecure mode.

## Implementation Plan

### Phase 1: Fix JWT Token Logging
**File:** `internal/oauth/middleware.go`  
**Lines:** 127-132

1. Replace token preview with SHA256 hash (first 16 chars)
2. Maintain debugging capability without exposing sensitive data
3. Update log message to indicate hash usage

### Phase 2: Add jwtSecret Hard Failure
**File:** `internal/config/config.go`  
**Lines:** 94-96

1. Change warning to error return
2. Prevent server startup when jwtSecret is empty in HMAC mode
3. Update error message to guide proper configuration

### Phase 3: Sanitize Connection Errors
**File:** `internal/trino/client.go`  
**Lines:** 40-43, 51-57

1. Create sanitized error messages for connection failures
2. Strip password from error context
3. Maintain debugging information without exposing credentials

### Phase 4: Change OAuth Defaults
**File:** `internal/config/config.go`  
**Line:** 47

1. Change default from "false" to "true"
2. Add environment variable for explicit insecure mode opt-out
3. Update configuration validation and logging

### Phase 5: Testing and Validation
1. Run existing test suite
2. Test configuration validation
3. Verify error handling
4. Test OAuth flow with new defaults
5. Validate log output security

## Files to Modify

1. `internal/oauth/middleware.go` - JWT token logging fix
2. `internal/config/config.go` - jwtSecret validation and OAuth defaults
3. `internal/trino/client.go` - Password sanitization in error messages

## Testing Strategy

1. **Unit Tests:** Update existing tests for new validation logic
2. **Integration Tests:** Verify OAuth flow with new defaults
3. **Security Tests:** Confirm no sensitive data in logs
4. **Error Handling Tests:** Validate sanitized error messages

## Acceptance Criteria

- [ ] JWT tokens no longer expose user info in logs
- [ ] Server fails to start without jwtSecret in HMAC mode
- [ ] Connection errors don't expose Trino passwords
- [ ] OAuth mode is enabled by default
- [ ] All existing tests pass
- [ ] New security validations are tested

## Risk Assessment

**Low Risk Changes:**
- JWT token logging (no functional impact)
- Error message sanitization (improves security without breaking functionality)

**Medium Risk Changes:**
- OAuth default change (may affect existing deployments)
- jwtSecret validation (prevents insecure deployments)

## Rollback Plan

All changes are backwards compatible through environment variables:
- `TRINO_OAUTH_ENABLED=false` to disable OAuth
- `JWT_SECRET` can be set to enable HMAC mode
- Connection handling remains functionally identical