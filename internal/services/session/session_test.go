package session_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/services/session"
	"github.com/swopcart/server/internal/testkit"
)

// testUser creates a test user using the identity service from the harness.
func testUser(t *testing.T, h *testkit.Harness) *identity.User {
	t.Helper()

	ctx := context.Background()
	user, err := h.Services.Identity.CreateUser(ctx, "testuser", "Password123", false)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	return user
}

func TestCreateSession(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, refreshToken, err := svc.CreateSession(ctx, user, "test-user-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if sess == nil {
		t.Fatal("expected session to be non-nil")
	}

	if refreshToken == "" {
		t.Fatal("expected refresh token to be non-empty")
	}

	if sess.UUID() == uuid.Nil {
		t.Error("expected session UUID to be set")
	}

	if sess.UserID() != user.ID() {
		t.Errorf("expected UserID %d, got %d", user.ID(), sess.UserID())
	}

	if sess.UserAgent() != "test-user-agent" {
		t.Errorf("expected UserAgent %q, got %q", "test-user-agent", sess.UserAgent())
	}

	if !sess.Active() {
		t.Error("expected session to be active")
	}
}

func TestGetSessionByUUID(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	created, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	t.Run("existing session", func(t *testing.T) {
		sess, err := svc.GetSessionByUUID(ctx, created.UUID())
		if err != nil {
			t.Fatalf("GetSessionByUUID failed: %v", err)
		}

		if sess.UUID() != created.UUID() {
			t.Errorf("expected UUID %v, got %v", created.UUID(), sess.UUID())
		}
	})

	t.Run("non-existent session", func(t *testing.T) {
		_, err := svc.GetSessionByUUID(ctx, uuid.New())
		if err != session.ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGetSessionFromRefreshToken(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	created, refreshToken, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		sess, err := svc.GetSessionFromRefreshToken(ctx, refreshToken)
		if err != nil {
			t.Fatalf("GetSessionFromRefreshToken failed: %v", err)
		}

		if sess.UUID() != created.UUID() {
			t.Errorf("expected UUID %v, got %v", created.UUID(), sess.UUID())
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.GetSessionFromRefreshToken(ctx, "invalid-token")
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
	})

	t.Run("revoked session", func(t *testing.T) {
		if err := created.Revoke(ctx); err != nil {
			t.Fatalf("Revoke failed: %v", err)
		}

		if err := created.Reload(ctx); err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		_, err := svc.GetSessionFromRefreshToken(ctx, refreshToken)
		if err != session.ErrSessionEnded {
			t.Errorf("expected ErrSessionEnded, got %v", err)
		}
	})
}

func TestGetSessionDataFromAccessToken(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	accessToken, err := sess.NewAccessToken(ctx)
	if err != nil {
		t.Fatalf("NewAccessToken failed: %v", err)
	}

	t.Run("valid token", func(t *testing.T) {
		sessionData, err := svc.GetSessionDataFromAccessToken(accessToken)
		if err != nil {
			t.Fatalf("GetSessionDataFromAccessToken failed: %v", err)
		}

		if sessionData.SessionUUID != sess.UUID() {
			t.Errorf("expected session UUID %v, got %v", sess.UUID(), sessionData.SessionUUID)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := svc.GetSessionDataFromAccessToken("invalid-token")
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
	})
}

func TestGetSessionsForUser(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		_, _, err := svc.CreateSession(ctx, user, "test-agent")
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}
	}

	t.Run("get all sessions", func(t *testing.T) {
		sessions, total, err := svc.GetSessionsForUser(ctx, user, 10, 0)
		if err != nil {
			t.Fatalf("GetSessionsForUser failed: %v", err)
		}

		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}

		if len(sessions) != 5 {
			t.Errorf("expected 5 sessions, got %d", len(sessions))
		}
	})

	t.Run("pagination", func(t *testing.T) {
		sessions, total, err := svc.GetSessionsForUser(ctx, user, 2, 1)
		if err != nil {
			t.Fatalf("GetSessionsForUser failed: %v", err)
		}

		if total != 5 {
			t.Errorf("expected total 5, got %d", total)
		}

		if len(sessions) != 2 {
			t.Errorf("expected 2 sessions, got %d", len(sessions))
		}
	})
}

