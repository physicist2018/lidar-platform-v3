package domain

import "golang.org/x/crypto/bcrypt"

const minPasswordLen = 8

// Password is a value object that holds a bcrypt-hashed password.
type Password struct {
	hash string
}

// NewPassword validates strength and hashes the plain-text password.
func NewPassword(plain string) (Password, error) {
	if len(plain) < minPasswordLen {
		return Password{}, ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return Password{}, err
	}
	return Password{hash: string(hash)}, nil
}

// Hash returns the bcrypt hash string.
func (p Password) Hash() string {
	return p.hash
}

// Compare checks whether the plain-text password matches the hash.
func (p Password) Compare(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plain)) == nil
}
