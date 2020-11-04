package net

import (
	"os"
	"testing"

	"github.com/alecthomas/assert"
)

func TestEmail(t *testing.T) {
	s := NewSMTP(
		"smtp.gmail.com",
		465,
		os.Getenv("GOOGLE_USER"),
		os.Getenv("GOOGLE_PASSWORD"),
		os.Getenv("GOOGLE_USER"),
		true,
	)
	m := Email{
		To:       []string{os.Getenv("GOOGLE_USER")},
		Subject:  "Test Email",
		TextBody: "This is a test",
	}
	err := s.Send(m)
	assert.NoError(t, err)
}
