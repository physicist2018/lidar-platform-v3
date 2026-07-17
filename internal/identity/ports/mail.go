package ports

import "context"

// MailSender defines the contract for sending emails.
type MailSender interface {
	SendVerificationEmail(ctx context.Context, email, token string) error
}
