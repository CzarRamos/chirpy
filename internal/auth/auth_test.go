package auth_test

import (
	"strings"
	"testing"

	"github.com/CzarRamos/chirpy/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashing(t *testing.T) {
	rawPassword := "go@Testing!2453"
	want, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 4)
	if err != nil {
		t.Errorf(`bcrypt.GenerateFromPassword = %v`, err)
	}
	output, err := auth.HashPassword(rawPassword)
	if err != nil {
		t.Errorf(`bcrypt.GenerateFromPassword = %v`, err)
	}
	isMatch := strings.Compare(string(want), string(output))
	if isMatch > 0 {
		t.Errorf(`bcrypt.GenerateFromPassword = %q, %v, want match for %#q, nil`, output, err, want)
	}
}