func TestSessionActive(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	t.Run("active session", func(t *testing.T) {
		sess, _, err := svc.CreateSession(ctx, user, "test-agent")
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}

		if !sess.Active() {
			t.Error("expected session to be active")
		}
	})

	t.Run("revoked session", func(t *testing.T) {
		sess, _, err := svc.CreateSession(ctx, user, "test-agent")
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}

		if err := sess.Revoke(ctx); err != nil {
			t.Fatalf("Revoke failed: %v", err)
		}

		if err := sess.Reload(ctx); err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		if sess.Active() {
			t.Error("expected revoked session to be inactive")
		}
	})

	t.Run("expired session", func(t *testing.T) {
		sess, _, err := svc.CreateSession(ctx, user, "test-agent")
		if err != nil {
			t.Fatalf("CreateSession failed: %v", err)
		}

		// Manually set created_at to be in the past beyond the TTL
		pastTime := time.Now().Add(-time.Duration(h.Config.Auth.RefreshTokenTTL+1) * 24 * time.Hour)
		h.DB.Model(&database.Session{}).Where("uuid = ?", sess.UUID()).Update("created_at", pastTime)

		if err := sess.Reload(ctx); err != nil {
			t.Fatalf("Reload failed: %v", err)
		}

		if sess.Active() {
			t.Error("expected expired session to be inactive")
		}
	})
}

func TestSessionRevoke(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if err := sess.Revoke(ctx); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	if err := sess.Reload(ctx); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if sess.Active() {
		t.Error("expected session to be inactive after revocation")
	}
}

func TestSessionExpiresAt(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	expectedExpiry := sess.CreatedAt().Add(time.Duration(h.Config.Auth.RefreshTokenTTL) * 24 * time.Hour)
	actualExpiry := sess.ExpiresAt()

	// Allow for small time differences
	diff := expectedExpiry.Sub(actualExpiry)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("expected expiry around %v, got %v", expectedExpiry, actualExpiry)
	}
}

func TestSessionUser(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	retrievedUser, err := sess.User(ctx)
	if err != nil {
		t.Fatalf("User failed: %v", err)
	}

	if retrievedUser.UUID() != user.UUID() {
		t.Errorf("expected user UUID %v, got %v", user.UUID(), retrievedUser.UUID())
	}
}

func TestSessionReload(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	originalUserAgent := sess.UserAgent()

	// Update user agent directly in database
	h.DB.Model(&database.Session{}).Where("uuid = ?", sess.UUID()).Update("user_agent", "updated-agent")

	// Before reload, should still have old value
	if sess.UserAgent() != originalUserAgent {
		t.Error("expected user agent to remain unchanged before reload")
	}

	// After reload, should have new value
	if err := sess.Reload(ctx); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if diff := cmp.Diff("updated-agent", sess.UserAgent()); diff != "" {
		t.Errorf("UserAgent mismatch (-want +got):\n%s", diff)
	}
}

func TestNewAccessToken(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	accessToken, err := sess.NewAccessToken(ctx)
	if err != nil {
		t.Fatalf("NewAccessToken failed: %v", err)
	}

	if accessToken == "" {
		t.Error("expected access token to be non-empty")
	}

	// Verify the token can be parsed back
	sessionData, err := svc.GetSessionDataFromAccessToken(accessToken)
	if err != nil {
		t.Fatalf("GetSessionDataFromAccessToken failed: %v", err)
	}

	if sessionData.SessionUUID != sess.UUID() {
		t.Errorf("expected session UUID %v, got %v", sess.UUID(), sessionData.SessionUUID)
	}
}

func TestSessionIPAddress(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// IPAddress starts empty since CreateSession doesn't set it
	if sess.IPAddress() != "" {
		t.Errorf("expected empty IP address, got %q", sess.IPAddress())
	}

	// Update IP address directly in database and reload
	h.DB.Model(&database.Session{}).Where("uuid = ?", sess.UUID()).Update("ip_address", "192.168.1.1")

	if err := sess.Reload(ctx); err != nil {
		t.Fatalf("Reload failed: %v", err)
	}

	if sess.IPAddress() != "192.168.1.1" {
		t.Errorf("expected IP address %q, got %q", "192.168.1.1", sess.IPAddress())
	}
}

func TestSessionCreatedAt(t *testing.T) {
	h := testkit.New(t)
	svc := h.Services.Session
	ctx := context.Background()
	user := testUser(t, h)

	beforeCreate := time.Now().Add(-time.Second)
	sess, _, err := svc.CreateSession(ctx, user, "test-agent")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	afterCreate := time.Now().Add(time.Second)

	createdAt := sess.CreatedAt()
	if createdAt.Before(beforeCreate) || createdAt.After(afterCreate) {
		t.Errorf("CreatedAt %v not in expected range [%v, %v]", createdAt, beforeCreate, afterCreate)
	}
}
