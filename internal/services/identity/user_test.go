package identity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/swopcart/server/internal/services/identity"
	"github.com/swopcart/server/internal/testkit"
)

const DummyPassword = "Fo0b4rBaz"

func TestCreateUser(t *testing.T) {
	h := testkit.New(t)
	user, err := h.Services.Identity.CreateUser(t.Context(), "itisrazza", DummyPassword, true)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "itisrazza", user.Username(), "username should be the one passed in")
	assert.Equal(t, true, user.Admin(), "admin status should be the same as passed in")

	err = user.CheckPassword(DummyPassword)
	assert.NoError(t, err, "password should be the one passed in")
}

func TestCreateUser_UserNameClash(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	_, err := id.CreateUser(t.Context(), "itisrazza", DummyPassword, true)
	if !assert.NoError(t, err) {
		return
	}

	_, err = id.CreateUser(t.Context(), "itisrazza", DummyPassword, true)
	assert.ErrorIs(t, err, identity.ErrUserAlreadyExists)
}

func TestCreateUser_InvalidUsernamePassword(t *testing.T) {
	assert := assert.New(t)
	h := testkit.New(t)
	id := h.Services.Identity

	_, err := id.CreateUser(t.Context(), "0a@", "abc", false)
	assert.ErrorIs(err, identity.ErrUsernameTooShort)
	assert.ErrorIs(err, identity.ErrUsernameInvalidChars)
	assert.ErrorIs(err, identity.ErrUsernameStartsWithDigit)
	assert.ErrorIs(err, identity.ErrPasswordTooShort)
	assert.ErrorIs(err, identity.ErrPasswordNotComplex)
}

func TestCreateUser_InvalidUsernamePassword_Long(t *testing.T) {
	assert := assert.New(t)
	h := testkit.New(t)
	id := h.Services.Identity

	_, err := id.CreateUser(t.Context(),
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
		"abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
		false,
	)
	assert.ErrorIs(err, identity.ErrUsernameTooLong)
	assert.ErrorIs(err, identity.ErrPasswordTooLong)
}

func TestGetUserByUsername(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	created, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	found, err := id.GetUserByUsername(t.Context(), "testuser")
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, created.UUID(), found.UUID())
	assert.Equal(t, "testuser", found.Username())
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	_, err := id.GetUserByUsername(t.Context(), "nonexistent")
	assert.ErrorIs(t, err, identity.ErrNotFound)
}

func TestGetUserByID(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	created, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	found, err := id.GetUserByID(t.Context(), created.ID())
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, created.ID(), found.ID())
	assert.Equal(t, created.UUID(), found.UUID())
	assert.Equal(t, "testuser", found.Username())
}

func TestGetUserByID_NotFound(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	_, err := id.GetUserByID(t.Context(), 999999)
	assert.ErrorIs(t, err, identity.ErrNotFound)
}

func TestGetUserByUUID(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	created, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	found, err := id.GetUserByUUID(t.Context(), created.UUID())
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, created.UUID(), found.UUID())
	assert.Equal(t, "testuser", found.Username())
}

func TestGetUserByUUID_NotFound(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	_, err := id.GetUserByUUID(t.Context(), uuid.New())
	assert.ErrorIs(t, err, identity.ErrNotFound)
}

func TestCheckPassword_Incorrect(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	err = user.CheckPassword("WrongPassword123")
	assert.ErrorIs(t, err, identity.ErrIncorrectPassword)
}

func TestSetUsername(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "oldname", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	err = user.SetUsername(t.Context(), "newname")
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "newname", user.Username())

	// Verify the change persisted by fetching again
	found, err := id.GetUserByUUID(t.Context(), user.UUID())
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "newname", found.Username())
}

func TestSetUsername_Invalid(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "validuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	err = user.SetUsername(t.Context(), "ab")
	assert.ErrorIs(t, err, identity.ErrUsernameTooShort)

	// Username should remain unchanged
	assert.Equal(t, "validuser", user.Username())
}

