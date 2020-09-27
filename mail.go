package gutil

import (
	"context"
	"github.com/mailgun/mailgun-go/v3"
	"os"
	"time"
)

var (
	// Mg is Mailgun object
	Mg = mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_API_KEY"))
)

// SendEmailMG sends an email with MailGun
func SendEmailMG(msg *mailgun.Message) (string, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := Mg.Send(ctx, msg)
	return id, err
}
