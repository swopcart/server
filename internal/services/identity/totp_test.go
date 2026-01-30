package identity_test

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/testkit"
)

func TestGenerateTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	assert.NotEmpty(t, pending.Secret, "secret should be generated")
	assert.NotEmpty(t, pending.URL, "URL should be generated")
	assert.Contains(t, pending.URL, "testuser", "URL should contain username")
	assert.Contains(t, pending.URL, identity.TOTPIssuer, "URL should contain issuer")
	assert.False(t, pending.ExpiresAt.IsZero(), "expiration should be set")
	assert.True(t, time.Now().Before(pending.ExpiresAt), "should not be expired")
}

func TestGetPendingTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	// Should be nil initially
	pending := id.GetPendingTOTP(user.UUID())
	assert.Nil(t, pending, "should have no pending TOTP initially")

	// Generate TOTP
	generated, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Should retrieve the pending TOTP
	pending = id.GetPendingTOTP(user.UUID())
	if assert.NotNil(t, pending, "should have pending TOTP after generation") {
		assert.Equal(t, generated.Secret, pending.Secret)
		assert.Equal(t, generated.URL, pending.URL)
	}
}

func TestClearPendingTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	_, err = id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Clear pending TOTP
	id.ClearPendingTOTP(user.UUID())

	// Should be nil after clearing
	pending := id.GetPendingTOTP(user.UUID())
	assert.Nil(t, pending, "should have no pending TOTP after clearing")
}

func TestEnableTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Generate a valid token at a specific time
	now := time.Now()
	token, err := totp.GenerateCode(pending.Secret, now)
	if !assert.NoError(t, err) {
		return
	}

	// Enable TOTP with the valid token
	err = user.EnableTOTPAtTime(t.Context(), pending.Secret, token, &now)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, user.HasTOTP(), "TOTP should be enabled")
}

func TestEnableTOTP_InvalidToken(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Try to enable with an invalid token
	err = user.EnableTOTP(t.Context(), pending.Secret, "000000")
	assert.ErrorIs(t, err, identity.ErrInvalidTOTP, "should reject invalid token")
	assert.False(t, user.HasTOTP(), "TOTP should not be enabled")
}

func TestCheckTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Enable TOTP
	now := time.Now()
	token, err := totp.GenerateCode(pending.Secret, now)
	if !assert.NoError(t, err) {
		return
	}

	err = user.EnableTOTPAtTime(t.Context(), pending.Secret, token, &now)
	if !assert.NoError(t, err) {
		return
	}

	// Generate a new token and verify it
	checkTime := now.Add(35 * time.Second) // Move to next time window
	checkToken, err := totp.GenerateCode(pending.Secret, checkTime)
	if !assert.NoError(t, err) {
		return
	}

	err = user.CheckTOTPAtTime(checkToken, &checkTime)
	assert.NoError(t, err, "valid token should be accepted")
}

func TestCheckTOTP_InvalidToken(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Enable TOTP
	now := time.Now()
	token, err := totp.GenerateCode(pending.Secret, now)
	if !assert.NoError(t, err) {
		return
	}

	err = user.EnableTOTPAtTime(t.Context(), pending.Secret, token, &now)
	if !assert.NoError(t, err) {
		return
	}

	// Try invalid token
	err = user.CheckTOTP("000000")
	assert.ErrorIs(t, err, identity.ErrInvalidTOTP, "invalid token should be rejected")
}

func TestCheckTOTP_NotEnabled(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	// TOTP not enabled, should return nil (no error)
	err = user.CheckTOTP("000000")
	assert.NoError(t, err, "checking TOTP when not enabled should return no error")
}

func TestDisableTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Enable TOTP
	now := time.Now()
	token, err := totp.GenerateCode(pending.Secret, now)
	if !assert.NoError(t, err) {
		return
	}

	err = user.EnableTOTPAtTime(t.Context(), pending.Secret, token, &now)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, user.HasTOTP(), "TOTP should be enabled")

	// Disable TOTP
	err = user.DisableTOTP(t.Context())
	assert.NoError(t, err, "should disable TOTP successfully")
	assert.False(t, user.HasTOTP(), "TOTP should be disabled")
}

func TestDisableTOTP_NotEnabled(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	// Try to disable when not enabled
	err = user.DisableTOTP(t.Context())
	assert.ErrorIs(t, err, identity.ErrTOTPNotEnabled, "should error when TOTP not enabled")
}

func TestHasTOTP(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	// Initially should not have TOTP
	assert.False(t, user.HasTOTP(), "should not have TOTP initially")

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Enable TOTP
	now := time.Now()
	token, err := totp.GenerateCode(pending.Secret, now)
	if !assert.NoError(t, err) {
		return
	}

	err = user.EnableTOTPAtTime(t.Context(), pending.Secret, token, &now)
	if !assert.NoError(t, err) {
		return
	}

	// Should have TOTP after enabling
	assert.True(t, user.HasTOTP(), "should have TOTP after enabling")
}

func TestTOTP_TimeSkew(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	pending, err := id.GenerateTOTP(user.UUID(), user.Username())
	if !assert.NoError(t, err) {
		return
	}

	// Enable TOTP
	now := time.Now()
	token, err := totp.GenerateCode(pending.Secret, now)
	if !assert.NoError(t, err) {
		return
	}

	err = user.EnableTOTPAtTime(t.Context(), pending.Secret, token, &now)
	if !assert.NoError(t, err) {
		return
	}

	// Test with token from previous time window (should work with skew=1)
	previousTime := now.Add(-30 * time.Second)
	previousToken, err := totp.GenerateCode(pending.Secret, previousTime)
	if !assert.NoError(t, err) {
		return
	}

	err = user.CheckTOTPAtTime(previousToken, &now)
	assert.NoError(t, err, "token from previous window should be accepted with skew")

	// Test with token from next time window (should work with skew=1)
	nextTime := now.Add(30 * time.Second)
	nextToken, err := totp.GenerateCode(pending.Secret, nextTime)
	if !assert.NoError(t, err) {
		return
	}

	err = user.CheckTOTPAtTime(nextToken, &now)
	assert.NoError(t, err, "token from next window should be accepted with skew")

	// Test with token from 2 windows away (should fail with skew=1)
	farTime := now.Add(60 * time.Second)
	farToken, err := totp.GenerateCode(pending.Secret, farTime)
	if !assert.NoError(t, err) {
		return
	}

	err = user.CheckTOTPAtTime(farToken, &now)
	assert.ErrorIs(t, err, identity.ErrInvalidTOTP, "token from 2 windows away should be rejected")
}