func TestSetPassword(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	newPassword := "NewP4ssword"
	err = user.SetPassword(t.Context(), newPassword)
	if !assert.NoError(t, err) {
		return
	}

	// Old password should no longer work
	err = user.CheckPassword(DummyPassword)
	assert.ErrorIs(t, err, identity.ErrIncorrectPassword)

	// New password should work
	err = user.CheckPassword(newPassword)
	assert.NoError(t, err)
}

func TestSetPassword_Invalid(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	err = user.SetPassword(t.Context(), "short")
	assert.ErrorIs(t, err, identity.ErrPasswordTooShort)

	// Original password should still work
	err = user.CheckPassword(DummyPassword)
	assert.NoError(t, err)
}

func TestSetAdmin(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	user, err := id.CreateUser(t.Context(), "testuser", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	assert.False(t, user.Admin())

	err = user.SetAdmin(t.Context(), true)
	if !assert.NoError(t, err) {
		return
	}

	assert.True(t, user.Admin())

	// Verify by fetching again
	found, err := id.GetUserByUUID(t.Context(), user.UUID())
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, found.Admin())

	// Toggle back to false
	err = user.SetAdmin(t.Context(), false)
	if !assert.NoError(t, err) {
		return
	}
	assert.False(t, user.Admin())
}

func TestGetAllUsers_DefaultAdmin(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	// The identity service creates a default admin user on initialization
	users, count, err := id.GetAllUsers(t.Context(), 10, 0)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, uint(1), count)
	assert.Len(t, users, 1)
	assert.Equal(t, identity.DefaultAdminUsername, users[0].Username())
}

func TestGetAllUsers(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	// Create additional users (default admin already exists)
	_, err := id.CreateUser(t.Context(), "user1", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}
	_, err = id.CreateUser(t.Context(), "user2", DummyPassword, true)
	if !assert.NoError(t, err) {
		return
	}
	_, err = id.CreateUser(t.Context(), "user3", DummyPassword, false)
	if !assert.NoError(t, err) {
		return
	}

	users, count, err := id.GetAllUsers(t.Context(), 10, 0)
	if !assert.NoError(t, err) {
		return
	}

	// 1 default admin + 3 created = 4 total
	assert.Equal(t, uint(4), count)
	assert.Len(t, users, 4)
}

func TestGetAllUsers_Limit(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	// Create 3 additional users (default admin already exists)
	for i := 1; i <= 3; i++ {
		_, err := id.CreateUser(t.Context(), "user"+string(rune('0'+i)), DummyPassword, false)
		if !assert.NoError(t, err) {
			return
		}
	}

	users, count, err := id.GetAllUsers(t.Context(), 2, 0)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, uint(4), count, "total count should be 4 (1 admin + 3 created)")
	assert.Len(t, users, 2, "should only return 2 users due to limit")
}

func TestGetAllUsers_Offset(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	// Create 3 additional users (default admin already exists)
	for i := 1; i <= 3; i++ {
		_, err := id.CreateUser(t.Context(), "user"+string(rune('0'+i)), DummyPassword, false)
		if !assert.NoError(t, err) {
			return
		}
	}

	users, count, err := id.GetAllUsers(t.Context(), 10, 2)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, uint(4), count, "total count should be 4 (1 admin + 3 created)")
	assert.Len(t, users, 2, "should return 2 users after offset of 2")
}

func TestGetAllUsers_LimitAndOffset(t *testing.T) {
	h := testkit.New(t)
	id := h.Services.Identity

	// Create 5 additional users (default admin already exists)
	for i := 1; i <= 5; i++ {
		_, err := id.CreateUser(t.Context(), "user"+string(rune('0'+i)), DummyPassword, false)
		if !assert.NoError(t, err) {
			return
		}
	}

	users, count, err := id.GetAllUsers(t.Context(), 2, 1)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, uint(6), count, "total count should be 6 (1 admin + 5 created)")
	assert.Len(t, users, 2, "should return 2 users with limit=2, offset=1")
}
