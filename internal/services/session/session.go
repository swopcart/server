package session

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/database"
	"github.com/swopcart/server/internal/services/identity"
	"gorm.io/gorm"
)

var (
	ErrInternal = errors.New("internal error")

	ErrNotFound     = errors.New("session not found")
	ErrSessionEnded = errors.New("session has ended")

	ErrInvalidToken = errors.New("invalid token")
)

// Session represents an authenticated user session with associated metadata.
// It wraps the database session record and provides methods for token
// management.
type Session struct {
	service *SessionService
	uuid    uuid.UUID
	data    database.Session
}

func (svc *SessionService) newSession(session database.Session) *Session {
	return &Session{
		service: svc,
		uuid:    session.UUID,
		data:    session,
	}
}

// CreateSession creates a new session for the given user and returns the
// session along with a refresh token. The refresh token should be stored
// securely by the client and used to obtain new access tokens.
func (svc *SessionService) CreateSession(
	ctx context.Context,
	user *identity.User,
	userAgent string,
) (s *Session, refreshToken string, err error) {
	l := svc.logger.
		WithGroup("CreateSession").
		With("user.uuid", user.UUID())

	sessionUUID := uuid.New()

	refreshToken, err = svc.generateRefreshToken(sessionUUID)
	if err != nil {
		l.Error("Failed to generate refresh token", "err", err)
		err = errors.Join(ErrInternal, err)
		return
	}

	refreshTokenHash := hashToken(refreshToken)

	session := database.Session{
		UUID:             sessionUUID,
		RefreshTokenHash: refreshTokenHash,
		UserID:           user.ID(),
		UserAgent:        userAgent,
	}

	err = gorm.G[database.Session](svc.db).Create(ctx, &session)
	if err != nil {
		l.Error("Failed to store session", "err", err)
		err = errors.Join(ErrInternal, err)
		return
	}

	s = svc.newSession(session)
	return
}

func (svc *SessionService) GetSessionByUUID(ctx context.Context, sessionUUID uuid.UUID) (*Session, error) {
	l := svc.logger.WithGroup("GetSessionByUUID").
		With("session.uuid", sessionUUID)

	session, err := gorm.G[database.Session](svc.db).
		Where("uuid = ?", sessionUUID).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		l.Error("Failed to get session")
		return nil, err
	}

	return svc.newSession(session), nil
}

// GetSessionDataFromAccessToken parses and validates an access token, returning
// the session UUID and user UUID embedded in the token claims. Returns
// ErrInvalidToken if the token is malformed, expired, or has an invalid
// signature.
func (svc *SessionService) GetSessionDataFromAccessToken(
	accessToken string,
) (sessionUUID, userUUID uuid.UUID, err error) {
	l := svc.logger.WithGroup("GetSessionDataFromAccessToken")

	token, err := jwt.ParseWithClaims(accessToken, &AccessTokenClaims{}, svc.jwtKeyFunc(l))
	if err != nil {
		l.Error("Can't parse JWT token", "err", err)
		err = errors.Join(ErrInvalidToken, err)
		return
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		err = ErrInvalidToken
		return
	}

	sessionUUID = claims.SessionUUID
	userUUID = claims.UserUUID
	return
}

// GetSessionFromRefreshToken validates a refresh token and returns the
// associated session. Returns ErrInvalidToken if the token is malformed or
// doesn't match the stored hash, ErrNotFound if the session doesn't exist,
// or ErrSessionEnded if the session has been terminated.
func (svc *SessionService) GetSessionFromRefreshToken(
	ctx context.Context,
	refreshToken string,
) (*Session, error) {
	l := svc.logger.WithGroup("GetSessionFromRefreshToken")

	token, err := jwt.ParseWithClaims(refreshToken, &RefreshTokenClaims{}, svc.jwtKeyFunc(l))
	if err != nil {
		l.Error("Can't parse JWT token", "err", err)
		return nil, errors.Join(ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*RefreshTokenClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	l = l.With("session.uuid", claims.SessionUUID)

	session, err := gorm.G[database.Session](svc.db).
		Where("uuid = ?", claims.SessionUUID).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		l.Error("Can't get the session database entry")
		return nil, errors.Join(ErrInternal, err)
	}

	if session.RefreshTokenHash != hashToken(refreshToken) {
		l.Error("Refresh token doesn't match the one we have stored")
		return nil, ErrInvalidToken
	}

	s := svc.newSession(session)

	if !s.Active() {
		return nil, ErrSessionEnded
	}

	return s, nil
}

func (svc *SessionService) GetSessionsForUser(
	ctx context.Context,
	u *identity.User,
	limit, offset int,
) (s []*Session, total uint, err error) {
	userSessions := gorm.G[database.Session](svc.db).Where("user_id = ?", u.ID())

	sessionCount, err := userSessions.Count(ctx, "id")
	if err != nil {
		return
	}

	sessions, err := userSessions.Offset(offset).Limit(limit).Find(ctx)
	if err != nil {
		return
	}

	s = make([]*Session, 0, limit)
	total = uint(sessionCount)

	for _, session := range sessions {
		s = append(s, svc.newSession(session))
	}

	return
}

func (s *Session) UUID() uuid.UUID {
	return s.data.UUID
}

// NewAccessToken generates a new short-lived access token for this session.
// Returns the token string and its time-to-live in seconds.
func (s *Session) NewAccessToken(ctx context.Context) (accessToken string, err error) {
	svc := s.service
	l := svc.logger.
		WithGroup("NewAccessToken").
		With("session.uuid", s.uuid)

	user, err := s.User(ctx)
	if err != nil {
		l.Error("Failed to retrieve user", "err", err)
		return
	}

	accessToken, err = svc.generateAccessToken(s.uuid, user.UUID())
	if err != nil {
		l.Error("Failed to generate access token", "err", err)
		return
	}

	return
}

// Active returns true if the session is still valid and has not been terminated.
func (s *Session) Active() bool {
	return s.data.RevokedAt == nil &&
		time.Now().Before(s.ExpiresAt())
}

func (s *Session) ExpiresAt() time.Time {
	cfg := s.service.config
	return s.data.CreatedAt.Add(time.Duration(cfg.Auth.RefreshTokenTTL) * 24 * time.Hour)
}

func (s *Session) Revoke(ctx context.Context) error {
	_, err := gorm.G[database.Session](s.service.db).
		Where("uuid = ?", s.uuid).
		Update(ctx, "revoked_at", time.Now())
	return err
}

func (s *Session) CreatedAt() time.Time {
	return s.data.CreatedAt
}

func (s *Session) UserAgent() string {
	return s.data.UserAgent
}

func (s *Session) IPAddress() string {
	return s.data.IPAddress
}

func (s *Session) UserID() uint {
	return s.data.UserID
}

func (s *Session) User(ctx context.Context) (*identity.User, error) {
	return s.service.identity.GetUserByID(ctx, s.data.UserID)
}

// Reload refreshes the session data from the database, updating the local
// copy with any changes that may have occurred.
func (s *Session) Reload(ctx context.Context) error {
	session, err := gorm.G[database.Session](s.service.db).
		Where("uuid = ?", s.uuid).
		First(ctx)
	if err != nil {
		return err
	}

	s.data = session
	return nil
}
