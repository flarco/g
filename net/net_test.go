package net

import (
	"os"
	"testing"

	"github.com/flarco/g"
	"github.com/stretchr/testify/assert"
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
func TestURL(t *testing.T) {
	urlStr := "s3://ocral/LargeDataset.csv.gz"
	u, err := NewURL(urlStr)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "ocral", u.Hostname())
	assert.Equal(t, "/LargeDataset.csv.gz", u.Path())

	urlStr = "file:///gaf"

	u, err = NewURL(urlStr)
	g.Info(u.Hostname())
	g.Info(u.Path())

	// MongoDB seed-list URL (multi-host authority, rejected by url.Parse on Go 1.26+)
	urlStr = "mongodb://user:pass@h0.mongodb.net:27017,h1.mongodb.net:27017,h2.mongodb.net:27017/?ssl=true&replicaSet=atlas-abc-shard-0&authSource=admin"
	u, err = NewURL(urlStr)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "h0.mongodb.net", u.Hostname())
	assert.Equal(t, 27017, u.Port())
	assert.Equal(t, "user", u.Username())
	assert.Equal(t, "pass", u.Password())
	assert.Equal(t, "true", u.Query()["ssl"])
	assert.Equal(t, "atlas-abc-shard-0", u.Query()["replicaSet"])
	assert.Equal(t, urlStr, u.OrigURL)   // full seed list preserved
	assert.Equal(t, urlStr, u.String())  // String() re-inserts stripped hosts
}

func TestURLErrorRedactsPassword(t *testing.T) {
	tests := []string{
		"oracle://user:s3cr3t@(DESCRIPTION = (ADDRESS = (HOST = h)))",
		"oracle://user:s3c@r:e/t#?@(DESCRIPTION = (HOST = h))",
		"postgres://user:s3cr3t@bad host:5432/db",
	}
	for _, urlStr := range tests {
		_, err := NewURL(urlStr)
		if !assert.Error(t, err, urlStr) {
			continue
		}
		assert.NotContains(t, err.Error(), "s3cr3t")
		assert.NotContains(t, err.Error(), "s3c@r")
		assert.NotContains(t, err.Error(), "e/t#?")
		assert.Contains(t, err.Error(), "user:***@")
	}

	// URLs without a password stay as is
	_, err := NewURL("http://bad host/path")
	assert.ErrorContains(t, err, "http://bad host/path")
}
