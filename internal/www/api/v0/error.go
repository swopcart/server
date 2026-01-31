package v0

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/services/session"
)

// Error codes - Authentication
const (
	ErrCodeInvalidCredentials = "InvalidCredentials"
	ErrCodeRequiresTOTP       = "RequiresTOTP"
	ErrCodeInvalidTOTP        = "InvalidTOTP"
	ErrCodeInvalidToken       = "InvalidToken"
	ErrCodeMissingAuth        = "MissingAuth"
)

// Error codes - Authorization
const (
	ErrCodeForbidden       = "Forbidden"
	ErrCodeOnlyAdmins      = "OnlyAdmins"
	ErrCodeOnlyOwnResource = "OnlyOwnResource"
)

// Error codes - Validation
const (
	ErrCodeInvalidRequest = "InvalidRequest"
	ErrCodeInvalidUUID    = "InvalidUUID"
	ErrCodeRequiredField  = "RequiredField"
	ErrCodeTooShort       = "TooShort"
	ErrCodeTooLong        = "TooLong"
	ErrCodeInvalidChars   = "InvalidChars"
	ErrCodeInvalidFormat  = "InvalidFormat"
)

// Error codes - User management
const (
	ErrCodeUserNotFound      = "UserNotFound"
	ErrCodeUserAlreadyExists = "UserAlreadyExists"
)

// Error codes - Password management
const (
	ErrCodePasswordIncorrect  = "PasswordIncorrect"
	ErrCodePasswordRequired   = "PasswordRequired"
	ErrCodePasswordNotComplex = "PasswordNotComplex"
)

// Error codes - TOTP management
const (
	ErrCodeTOTPNotEnabled     = "TOTPNotEnabled"
	ErrCodeTOTPNotPending     = "TOTPNotPending"
	ErrCodeTOTPSecretMismatch = "TOTPSecretMismatch"
)

// Error codes - Session management
const (
	ErrCodeSessionNotFound = "SessionNotFound"
	ErrCodeSessionEnded    = "SessionEnded"
)

// respondError sends a single error response
func respondError(c *gin.Context, status int, code, message string) {
	c.JSON(status, ErrorResponse{
		Errors: []Error{{
			Error:   code,
			Message: message,
		}},
	})
}

// respondFieldError sends an error for a specific field
func respondFieldError(c *gin.Context, status int, code, message, key string) {
	c.JSON(status, ErrorResponse{
		Errors: []Error{{
			Error:   code,
			Message: message,
			Key:     key,
		}},
	})
}

type errorMapping struct {
	code    string
	message string
	key     string
}

var identityErrorMap = map[error]errorMapping{
	identity.ErrNotFound:                {ErrCodeUserNotFound, "User not found", ""},
	identity.ErrIncorrectPassword:       {ErrCodePasswordIncorrect, "Password is incorrect", ".password"},
	identity.ErrUserAlreadyExists:       {ErrCodeUserAlreadyExists, "User already exists", ".username"},
	identity.ErrUsernameInvalidChars:    {ErrCodeInvalidChars, "Username contains invalid characters", ".username"},
	identity.ErrUsernameStartsWithDigit: {ErrCodeInvalidFormat, "Username cannot start with a digit", ".username"},
	identity.ErrPasswordNotComplex:      {ErrCodePasswordNotComplex, "Password must contain a lowercase letter, uppercase letter, and digit", ".password"},
	identity.ErrInvalidTOTP:             {ErrCodeInvalidTOTP, "TOTP token is invalid", ".totp"},
	identity.ErrTOTPNotEnabled:          {ErrCodeTOTPNotEnabled, "TOTP is not enabled", ""},
	identity.ErrTOTPNotPending:          {ErrCodeTOTPNotPending, "No pending TOTP secret found. Please generate a new secret first.", ""},
	identity.ErrTOTPSecretMismatch:      {ErrCodeTOTPSecretMismatch, "Secret does not match pending secret", ""},
	identity.ErrUsernameTooShort:        {ErrCodeTooShort, fmt.Sprintf("Username must be at least %d characters", identity.MinUsernameLength), ".username"},
	identity.ErrUsernameTooLong:         {ErrCodeTooLong, fmt.Sprintf("Username must be at most %d characters", identity.MaxUsernameLength), ".username"},
	identity.ErrPasswordTooShort:        {ErrCodeTooShort, fmt.Sprintf("Password must be at least %d characters", identity.MinPasswordLength), ".password"},
	identity.ErrPasswordTooLong:         {ErrCodeTooLong, fmt.Sprintf("Password must be at most %d characters", identity.MaxPasswordLength), ".password"},
}

var sessionErrorMap = map[error]errorMapping{
	session.ErrNotFound:     {ErrCodeSessionNotFound, "Session not found", ""},
	session.ErrSessionEnded: {ErrCodeSessionEnded, "Session has ended", ""},
	session.ErrInvalidToken: {ErrCodeInvalidToken, "Invalid token", ""},
}

// mapIdentityError maps service layer errors to API error codes.
// Returns (code, message, key) where key is the field path (e.g., ".username", ".password")
func mapIdentityError(err error) (code, message, key string) {
	for target, mapping := range identityErrorMap {
		if errors.Is(err, target) {
			return mapping.code, mapping.message, mapping.key
		}
	}
	return "", "", ""
}

// mapSessionError maps session service errors to API error codes
func mapSessionError(err error) (code, message string) {
	for target, mapping := range sessionErrorMap {
		if errors.Is(err, target) {
			return mapping.code, mapping.message
		}
	}
	return "", ""
}

// respondWithIdentityError handles service layer identity errors
func respondWithIdentityError(c *gin.Context, err error, defaultStatus int) {
	code, message, key := mapIdentityError(err)
	if code == "" {
		// Unknown error - log and return 500 status only
		c.Status(http.StatusInternalServerError)
		return
	}

	if key != "" {
		respondFieldError(c, defaultStatus, code, message, key)
	} else {
		respondError(c, defaultStatus, code, message)
	}
}

// respondWithSessionError handles service layer session errors.
// For security reasons, all session errors are reported to the client as
// "Invalid token" to avoid leaking information about session state.
// The actual error details are logged server-side for debugging.
func respondWithSessionError(c *gin.Context, l *slog.Logger, err error, defaultStatus int) {
	// Check if this is a known session error and log the actual details
	code, message := mapSessionError(err)
	if code == "" {
		// Unknown error - log and return 500 status only
		l.Error("Unknown session error", "err", err)
		c.Status(http.StatusInternalServerError)
		return
	}

	// Log the actual error details server-side
	l.Debug("Session error", "code", code, "message", message, "err", err)

	// Always return generic "Invalid token" message to client for security
	respondError(c, defaultStatus, ErrCodeInvalidToken, "Invalid token")
}

// respondBindingError handles Gin validation errors
func respondBindingError(c *gin.Context, err error) {
	respondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
}
