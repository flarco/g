package gutil

import (
	"context"
	"crypto/tls"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v3"
	"gopkg.in/gomail.v2"
)

var (
	// Mg is Mailgun object
	Mg = mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_API_KEY"))
)

// SendEmailMG sends an email with MailGun
func SendEmailMG(msg *mailgun.Message) (string, error) {
	if os.Getenv("TESTING") == "TRUE" {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := Mg.Send(ctx, msg)
	return id, err
}

// AlertEmail is the email address to send errors to
var AlertEmail = os.Getenv("ALERT_EMAIL")

// SMTPServer is email SMTP server host
var SMTPServer = "smtp.gmail.com"

// SMTPPort is email SMTP server port
var SMTPPort = 465

// SMTPUser is SMTP user name
var SMTPUser = os.Getenv("SMTP_USER")

// SMTPPass is user password
var SMTPPass = os.Getenv("SMTP_PASSWORD")

// SendMail sends an email to the specific email address
// https://godoc.org/gopkg.in/gomail.v2#example-package
func SendMail(from string, to []string, subject string, textHTML string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to...)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan")
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", textHTML)
	// m.Attach("/home/Alex/lolcat.jpg")

	d := gomail.NewDialer(SMTPServer, SMTPPort, SMTPUser, SMTPPass)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// Send the email
	err := d.DialAndSend(m)
	return err
}
