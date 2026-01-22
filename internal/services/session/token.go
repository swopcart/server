package session

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type AccessTokenClaims struct {
	SessionUUID uuid.UUID `json:"sessionId"`
	UserUUID    uuid.UUID `json:"userId"`
	jwt.RegisteredClaims
}

type RefreshTokenClaims struct {
	SessionUUID uuid.UUID `json:"sessionId"`
	jwt.RegisteredClaims
}

func (svc *SessionService) generateAccessToken(sessionUUID, userUUID uuid.UUID) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(svc.config.Auth.AccessTokenTTL) * time.Second)

	claims := AccessTokenClaims{
		SessionUUID: sessionUUID,
		UserUUID:    userUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    svc.config.Auth.JWTIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(svc.jwtKeys.private)
}

func (svc *SessionService) generateRefreshToken(sessionUUID uuid.UUID) (string, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(svc.config.Auth.RefreshTokenTTL) * 24 * time.Hour)

	claims := RefreshTokenClaims{
		SessionUUID: sessionUUID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        sessionUUID.String(),
			Issuer:    svc.config.Auth.JWTIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(svc.jwtKeys.private)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func (svc *SessionService) jwtKeyFunc(l *slog.Logger) func(*jwt.Token) (any, error) {
	return func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			l.Error("Unexpected signing method")
			return nil, ErrInvalidToken
		}
		return svc.jwtKeys.public, nil
	}
}
