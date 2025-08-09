package auth_test

import (
	"testing"

	"github.com/CzarRamos/chirpy/internal/auth"
	"github.com/google/uuid"
)

func TestGeneralPasswordHashing(t *testing.T) {
	originalPassword := "my-super-secure-password"

	output, err := auth.HashPassword(originalPassword)
	if err != nil {
		t.Errorf(`error hashing password did not work: %v`, err)
		return
	}

	err = auth.CheckPasswordHash(originalPassword, output)
	if err != nil {
		t.Errorf(`CheckPasswordHash failed with the correct password: %v`, err)
		return
	}

	err = auth.CheckPasswordHash("some-random-password", output)
	if err == nil {
		t.Errorf(`CheckPasswordHash should have failed with the incorrect password: %v`, err)
		return
	}
}

func TestPasswordHashingTwice(t *testing.T) {
	originalPassword := "my-super-secure-password"

	// same password, just hashed at different moments
	hash1, err := auth.HashPassword(originalPassword)
	if err != nil {
		t.Errorf(`error hashing password did not work: %v`, err)
		return
	}

	hash2, err := auth.HashPassword(originalPassword)
	if err != nil {
		t.Errorf(`error hashing password did not work: %v`, err)
		return
	}

	// both hashes should work
	err = auth.CheckPasswordHash(originalPassword, hash1)
	if err != nil {
		t.Errorf(`CheckPasswordHash failed with the first hash verison of the password: %v`, err)
		return
	}

	err = auth.CheckPasswordHash("some-random-password", hash2)
	if err == nil {
		t.Errorf(`CheckPasswordHash failed with a second hash verison of the password %v`, err)
		return
	}
}

func TestInvalidPasswordLengthHashing(t *testing.T) {
	emptyPassword := ""
	veryLongPassword := "A9v!X2blrmZP0qsd4eT7fyUjKDfghjQWERTYUIOPasdfghjklzxcvbnm1234567890!@#$%^&*()_+QWERTY"

	// same password, just hashed at different moments
	_, err := auth.HashPassword(emptyPassword)
	if err == nil {
		t.Errorf(`HashPassword should not have succeeded hashing an empty password: %v`, err)
		return
	}

	_, err = auth.HashPassword(veryLongPassword)
	if err == nil {
		t.Errorf(`HashPassword should not have succeeded hashing a password over 72 bytes: %v`, err)
		return
	}
}

func TestJWTValidToken(t *testing.T) {
	//generate uuid
	userID := uuid.New()
	//token secret
	tokenSecret := "this-is-my-secret-token"
	newJWT, err := auth.MakeJWT(userID, tokenSecret)
	if err != nil {
		t.Errorf(`MakeJWT failed: %v`, err)
		return
	}

	output, err := auth.ValidateJWT(newJWT, tokenSecret)
	if err != nil {
		t.Errorf(`ValidateJWT failed: %v`, err)
		return
	}

	if output != userID {
		t.Errorf(`ValidateJWT returned wrong userID: got %v, want %v`, output, userID.String())
		return
	}
}

func TestWrongSecretKey(t *testing.T) {
	//generate uuid
	userID := uuid.New()
	//token secret
	tokenSecret := "this-is-my-secret-token"
	// some token secret for something else
	differentTokenSecret := "this-is-a-different-secret-token"
	newJWT, err := auth.MakeJWT(userID, tokenSecret)
	if err != nil {
		t.Errorf(`MakeJWT failed: %v`, err)
		return
	}

	_, err = auth.ValidateJWT(newJWT, differentTokenSecret)
	if err == nil {
		t.Errorf(`ValidateJWT should have rejected when validating with a different key than initialization`)
	}
}

func TestValidatingInvalidStrings(t *testing.T) {

	//token secret
	tokenSecret := "this-is-my-secret-token"

	_, err := auth.ValidateJWT("a-made-up-token", tokenSecret)
	if err == nil {
		t.Errorf(`ValidateJWT should have rejected non-existent token`)
	}

	// test with empty string
	_, err = auth.ValidateJWT("", tokenSecret)
	if err == nil {
		t.Errorf(`ValidateJWT should have rejected validating an empty string`)
	}
}

func TestCorruptedToken(t *testing.T) {

	//generate uuid
	userID := uuid.New()
	//token secret
	tokenSecret := "this-is-my-secret-token"
	newJWT, err := auth.MakeJWT(userID, tokenSecret)
	if err != nil {
		t.Errorf(`MakeJWT failed: %v`, err)
		return
	}

	// corruption happens
	corruptedJWT := newJWT + "i-am-inflitrating-the-token"

	_, err = auth.ValidateJWT(corruptedJWT, tokenSecret)
	if err == nil {
		t.Errorf(`ValidateJWT should have rejected a corrupted/modified token`)
		return
	}
}
