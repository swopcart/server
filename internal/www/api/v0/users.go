package v0

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/services/identity"
)

func (h *APIHandlers) userChangePassword(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user UUID"})
		return
	}

	// Get current user from auth context
	currentUserUUID := uuid.MustParse(c.GetString("user_uuid"))
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, currentUserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current user"})
		return
	}

	// Parse request body
	var req struct {
		CurrentPassword *string `json:"currentPassword"`
		NewPassword     string  `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", currentUserUUID)

	// Authorization: non-admins can only change their own password
	isOwnPassword := currentUser.UUID() == targetUser.UUID()
	if !isOwnPassword && !currentUser.Admin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only change your own password"})
		return
	}

	// Password verification logic:
	// - Non-admins must always provide current password
	// - Admins can skip current password when changing others' passwords
	// - If currentPassword is provided (not nil), it MUST be checked
	mustCheckPassword := !currentUser.Admin() || req.CurrentPassword != nil

	if mustCheckPassword {
		if req.CurrentPassword == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is required"})
			return
		}

		if err := targetUser.CheckPassword(*req.CurrentPassword); err != nil {
			if errors.Is(err, identity.ErrIncorrectPassword) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
				return
			}
			l.Error("Failed to check password", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password"})
			return
		}
	}

	// Set new password
	if err := targetUser.SetPassword(ctx, req.NewPassword); err != nil {
		if errors.Is(err, identity.ErrPasswordTooShort) ||
			errors.Is(err, identity.ErrPasswordTooLong) ||
			errors.Is(err, identity.ErrPasswordNotComplex) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		l.Error("Failed to set password", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}

	l.Info("Password changed successfully")
	c.Status(http.StatusOK)
}

func (h *APIHandlers) userGetDetails(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user UUID"})
		return
	}

	// Get current user from auth context
	currentUserUUID := uuid.MustParse(c.GetString("user_uuid"))
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, currentUserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current user"})
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	// Authorization: non-admins can only view their own details
	isOwnDetails := currentUser.UUID() == targetUser.UUID()
	if !isOwnDetails && !currentUser.Admin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"uuid":        targetUser.UUID(),
		"username":    targetUser.Username(),
		"admin":       targetUser.Admin(),
		"totpEnabled": targetUser.HasTOTP(),
	})
}

func (h *APIHandlers) userGenerateTOTP(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user UUID"})
		return
	}

	// Get current user from auth context
	currentUserUUID := uuid.MustParse(c.GetString("user_uuid"))
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, currentUserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current user"})
		return
	}

	// Parse request body
	var req struct {
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", currentUserUUID)

	// Authorization: non-admins can only generate TOTP for themselves
	isOwnTOTP := currentUser.UUID() == targetUser.UUID()
	if !isOwnTOTP && !currentUser.Admin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only set up TOTP for yourself"})
		return
	}

	// Password verification: required for non-admins, or if provided by admin
	mustCheckPassword := !currentUser.Admin() || req.Password != nil

	if mustCheckPassword {
		if req.Password == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
			return
		}

		if err := targetUser.CheckPassword(*req.Password); err != nil {
			if errors.Is(err, identity.ErrIncorrectPassword) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Password is incorrect"})
				return
			}
			l.Error("Failed to check password", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password"})
			return
		}
	}

	// Generate TOTP secret
	pending, err := h.services.Identity.GenerateTOTP(targetUser.UUID(), targetUser.Username())
	if err != nil {
		l.Error("Failed to generate TOTP", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate TOTP"})
		return
	}

	l.Info("TOTP secret generated")
	c.JSON(http.StatusOK, gin.H{
		"secret": pending.Secret,
		"url":    pending.URL,
	})
}

func (h *APIHandlers) userEnableTOTP(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user UUID"})
		return
	}

	// Get current user from auth context
	currentUserUUID := uuid.MustParse(c.GetString("user_uuid"))
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, currentUserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current user"})
		return
	}

	// Parse request body
	var req struct {
		Secret string `json:"secret" binding:"required"`
		Token  string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", currentUserUUID)

	// Authorization: non-admins can only enable TOTP for themselves
	isOwnTOTP := currentUser.UUID() == targetUser.UUID()
	if !isOwnTOTP && !currentUser.Admin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only set up TOTP for yourself"})
		return
	}

	// Verify secret matches the pending secret
	pending := h.services.Identity.GetPendingTOTP(targetUser.UUID())
	if pending == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No pending TOTP secret found. Please generate a new secret first."})
		return
	}

	if pending.Secret != req.Secret {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Secret does not match pending secret"})
		return
	}

	// Enable TOTP (this also validates the token)
	if err := targetUser.EnableTOTP(ctx, req.Secret, req.Token); err != nil {
		if errors.Is(err, identity.ErrInvalidTOTP) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TOTP token"})
			return
		}
		l.Error("Failed to enable TOTP", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enable TOTP"})
		return
	}

	// Clear the pending secret
	h.services.Identity.ClearPendingTOTP(targetUser.UUID())

	l.Info("TOTP enabled successfully")
	c.Status(http.StatusOK)
}

func (h *APIHandlers) userDisableTOTP(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user UUID"})
		return
	}

	// Get current user from auth context
	currentUserUUID := uuid.MustParse(c.GetString("user_uuid"))
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, currentUserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get current user"})
		return
	}

	// Parse request body
	var req struct {
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user"})
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", currentUserUUID)

	// Authorization: non-admins can only disable TOTP for themselves
	isOwnTOTP := currentUser.UUID() == targetUser.UUID()
	if !isOwnTOTP && !currentUser.Admin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only disable TOTP for yourself"})
		return
	}

	// Password verification: required for non-admins, or if provided by admin
	mustCheckPassword := !currentUser.Admin() || req.Password != nil

	if mustCheckPassword {
		if req.Password == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password is required"})
			return
		}

		if err := targetUser.CheckPassword(*req.Password); err != nil {
			if errors.Is(err, identity.ErrIncorrectPassword) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Password is incorrect"})
				return
			}
			l.Error("Failed to check password", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify password"})
			return
		}
	}

	// Disable TOTP
	if err := targetUser.DisableTOTP(ctx); err != nil {
		if errors.Is(err, identity.ErrTOTPNotEnabled) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP is not enabled"})
			return
		}
		l.Error("Failed to disable TOTP", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable TOTP"})
		return
	}

	l.Info("TOTP disabled successfully")
	c.Status(http.StatusOK)
}
