package domain

import (
	"net/mail"
	"strings"
	"time"
)

// User is someone with an account. The same account can give materials
// away and request them; Role says what kind of receiver they are.
type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Area      string    `json:"area"`
	CreatedAt time.Time `json:"createdAt"`
}

// Signup is what a new user sends to create an account.
type Signup struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Area     string `json:"area"`
}

// MinPasswordLen is the shortest password accepted at sign-up.
const MinPasswordLen = 8

// NormalizeEmail lower-cases and trims an address so lookups match.
func NormalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// ValidateSignup trims the form and checks every field.
func ValidateSignup(s *Signup) error {
	s.Name = strings.TrimSpace(s.Name)
	s.Email = NormalizeEmail(s.Email)
	switch {
	case s.Name == "" || len(s.Name) > 80:
		return invalidf("add your name or your organisation's name")
	case !validEmail(s.Email):
		return invalidf("enter a valid email address")
	case ValidatePassword(s.Password) != nil:
		return ValidatePassword(s.Password)
	case !ValidRole(s.Role):
		return invalidf("choose who you are")
	case !ValidArea(s.Area):
		return invalidf("choose your area")
	}
	return nil
}

func validEmail(s string) bool {
	if len(s) > 254 {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s && strings.Contains(s[strings.LastIndex(s, "@"):], ".")
}

// ValidateProfile trims and checks the fields a user can change in Settings.
func ValidateProfile(name, area *string) error {
	*name = strings.TrimSpace(*name)
	switch {
	case *name == "" || len(*name) > 80:
		return invalidf("add your name or your organisation's name")
	case !ValidArea(*area):
		return invalidf("choose your area")
	}
	return nil
}

// ValidatePassword checks a new password.
func ValidatePassword(pw string) error {
	switch {
	case len(pw) < MinPasswordLen:
		return invalidf("use a password of at least %d characters", MinPasswordLen)
	case len(pw) > 200:
		return invalidf("that password is too long")
	}
	return nil
}
