package net

import (
	"os"
	"strings"
	"time"

	"github.com/flarco/g"
	"gopkg.in/gomail.v2"
)

// SMTP is for emailing
type SMTP struct {
	Host       string
	Port       int
	SSL        bool
	User       string
	Password   string
	From       string
	EmailQueue []*gomail.Message
}

// Email is a single email message
type Email struct {
	Subject  string   `json:"subject"`
	To       []string `json:"to"`
	Cc       []string `json:"cc,omitempty"`
	Bcc      []string `json:"bcc,omitempty"`
	HTMLBody string   `json:"html_body,omitempty"`
	TextBody string   `json:"text_body,omitempty"`
	Files    []string `json:"files,omitempty"` // File paths to attach
}

// NewSMTP returns an SMTP object
func NewSMTP(host string, port int, user, password, from string, ssl bool) *SMTP {
	return &SMTP{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   password,
		SSL:        ssl,
		From:       from,
		EmailQueue: []*gomail.Message{},
	}
}

// parseAddress parses an email address
// `Jane Doe <jane.doe@gmail.com>` should return
// `"Jane Doe", "jane.doe@gmail.com"
func (s *SMTP) parseAddress(addr string) (email, name string) {
	arr := strings.Split(addr, "<")
	if len(arr) != 2 {
		return addr, ""
	}
	name = strings.TrimSpace(arr[0])
	email = strings.TrimSuffix(strings.TrimSpace(arr[1]), ">")
	return
}

func (s *SMTP) setRecipient(m *gomail.Message, field string, list []string) {
	if len(list) == 1 {
		email, name := s.parseAddress(list[0])
		if name != "" {
			m.SetAddressHeader(field, email, name)
		} else {
			m.SetHeader(field, list...)
		}
	} else if len(list) > 1 {
		m.SetHeader(field, list...)
	}
}

// QueueEmail queues a new email message
func (s *SMTP) QueueEmail(emails ...Email) (err error) {
	for _, e := range emails {
		m := gomail.NewMessage()
		m.SetHeader("From", s.From)
		m.SetHeader("Subject", e.Subject)
		s.setRecipient(m, "To", e.To)
		s.setRecipient(m, "Cc", e.Cc)
		s.setRecipient(m, "Bcc", e.Bcc)
		if e.HTMLBody != "" {
			m.SetBody("text/html", e.HTMLBody)
		} else {
			m.SetBody("text/plain", e.TextBody)
		}
		for _, f := range e.Files {
			m.Attach(f)
		}
		s.EmailQueue = append(s.EmailQueue, m)
	}
	return
}

// Send sends all email messages
func (s *SMTP) Send(emails ...Email) (err error) {
	s.QueueEmail(emails...)
	return s.SendQueue()
}

// SendQueue sends all email messages in queue
func (s *SMTP) SendQueue() (err error) {
	if os.Getenv("TESTING") == "TRUE" {
		return nil
	}
	d := gomail.NewDialer(s.Host, s.Port, s.User, s.Password)
	// d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	d.SSL = s.SSL

	// Bound the entire SMTP exchange so a hung relay cannot block the caller forever.
	// gomail's TCP dial has its own 10s timeout, but STARTTLS / DATA can still stall.
	done := make(chan error, 1)
	go func() { done <- d.DialAndSend(s.EmailQueue...) }()
	select {
	case err = <-done:
	case <-time.After(30 * time.Second):
		err = g.Error("smtp send timed out after 30s")
	}
	if err != nil {
		err = g.Error(err, "could not send email")
		return
	}
	// clear the queue
	s.EmailQueue = []*gomail.Message{}
	return
}
