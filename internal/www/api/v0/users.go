package v0

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/swopcart/server/internal/services/identity"
)

func (h *APIHandlers) userList(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	var req struct {
		Offset int `form:"offset"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		l.Debug("Invalid request", "err", err)
		req.Offset = 0
	}

	if req.Offset < 0 {
		req.Offset = 0
	}

	users, total, err := h.services.Identity.GetAllUsers(ctx, PageSize, req.Offset)
	if err != nil {
		l.Error("Failed to get users", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	response := PaginatedResponse[gin.H]{
		Offset: uint(req.Offset),
		Total:  total,
		Items:  make([]gin.H, 0, len(users)),
	}

	for _, user := range users {
		response.Items = append(response.Items, gin.H{
			"uuid":        user.UUID(),
			"username":    user.Username(),
			"admin":       user.Admin(),
			"totpEnabled": user.HasTOTP(),
			"createdAt":   user.CreatedAt(),
		})
	}

	c.JSON(http.StatusOK, response)
}

func (h *APIHandlers) userChangePassword(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid user UUID")
		return
	}

	// Get current user from auth context
	sessionData := getSessionData(c)
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, sessionData.UserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Parse request body
	var req struct {
		CurrentPassword *string `json:"currentPassword"`
		NewPassword     string  `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			respondWithIdentityError(c, err, http.StatusNotFound)
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", sessionData.UserUUID)

	// Authorization: non-admins can only change their own password
	isOwnPassword := currentUser.UUID() == targetUser.UUID()
	if !isOwnPassword && !currentUser.Admin() {
		respondError(c, http.StatusForbidden, ErrCodeOnlyOwnResource, "You can only change your own password")
		return
	}

	// Password verification logic:
	// - Non-admins must always provide current password
	// - Admins can skip current password when changing others' passwords
	// - If currentPassword is provided (not nil), it MUST be checked
	mustCheckPassword := !currentUser.Admin() || req.CurrentPassword != nil

	if mustCheckPassword {
		if req.CurrentPassword == nil {
			respondFieldError(c, http.StatusBadRequest, ErrCodePasswordRequired, "Current password is required", ".currentPassword")
			return
		}

		if err := targetUser.CheckPassword(*req.CurrentPassword); err != nil {
			if errors.Is(err, identity.ErrIncorrectPassword) {
				respondFieldError(c, http.StatusUnauthorized, ErrCodePasswordIncorrect, "Current password is incorrect", ".currentPassword")
				return
			}
			l.Error("Failed to check password", "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	// Set new password
	if err := targetUser.SetPassword(ctx, req.NewPassword); err != nil {
		if errors.Is(err, identity.ErrPasswordTooShort) ||
			errors.Is(err, identity.ErrPasswordTooLong) ||
			errors.Is(err, identity.ErrPasswordNotComplex) {
			code, message, _ := mapIdentityError(err)
			respondFieldError(c, http.StatusBadRequest, code, message, ".newPassword")
			return
		}
		l.Error("Failed to set password", "err", err)
		c.Status(http.StatusInternalServerError)
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
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid user UUID")
		return
	}

	// Get current user from auth context
	sessionData := getSessionData(c)
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, sessionData.UserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			respondWithIdentityError(c, err, http.StatusNotFound)
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Authorization: non-admins can only view their own details
	isOwnDetails := currentUser.UUID() == targetUser.UUID()
	if !isOwnDetails && !currentUser.Admin() {
		respondError(c, http.StatusForbidden, ErrCodeOnlyOwnResource, "You can only view your own details")
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
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid user UUID")
		return
	}

	// Get current user from auth context
	sessionData := getSessionData(c)
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, sessionData.UserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Parse request body
	var req struct {
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			respondWithIdentityError(c, err, http.StatusNotFound)
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", sessionData.UserUUID)

	// Authorization: non-admins can only generate TOTP for themselves
	isOwnTOTP := currentUser.UUID() == targetUser.UUID()
	if !isOwnTOTP && !currentUser.Admin() {
		respondError(c, http.StatusForbidden, ErrCodeOnlyOwnResource, "You can only set up TOTP for yourself")
		return
	}

	// Password verification: required for non-admins, or if provided by admin
	mustCheckPassword := !currentUser.Admin() || req.Password != nil

	if mustCheckPassword {
		if req.Password == nil {
			respondFieldError(c, http.StatusBadRequest, ErrCodePasswordRequired, "Password is required", ".password")
			return
		}

		if err := targetUser.CheckPassword(*req.Password); err != nil {
			if errors.Is(err, identity.ErrIncorrectPassword) {
				respondFieldError(c, http.StatusUnauthorized, ErrCodePasswordIncorrect, "Password is incorrect", ".password")
				return
			}
			l.Error("Failed to check password", "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	// Generate TOTP secret
	pending, err := h.services.Identity.GenerateTOTP(targetUser.UUID(), targetUser.Username())
	if err != nil {
		l.Error("Failed to generate TOTP", "err", err)
		c.Status(http.StatusInternalServerError)
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
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid user UUID")
		return
	}

	// Get current user from auth context
	sessionData := getSessionData(c)
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, sessionData.UserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Parse request body
	var req struct {
		Secret string `json:"secret" binding:"required"`
		Token  string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			respondWithIdentityError(c, err, http.StatusNotFound)
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", sessionData.UserUUID)

	// Authorization: non-admins can only enable TOTP for themselves
	isOwnTOTP := currentUser.UUID() == targetUser.UUID()
	if !isOwnTOTP && !currentUser.Admin() {
		respondError(c, http.StatusForbidden, ErrCodeOnlyOwnResource, "You can only set up TOTP for yourself")
		return
	}

	// Verify secret matches the pending secret
	pending := h.services.Identity.GetPendingTOTP(targetUser.UUID())
	if pending == nil {
		respondWithIdentityError(c, identity.ErrTOTPNotPending, http.StatusBadRequest)
		return
	}

	if pending.Secret != req.Secret {
		respondWithIdentityError(c, identity.ErrTOTPSecretMismatch, http.StatusBadRequest)
		return
	}

	// Enable TOTP (this also validates the token)
	if err := targetUser.EnableTOTP(ctx, req.Secret, req.Token); err != nil {
		if errors.Is(err, identity.ErrInvalidTOTP) {
			respondFieldError(c, http.StatusBadRequest, ErrCodeInvalidTOTP, "Invalid TOTP token", ".token")
			return
		}
		l.Error("Failed to enable TOTP", "err", err)
		c.Status(http.StatusInternalServerError)
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
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid user UUID")
		return
	}

	// Get current user from auth context
	sessionData := getSessionData(c)
	currentUser, err := h.services.Identity.GetUserByUUID(ctx, sessionData.UserUUID)
	if err != nil {
		l.Error("Failed to get current user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Parse request body
	var req struct {
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	// Get target user
	targetUser, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			respondWithIdentityError(c, err, http.StatusNotFound)
			return
		}
		l.Error("Failed to get target user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l = l.With("target_user_uuid", targetUUID, "current_user_uuid", sessionData.UserUUID)

	// Authorization: non-admins can only disable TOTP for themselves
	isOwnTOTP := currentUser.UUID() == targetUser.UUID()
	if !isOwnTOTP && !currentUser.Admin() {
		respondError(c, http.StatusForbidden, ErrCodeOnlyOwnResource, "You can only disable TOTP for yourself")
		return
	}

	// Password verification: required for non-admins, or if provided by admin
	mustCheckPassword := !currentUser.Admin() || req.Password != nil

	if mustCheckPassword {
		if req.Password == nil {
			respondFieldError(c, http.StatusBadRequest, ErrCodePasswordRequired, "Password is required", ".password")
			return
		}

		if err := targetUser.CheckPassword(*req.Password); err != nil {
			if errors.Is(err, identity.ErrIncorrectPassword) {
				respondFieldError(c, http.StatusUnauthorized, ErrCodePasswordIncorrect, "Password is incorrect", ".password")
				return
			}
			l.Error("Failed to check password", "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}
	}

	// Disable TOTP
	if err := targetUser.DisableTOTP(ctx); err != nil {
		if errors.Is(err, identity.ErrTOTPNotEnabled) {
			respondWithIdentityError(c, err, http.StatusBadRequest)
			return
		}
		l.Error("Failed to disable TOTP", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l.Info("TOTP disabled successfully")
	c.Status(http.StatusOK)
}

func (h *APIHandlers) userCreate(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Check admin from JWT token claims
	if !getSessionData(c).Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can create users")
		return
	}

	// Parse request body
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Admin    bool   `json:"admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondBindingError(c, err)
		return
	}

	l = l.With("new_username", req.Username, "admin", req.Admin)

	// Create the user
	newUser, err := h.services.Identity.CreateUser(ctx, req.Username, req.Password, req.Admin)
	if err != nil {
		if errors.Is(err, identity.ErrUserAlreadyExists) {
			code, message, _ := mapIdentityError(err)
			respondFieldError(c, http.StatusConflict, code, message, ".username")
			return
		}
		if errors.Is(err, identity.ErrUsernameTooShort) ||
			errors.Is(err, identity.ErrUsernameTooLong) ||
			errors.Is(err, identity.ErrUsernameInvalidChars) ||
			errors.Is(err, identity.ErrUsernameStartsWithDigit) {
			code, message, _ := mapIdentityError(err)
			respondFieldError(c, http.StatusBadRequest, code, message, ".username")
			return
		}
		if errors.Is(err, identity.ErrPasswordTooShort) ||
			errors.Is(err, identity.ErrPasswordTooLong) ||
			errors.Is(err, identity.ErrPasswordNotComplex) {
			code, message, _ := mapIdentityError(err)
			respondFieldError(c, http.StatusBadRequest, code, message, ".password")
			return
		}
		l.Error("Failed to create user", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	l.Info("User created successfully", "new_user_uuid", newUser.UUID())
	c.JSON(http.StatusCreated, gin.H{
		"uuid":        newUser.UUID(),
		"username":    newUser.Username(),
		"admin":       newUser.Admin(),
		"totpEnabled": newUser.HasTOTP(),
		"createdAt":   newUser.CreatedAt(),
	})
}

func (h *APIHandlers) userListSessions(c *gin.Context) {
	ctx := c.Request.Context()
	l := h.getLogger(c)

	// Check admin from JWT token claims
	if !getSessionData(c).Admin {
		respondError(c, http.StatusForbidden, ErrCodeOnlyAdmins, "Only admins can view other users' sessions")
		return
	}

	// Parse target user UUID from URL
	targetUUID, err := uuid.Parse(c.Param("user_uuid"))
	if err != nil {
		respondError(c, http.StatusBadRequest, ErrCodeInvalidUUID, "Invalid user UUID")
		return
	}

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

	// Get the target user
	user, err := h.services.Identity.GetUserByUUID(ctx, targetUUID)
	if err != nil {
		if errors.Is(err, identity.ErrNotFound) {
			respondWithIdentityError(c, err, http.StatusNotFound)
			return
		}
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

	response := PaginatedResponse[gin.H]{
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
