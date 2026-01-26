package identity

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	ErrTOTPNotPending     = errors.New("no pending TOTP secret found")
	ErrTOTPNotEnabled     = errors.New("TOTP is not enabled")
	ErrTOTPSecretMismatch = errors.New("TOTP secret does not match pending secret")
)

// PendingTOTP stores a generated TOTP secret waiting for confirmation
type PendingTOTP struct {
	Secret    string
	URL       string
	ExpiresAt time.Time
}

// pendingTOTPStore manages pending TOTP secrets with automatic expiration
type pendingTOTPStore struct {
	mu      sync.RWMutex
	secrets map[uuid.UUID]*PendingTOTP
	ttl     time.Duration
}

func newPendingTOTPStore(ttl time.Duration) *pendingTOTPStore {
	return &pendingTOTPStore{
		secrets: make(map[uuid.UUID]*PendingTOTP),
		ttl:     ttl,
	}
}

// Set stores a pending TOTP secret for a user
func (s *pendingTOTPStore) Set(userUUID uuid.UUID, secret, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.secrets[userUUID] = &PendingTOTP{
		Secret:    secret,
		URL:       url,
		ExpiresAt: time.Now().Add(s.ttl),
	}
}

// Get retrieves a pending TOTP secret for a user, or nil if not found/expired
func (s *pendingTOTPStore) Get(userUUID uuid.UUID) *PendingTOTP {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pending, ok := s.secrets[userUUID]
	if !ok {
		return nil
	}
	if time.Now().After(pending.ExpiresAt) {
		return nil
	}
	return pending
}

// Delete removes a pending TOTP secret for a user
func (s *pendingTOTPStore) Delete(userUUID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.secrets, userUUID)
}

// Cleanup removes expired secrets (call periodically)
func (s *pendingTOTPStore) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for userUUID, pending := range s.secrets {
		if now.After(pending.ExpiresAt) {
			delete(s.secrets, userUUID)
		}
	}
}

const (
	TOTPIssuer     = "Swopcart"
	PendingTOTPTTL = 10 * time.Minute
)

// GenerateTOTP creates a new TOTP secret for a user and stores it as pending
func (svc *IdentityService) GenerateTOTP(userUUID uuid.UUID, username string) (*PendingTOTP, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      TOTPIssuer,
		AccountName: username,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return nil, errors.Join(ErrInternal, err)
	}

	pending := &PendingTOTP{
		Secret:    key.Secret(),
		URL:       key.URL(),
		ExpiresAt: time.Now().Add(PendingTOTPTTL),
	}

	svc.pendingTOTP.Set(userUUID, pending.Secret, pending.URL)

	return pending, nil
}

// GetPendingTOTP retrieves the pending TOTP secret for a user
func (svc *IdentityService) GetPendingTOTP(userUUID uuid.UUID) *PendingTOTP {
	return svc.pendingTOTP.Get(userUUID)
}

// ClearPendingTOTP removes the pending TOTP secret for a user
func (svc *IdentityService) ClearPendingTOTP(userUUID uuid.UUID) {
	svc.pendingTOTP.Delete(userUUID)
}

// User TOTP methods

// HasTOTP returns true if the user has TOTP enabled
func (u *User) HasTOTP() bool {
	return u.data.TOTP != nil
}

// DisableTOTP removes the TOTP secret from the user
func (u *User) DisableTOTP(ctx context.Context) error {
	if u.data.TOTP == nil {
		return ErrTOTPNotEnabled
	}

	_, err := u.gormChain().Update(ctx, "totp", nil)
	if err != nil {
		return err
	}

	return u.Reload(ctx)
}
