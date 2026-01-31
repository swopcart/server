package session

import (
	"crypto/rand"
	"crypto/rsa"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/config"
)

// testServiceForToken creates a minimal SessionService for token-only tests.
// These tests don't require database access.
func testServiceForToken(t *testing.T) *SessionService {
	t.Helper()

	cfg := &config.Config{
		Auth: config.Auth{
			AccessTokenTTL:  300,
			RefreshTokenTTL: 90,
			JWTIssuer:       "test-issuer",
		},
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	return &SessionService{
		config: cfg,
		logger: logger,
		jwtKeys: keyPair{
			private: privateKey,
			public:  &privateKey.PublicKey,
		},
	}
}

func TestGenerateAccessToken(t *testing.T) {
	svc := testServiceForToken(t)
	sessionUUID := uuid.New()
	userUUID := uuid.New()
	username := "testuser"
	admin := true

	token, err := svc.generateAccessToken(sessionUUID, userUUID, username, admin)
	if err != nil {
		t.Fatalf("generateAccessToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Parse and verify the token
	parsedToken, err := jwt.ParseWithClaims(token, &AccessTokenClaims{}, func(token *jwt.Token) (any, error) {
		return svc.jwtKeys.public, nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, ok := parsedToken.Claims.(*AccessTokenClaims)
	if !ok {
		t.Fatal("failed to cast claims")
	}

	if claims.SessionUUID != sessionUUID {
		t.Errorf("expected session UUID %v, got %v", sessionUUID, claims.SessionUUID)
	}

	if claims.UserUUID != userUUID {
		t.Errorf("expected user UUID %v, got %v", userUUID, claims.UserUUID)
	}

	if claims.Username != username {
		t.Errorf("expected username %q, got %q", username, claims.Username)
	}

	if claims.Admin != admin {
		t.Errorf("expected admin %v, got %v", admin, claims.Admin)
	}

	if claims.Subject != userUUID.String() {
		t.Errorf("expected subject %q, got %q", userUUID.String(), claims.Subject)
	}

	if claims.Issuer != svc.config.Auth.JWTIssuer {
		t.Errorf("expected issuer %q, got %q", svc.config.Auth.JWTIssuer, claims.Issuer)
	}

	// Verify expiry is approximately AccessTokenTTL seconds from now
	expectedExpiry := time.Now().Add(time.Duration(svc.config.Auth.AccessTokenTTL) * time.Second)
	actualExpiry := claims.ExpiresAt.Time
	diff := expectedExpiry.Sub(actualExpiry)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, actualExpiry)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	svc := testServiceForToken(t)
	sessionUUID := uuid.New()

	token, err := svc.generateRefreshToken(sessionUUID)
	if err != nil {
		t.Fatalf("generateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Parse and verify the token
	parsedToken, err := jwt.ParseWithClaims(token, &RefreshTokenClaims{}, func(token *jwt.Token) (any, error) {
		return svc.jwtKeys.public, nil
	})
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	claims, ok := parsedToken.Claims.(*RefreshTokenClaims)
	if !ok {
		t.Fatal("failed to cast claims")
	}

	if claims.SessionUUID != sessionUUID {
		t.Errorf("expected session UUID %v, got %v", sessionUUID, claims.SessionUUID)
	}

	if claims.Issuer != svc.config.Auth.JWTIssuer {
		t.Errorf("expected issuer %q, got %q", svc.config.Auth.JWTIssuer, claims.Issuer)
	}

	if claims.ID != sessionUUID.String() {
		t.Errorf("expected JWT ID %q, got %q", sessionUUID.String(), claims.ID)
	}

	// Verify expiry is approximately RefreshTokenTTL days from now
	expectedExpiry := time.Now().Add(time.Duration(svc.config.Auth.RefreshTokenTTL) * 24 * time.Hour)
	actualExpiry := claims.ExpiresAt.Time
	diff := expectedExpiry.Sub(actualExpiry)
	if diff < -2*time.Second || diff > 2*time.Second {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, actualExpiry)
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		length int
	}{
		{
			name:   "simple token",
			token:  "test-token",
			length: 64, // SHA256 produces 32 bytes = 64 hex chars
		},
		{
			name:   "empty token",
			token:  "",
			length: 64,
		},
		{
			name:   "long token",
			token:  "this-is-a-very-long-token-that-simulates-a-real-jwt-token-with-many-characters",
			length: 64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashToken(tt.token)

			if len(hash) != tt.length {
				t.Errorf("expected hash length %d, got %d", tt.length, len(hash))
			}
		})
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	token := "test-token-12345"

	hash1 := hashToken(token)
	hash2 := hashToken(token)

	if hash1 != hash2 {
		t.Error("expected same token to produce same hash")
	}
}

func TestHashTokenUnique(t *testing.T) {
	hash1 := hashToken("token-a")
	hash2 := hashToken("token-b")

	if hash1 == hash2 {
		t.Error("expected different tokens to produce different hashes")
	}
}

func TestJwtKeyFunc(t *testing.T) {
	svc := testServiceForToken(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	keyFunc := svc.jwtKeyFunc(logger)

	t.Run("valid RSA signing method", func(t *testing.T) {
		token := &jwt.Token{
			Method: jwt.SigningMethodRS256,
		}

		key, err := keyFunc(token)
		if err != nil {
			t.Fatalf("keyFunc failed: %v", err)
		}

		if key != svc.jwtKeys.public {
			t.Error("expected public key to be returned")
		}
	})

	t.Run("invalid signing method", func(t *testing.T) {
		token := &jwt.Token{
			Method: jwt.SigningMethodHS256,
		}

		_, err := keyFunc(token)
		if err != ErrInvalidToken {
			t.Errorf("expected ErrInvalidToken, got %v", err)
		}
	})
}

func TestAccessTokenClaims(t *testing.T) {
	sessionUUID := uuid.New()
	userUUID := uuid.New()

	claims := AccessTokenClaims{
		SessionUUID: sessionUUID,
		UserUUID:    userUUID,
		Username:    "testuser",
		Admin:       true,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID.String(),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "test",
		},
	}

	if claims.SessionUUID != sessionUUID {
		t.Errorf("expected SessionUUID %v, got %v", sessionUUID, claims.SessionUUID)
	}

	if claims.UserUUID != userUUID {
		t.Errorf("expected UserUUID %v, got %v", userUUID, claims.UserUUID)
	}

	if claims.Username != "testuser" {
		t.Errorf("expected Username %q, got %q", "testuser", claims.Username)
	}

	if claims.Admin != true {
		t.Errorf("expected Admin %v, got %v", true, claims.Admin)
	}
}

func TestRefreshTokenClaims(t *testing.T) {
	sessionUUID := uuid.New()

	claims := RefreshTokenClaims{
		SessionUUID: sessionUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(90 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        sessionUUID.String(),
			Issuer:    "test",
		},
	}

	if claims.SessionUUID != sessionUUID {
		t.Errorf("expected SessionUUID %v, got %v", sessionUUID, claims.SessionUUID)
	}

	if claims.ID != sessionUUID.String() {
		t.Errorf("expected ID %q, got %q", sessionUUID.String(), claims.ID)
	}
}

func TestTokenPair(t *testing.T) {
	pair := TokenPair{
		AccessToken:  "access-token-value",
		RefreshToken: "refresh-token-value",
	}

	if pair.AccessToken != "access-token-value" {
		t.Errorf("expected AccessToken %q, got %q", "access-token-value", pair.AccessToken)
	}

	if pair.RefreshToken != "refresh-token-value" {
		t.Errorf("expected RefreshToken %q, got %q", "refresh-token-value", pair.RefreshToken)
	}
}

func TestGetSessionDataFromAccessToken(t *testing.T) {
	svc := testServiceForToken(t)
	sessionUUID := uuid.MustParse("0b0676d1-9335-4f35-9f30-a84dfb795168")
	userUUID := uuid.MustParse("2ee2ac40-de2e-4536-87e3-6fb804bb5d18")
	username := "admin"
	admin := true

	// Generate a token
	token, err := svc.generateAccessToken(sessionUUID, userUUID, username, admin)
	if err != nil {
		t.Fatalf("generateAccessToken failed: %v", err)
	}

	// Parse it back using GetSessionDataFromAccessToken
	sessionData, err := svc.GetSessionDataFromAccessToken(token)
	if err != nil {
		t.Fatalf("GetSessionDataFromAccessToken failed: %v", err)
	}

	// Verify the fields
	if sessionData.SessionUUID != sessionUUID {
		t.Errorf("SessionUUID mismatch: got %v, want %v", sessionData.SessionUUID, sessionUUID)
	}

	if sessionData.UserUUID != userUUID {
		t.Errorf("UserUUID mismatch: got %v (zero: %v), want %v",
			sessionData.UserUUID, uuid.UUID{}, userUUID)
	}

	if sessionData.Admin != admin {
		t.Errorf("Admin mismatch: got %v, want %v", sessionData.Admin, admin)
	}
}
