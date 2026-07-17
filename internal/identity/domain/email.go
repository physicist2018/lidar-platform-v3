package domain

import "net/mail"

// Email is a value object that holds a validated email address.
type Email struct {
	value string
}

// NewEmail validates and creates an Email value object.
func NewEmail(email string) (Email, error) {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return Email{}, ErrInvalidEmail
	}
	return Email{value: email}, nil
}

// NewEmailUnsafe creates an Email without validation. Use only when the value
// is already trusted (e.g. read from the database).
func NewEmailUnsafe(email string) Email {
	return Email{value: email}
}

// String returns the email as a plain string.
func (e Email) String() string {
	return e.value
}

// Equals compares two Email value objects.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}
