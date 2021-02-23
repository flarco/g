package net

import (
	"os"
	"testing"

	"github.com/alecthomas/assert"
)

var htmltext = `<!DOCTYPE html>
<html>
<head>

  <meta charset="utf-8">
  <meta http-equiv="x-ua-compatible" content="ie=edge">
  <title>Password Reset</title>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	
	<h1> this is a test </h1>

</body>
</html>`

func TestEmail(t *testing.T) {
	var user, password string
	user = os.Getenv("GOOGLE_USER")
	password = os.Getenv("GOOGLE_PASSWORD")
	s := NewSMTP(
		"smtp.gmail.com", 465, user,
		password, user, true,
	)
	m := Email{
		To:       []string{user},
		Subject:  "Test Email",
		TextBody: "This is a test",
		HTMLBody: htmltext,
	}
	err := s.Send(m)
	assert.NoError(t, err)
}
