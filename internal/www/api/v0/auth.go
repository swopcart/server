package v0

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/services/session"
)

func (h *APIHandlers) authLogin(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		TOTP     string `json:"totp"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	l = l.With("username", req.Username)

	// TODO: add account lockout after a number of failed password checks

	user, err := h.services.Identity.GetUserByUsername(ctx, req.Username)
	if err != nil {
		l.Error("Failed to find user with username", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username and password are incorrect"})
		return
	}

	l = l.With("user.uuid", user.UUID())

	err = user.CheckPassword(req.Password)
	if err != nil {
		l.Error("Failed to check user password", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Username and password are incorrect"})
		return
	}

	err = user.CheckTOTP(req.TOTP)
	if err != nil {
		l.Error("Failed to check user's TOTP token", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":        "TOTP token is not valid",
			"requiresTotp": true,
		})
		return
	}

	session, refreshToken, err := h.services.Session.CreateSession(ctx, user, c.Request.UserAgent())
	if err != nil {
		l.Error("Failed to create session", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l = l.With("session.uuid", session.UUID())

	accessToken, err := session.NewAccessToken(ctx)
	if err != nil {
		l.Error("Failed to create access token", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"refreshToken": refreshToken,
		"accessToken":  accessToken,
		"user": gin.H{
			"id":       user.ID(),
			"uuid":     user.UUID(),
			"username": user.Username(),
			"admin":    user.Admin(),
		},
	})
}

func (h *APIHandlers) authConnect(c *gin.Context) { panic("TODO") }

func (h *APIHandlers) authRevokeSession(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c).With("session.uuid", c.Param("session_uuid"))

	currentUser, err := h.services.Identity.GetUserByUUID(ctx, uuid.MustParse(c.GetString("user_uuid")))
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l = l.With("current_user.uuid", currentUser.UUID())

	sessionUUID, err := uuid.Parse(c.Param("session_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	s, err := h.services.Session.GetSessionByUUID(ctx, sessionUUID)
	if errors.Is(err, session.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Session not found",
		})
		return
	} else if err != nil {
		l.Error("Failed to get session", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	if s.UserID() != currentUser.ID() && !currentUser.Admin() {
		l.Error("Refusing to delete another user's session", "session.user.id", s.UserID(), "current_user.id", currentUser.ID())
		c.JSON(http.StatusForbidden, gin.H{
			"error": "You can't revoke another user's session",
		})
		return
	}

	err = s.Revoke(ctx)
	if err != nil {
		l.Error("Failed to revoke session", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

func (h *APIHandlers) authRevokeSelf(c *gin.Context) {
	l := h.getLogger(c)
	ctx := c.Request.Context()

	s, err := h.services.Session.GetSessionByUUID(ctx, uuid.MustParse(c.GetString("session_uuid")))
	if err != nil {
		l.Error("Failed to get session", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	err = s.Revoke(ctx)
	if err != nil {
		l.Error("Failed to revoke session", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

func (h *APIHandlers) authRefresh(c *gin.Context) {
	l := h.getLogger(c)

	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(400)
		return
	}

	s, err := h.services.Session.GetSessionFromRefreshToken(
		c.Request.Context(),
		req.RefreshToken)
	if err != nil {
		l.Error("Failed to get session from refresh token", "err", err)
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid token",
		})

		return
	}

	accessToken, err := s.NewAccessToken(c.Request.Context())
	if err != nil {
		l.Error("Failed to generate new access token", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken": accessToken,
	})
}

func (h *APIHandlers) authListSessions(c *gin.Context) {
	l := h.getLogger(c)
	ctx := c.Request.Context()

	var req struct {
		Offset int `json:"offset"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		l.Debug("Invalid request", "err", err)
		req.Offset = 0
	}

	if req.Offset < 0 {
		req.Offset = 0
	}

	user, err := h.services.Identity.GetUserByUUID(
		ctx,
		uuid.MustParse(c.GetString("user_uuid")))
	if err != nil {
		l.Error("Failed to get user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l.Debug("Get sessions for user", "user.uuid", user.UUID(), "offset", req.Offset)
	sessions, total, err := h.services.Session.GetSessionsForUser(ctx, user, PageSize, req.Offset)
	if err != nil {
		l.Error("Failed to get user's sessions", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	response := Paginated[gin.H]{
		Offset: uint(req.Offset),
		Total:  total,
		Items:  make([]gin.H, 0, len(sessions)),
	}

	for _, session := range sessions {
		response.Items = append(response.Items, gin.H{
			"uuid":      session.UUID(),
			"createdAt": session.CreatedAt(),
			"userAgent": session.UserAgent(),
			"ipAddress": session.IPAddress(),
			"active":    session.Active(),
			"revokedAt": session.RevokedAt(),
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *APIHandlers) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authentication header"})
			c.Abort()
			return
		}

		accessToken := strings.TrimPrefix(authHeader, "Bearer ")

		sid, uid, err := h.services.Session.GetSessionDataFromAccessToken(accessToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("session_uuid", sid.String())
		c.Set("user_uuid", uid.String())
		c.Next()
	}
}
