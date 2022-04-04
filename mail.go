package g

import (
	"crypto/tls"
	"os"

	"gopkg.in/gomail.v2"
)

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
