package oauth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tuannvm/mcp-trino/internal/config"
)

// TestHMACValidator_AudienceValidation tests JWT audience validation
func TestHMACValidator_AudienceValidation(t *testing.T) {
	// Test configuration
	cfg := &config.TrinoConfig{
		JWTSecret:    "test-secret-key-for-hmac-validation",
		OIDCAudience: "test-service-audience",
	}
	
	validator := &HMACValidator{}
	err := validator.Initialize(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	t.Run("ValidAudience", func(t *testing.T) {
		// Create token with correct audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "test-user",
			"aud":   "test-service-audience",
			"exp":   time.Now().Add(time.Hour).Unix(),
			"iat":   time.Now().Unix(),
			"email": "test@example.com",
		})
		
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		
		user, err := validator.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("Expected valid token to pass, got error: %v", err)
		}
		
		if user == nil || user.Subject != "test-user" {
			t.Errorf("Expected valid user, got: %+v", user)
		}
	})

	t.Run("InvalidAudience", func(t *testing.T) {
		// Create token with wrong audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"aud": "wrong.audience.com", // Wrong audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		
		_, err = validator.ValidateToken(tokenString)
		if err == nil {
			t.Error("Expected token with wrong audience to fail validation")
		}
		
		if err != nil && err.Error() != "audience validation failed: invalid audience: expected test-service-audience, got wrong.audience.com" {
			t.Errorf("Expected specific audience error, got: %v", err)
		}
	})

	t.Run("MissingAudience", func(t *testing.T) {
		// Create token without audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		
		_, err = validator.ValidateToken(tokenString)
		if err == nil {
			t.Error("Expected token without audience to fail validation")
		}
		
		if err != nil && err.Error() != "audience validation failed: missing audience claim" {
			t.Errorf("Expected missing audience error, got: %v", err)
		}
	})

	t.Run("AudienceArray", func(t *testing.T) {
		// Create token with audience as array (valid)
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"aud": []string{"test-service-audience", "other.service.com"}, // Array with correct audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		
		user, err := validator.ValidateToken(tokenString)
		if err != nil {
			t.Errorf("Expected token with correct audience in array to pass, got error: %v", err)
		}
		
		if user == nil || user.Subject != "test-user" {
			t.Errorf("Expected valid user, got: %+v", user)
		}
	})

	t.Run("AudienceArrayInvalid", func(t *testing.T) {
		// Create token with audience array not containing expected audience
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "test-user",
			"aud": []string{"wrong.service.com", "other.service.com"}, // Array without correct audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
		})
		
		tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign token: %v", err)
		}
		
		_, err = validator.ValidateToken(tokenString)
		if err == nil {
			t.Error("Expected token with wrong audience array to fail validation")
		}
		
		if err != nil && err.Error() != "audience validation failed: invalid audience: expected test-service-audience not found in audience list" {
			t.Errorf("Expected specific audience array error, got: %v", err)
		}
	})
}

// TestHMACValidator_InitializationValidation tests validator initialization
func TestHMACValidator_InitializationValidation(t *testing.T) {
	t.Run("MissingSecret", func(t *testing.T) {
		cfg := &config.TrinoConfig{
			JWTSecret:    "", // Missing secret
			OIDCAudience: "test-service-audience",
		}
		
		validator := &HMACValidator{}
		err := validator.Initialize(cfg)
		
		if err == nil {
			t.Error("Expected initialization to fail with missing secret")
		}
		
		if err != nil && err.Error() != "JWT_SECRET is required for HMAC provider" {
			t.Errorf("Expected specific secret error, got: %v", err)
		}
	})

	t.Run("MissingAudience", func(t *testing.T) {
		cfg := &config.TrinoConfig{
			JWTSecret:    "test-secret",
			OIDCAudience: "", // Missing audience
		}
		
		validator := &HMACValidator{}
		err := validator.Initialize(cfg)
		
		if err == nil {
			t.Error("Expected initialization to fail with missing audience")
		}
		
		if err != nil && err.Error() != "JWT audience is required for HMAC provider" {
			t.Errorf("Expected specific audience error, got: %v", err)
		}
	})

	t.Run("ValidConfiguration", func(t *testing.T) {
		cfg := &config.TrinoConfig{
			JWTSecret:    "test-secret",
			OIDCAudience: "test-service-audience",
		}
		
		validator := &HMACValidator{}
		err := validator.Initialize(cfg)
		
		if err != nil {
			t.Errorf("Expected valid configuration to succeed, got error: %v", err)
		}
		
		if validator.secret != "test-secret" {
			t.Errorf("Expected secret to be set correctly")
		}
		
		if validator.audience != "test-service-audience" {
			t.Errorf("Expected audience to be set correctly")
		}
	})
}

// TestHMACValidator_SecurityValidation tests that the vulnerability is fixed
func TestHMACValidator_SecurityValidation(t *testing.T) {
	// This test specifically validates that the vulnerability described in PE-7429 is fixed
	
	t.Run("RejectCrossServiceToken", func(t *testing.T) {
		cfg := &config.TrinoConfig{
			JWTSecret:    "test-secret",
			OIDCAudience: "test-service-audience",
		}
		
		validator := &HMACValidator{}
		err := validator.Initialize(cfg)
		if err != nil {
			t.Fatalf("Failed to initialize validator: %v", err)
		}
		
		// Simulate a token from another service (different audience)
		crossServiceToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "cross-service-user",
			"aud": "other.service.com", // Different service audience
			"exp": time.Now().Add(time.Hour).Unix(),
			"iat": time.Now().Unix(),
			"iss": "company.okta.com", // Same issuer
		})
		
		tokenString, err := crossServiceToken.SignedString([]byte(cfg.JWTSecret))
		if err != nil {
			t.Fatalf("Failed to sign cross-service token: %v", err)
		}
		
		// This should FAIL - the vulnerability would allow this to pass
		_, err = validator.ValidateToken(tokenString)
		if err == nil {
			t.Error("SECURITY VULNERABILITY: Cross-service token was accepted! This should fail.")
		}
		
		// Verify it fails for the correct reason (audience validation)
		if err != nil && !strings.Contains(err.Error(), "audience validation failed") {
			t.Errorf("Token failed for wrong reason. Expected audience validation failure, got: %v", err)
		}
	})
}

