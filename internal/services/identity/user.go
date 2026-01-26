package identity

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/swopcart/server/internal/database"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	service *IdentityService
	uuid    uuid.UUID
	data    database.User
}

var ErrUpdateFailed = errors.New("user does not exist, or something worse happened")

var (
	ErrIncorrectPassword = errors.New("incorrect password")
	ErrUserAlreadyExists = errors.New("user already exists")
)

const (
	MinUsernameLength = 4
	MaxUsernameLength = 64
)

var ErrUsernameTooShort = errors.New("username is too short")
var ErrUsernameTooLong = errors.New("username is too long")
var ErrUsernameInvalidChars = errors.New("username contains invalid characters")
var ErrUsernameStartsWithDigit = errors.New("username cannot start with a digit")

const (
	MinPasswordLength = 8
	MaxPasswordLength = 72
)

var ErrPasswordTooShort = errors.New("password is too short")
var ErrPasswordTooLong = errors.New("password is too long")
var ErrPasswordNotComplex = errors.New("password must contain a lowercase letter, uppercase letter, and digit")

var ErrNotFound = errors.New("not found")
var ErrInternal = errors.New("internal error")

var ErrInvalidTOTP = errors.New("invalid TOTP token")

func (svc *IdentityService) newUser(user database.User) *User {
	return &User{
		service: svc,
		uuid:    user.UUID,
		data:    user,
	}
}

func (svc *IdentityService) CreateUser(
	ctx context.Context,
	username, password string,
	admin bool,
) (*User, error) {
	err := validateUsername(username)
	err = errors.Join(err, validatePassword(password))
	if err != nil {
		return nil, err
	}

	_, err = svc.GetUserByUsername(ctx, username)
	if err == nil {
		return nil, ErrUserAlreadyExists
	} else if !errors.Is(err, ErrNotFound) {
		return nil, errors.Join(ErrInternal, err)
	}

	userUUID := uuid.New()

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := database.User{
		UUID:     userUUID,
		Username: username,
		Password: passwordHash,
		Admin:    admin,
	}

	err = gorm.G[database.User](svc.db).Create(ctx, &user)
	if err != nil {
		return nil, err
	}

	return svc.newUser(user), nil
}

func (svc *IdentityService) GetUserByID(ctx context.Context, id uint) (*User, error) {
	user, err := gorm.G[database.User](svc.db).
		Where("id = ?", id).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return svc.newUser(user), nil
}

func (svc *IdentityService) GetUserByUUID(ctx context.Context, uuid uuid.UUID) (*User, error) {
	user, err := gorm.G[database.User](svc.db).
		Where("uuid = ?", uuid).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return svc.newUser(user), nil
}

func (svc *IdentityService) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	user, err := gorm.G[database.User](svc.db).
		Where("username = ?", username).
		First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return svc.newUser(user), nil
}

func (svc *IdentityService) GetAllUsers(ctx context.Context, limit, offset int) ([]*User, uint, error) {
	usersQuery := gorm.G[database.User](svc.db)

	count, err := usersQuery.Count(ctx, "id")
	if err != nil {
		return nil, 0, err
	}

	dbUsers, err := usersQuery.Order("created_at DESC").Offset(offset).Limit(limit).Find(ctx)
	if err != nil {
		return nil, 0, err
	}

	users := make([]*User, 0, len(dbUsers))
	for _, dbUser := range dbUsers {
		users = append(users, svc.newUser(dbUser))
	}

	return users, uint(count), nil
}

func (u *User) ID() uint {
	return u.data.ID
}

func (u *User) UUID() uuid.UUID {
	return u.data.UUID
}

func (u *User) Username() string {
	return u.data.Username
}

func (u *User) SetUsername(ctx context.Context, username string) error {
	err := validateUsername(username)
	if err != nil {
		return err
	}

	rowsAffected, err := u.gormChain().Update(ctx, "username", username)
	if err != nil {
		// TODO: error handling
		return err
	}

	if rowsAffected != 1 {
		return ErrUpdateFailed
	}

	return u.Reload(ctx)
}

func validateUsername(username string) (err error) {
	if len(username) < MinUsernameLength {
		err = errors.Join(err, ErrUsernameTooShort)
	}

	if len(username) > MaxUsernameLength {
		err = errors.Join(err, ErrUsernameTooLong)
	}

	if len(username) > 0 && username[0] >= '0' && username[0] <= '9' {
		err = errors.Join(err, ErrUsernameStartsWithDigit)
	}

	for _, c := range username {
		if !isValidUsernameChar(c) {
			err = errors.Join(err, ErrUsernameInvalidChars)
			break
		}
	}

	return err
}

func isValidUsernameChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}

func (u *User) CheckPassword(password string) error {
	return checkPassword(u.data.Password, password)
}

func (u *User) SetPassword(ctx context.Context, password string) error {
	err := validatePassword(password)
	if err != nil {
		return err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return err
	}

	rowsAffected, err := u.gormChain().Update(ctx, "password", passwordHash)
	if err != nil {
		// TODO: error handling
		return err
	}

	if rowsAffected != 1 {
		return ErrUpdateFailed
	}

	return u.Reload(ctx)
}

func validatePassword(password string) (err error) {
	if len(password) < MinPasswordLength {
		err = errors.Join(err, ErrPasswordTooShort)
	}

	if len(password) > MaxPasswordLength {
		err = errors.Join(err, ErrPasswordTooLong)
	}

	var hasLower, hasUpper, hasDigit bool
	for _, c := range password {
		switch {
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= '0' && c <= '9':
			hasDigit = true
		}
	}
	if !hasLower || !hasUpper || !hasDigit {
		err = errors.Join(err, ErrPasswordNotComplex)
	}

	return
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		return "", ErrPasswordTooLong
	} else if err != nil {
		return "", errors.Join(ErrInternal, err)
	}

	return string(hash), nil
}

func checkPassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrIncorrectPassword
	} else if err != nil {
		return errors.Join(ErrInternal, err)
	}

	return nil
}

func (u *User) CheckTOTP(token string) error {
	if u.data.TOTP == nil {
		return nil
	}

	if !totp.Validate(token, *u.data.TOTP) {
		return ErrInvalidTOTP
	}

	return nil
}

func (u *User) EnableTOTP(ctx context.Context, secret, token string) error {
	if !totp.Validate(token, secret) {
		return ErrInvalidTOTP
	}

	_, err := u.gormChain().Update(ctx, "totp", secret)
	if err != nil {
		return err
	}

	return u.Reload(ctx)
}

func (u *User) Admin() bool {
	return u.data.Admin
}

func (u *User) CreatedAt() time.Time {
	return u.data.CreatedAt
}

func (u *User) SetAdmin(ctx context.Context, admin bool) error {
	rowsAffected, err := u.gormChain().Update(ctx, "admin", admin)
	if err != nil {
		// TODO: error handling
		return err
	}

	if rowsAffected != 1 {
		// TODO: user does not exist, or something worse happened
		return ErrUpdateFailed
	}

	return u.Reload(ctx)
}

func (u *User) Reload(ctx context.Context) error {
	user, err := gorm.G[database.User](u.service.db).
		Where("uuid = ?", u.uuid).
		First(ctx)
	if err != nil {
		return err
	}

	u.data = user
	return nil
}

func (u *User) gormChain() gorm.ChainInterface[database.User] {
	return gorm.G[database.User](u.service.db).
		Where("uuid = ?", u.uuid)
}
