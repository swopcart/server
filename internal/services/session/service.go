package session

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"log/slog"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/swopcart/server/internal/config"
	"github.com/swopcart/server/internal/services/identity"
	"gorm.io/gorm"
)

type SessionService struct {
	config   *config.Config
	logger   *slog.Logger
	db       *gorm.DB
	identity *identity.IdentityService

	jwtKeys keyPair
}

type keyPair struct {
	private *rsa.PrivateKey
	public  *rsa.PublicKey
}

func NewSessionService(
	config *config.Config,
	logger *slog.Logger,
	db *gorm.DB,
	identity *identity.IdentityService,
) (*SessionService, error) {
	keyPair, err := loadOrCreateKeyPair(config)
	if err != nil {
		return nil, err
	}

	return &SessionService{
		config:   config,
		logger:   logger,
		db:       db,
		identity: identity,
		jwtKeys:  keyPair,
	}, nil
}

func loadOrCreateKeyPair(cfg *config.Config) (kp keyPair, err error) {
	keyPath := cfg.Auth.AbsoluteKeyPath()

	var privateKey *rsa.PrivateKey
	privateKeyData, err := os.ReadFile(keyPath)
	if errors.Is(err, os.ErrNotExist) {
		privateKey, err = generateKeyPair(keyPath)
		if err != nil {
			return
		}
	} else if err != nil {
		return
	}

	if privateKey == nil {
		privateKey, err = jwt.ParseRSAPrivateKeyFromPEM(privateKeyData)
		if err != nil {
			return
		}
	}

	kp = keyPair{
		private: privateKey,
		public:  &privateKey.PublicKey,
	}
	return
}

func generateKeyPair(keyPath string) (key *rsa.PrivateKey, err error) {
	key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyFile, err := os.Create(keyPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = privateKeyFile.Close() }()

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(privateKeyFile, privateKeyPEM)
	return
}
