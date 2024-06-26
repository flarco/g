package net

import (
	"net/url"
	"strings"

	"github.com/spf13/cast"
)

// URL is a url instance
type URL struct {
	U       *url.URL
	OrigURL string
}

// NewURL creates a new URL instance
func NewURL(urlStr string) (*URL, error) {
	if strings.Contains(urlStr, `:\`) {
		urlStr = strings.ReplaceAll(urlStr, `\`, `/`) // windows path fix
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return &URL{OrigURL: urlStr}, err
	}
	return &URL{U: u, OrigURL: urlStr}, nil
}

// URL returns the url object
func (u *URL) URL() *url.URL {
	return u.U
}

// Path returns the path of the url
func (u *URL) Path() string {
	return u.U.Path
}

// Port returns the port in the url or provided default
func (u *URL) Port(Default ...int) int {
	port := cast.ToInt(u.U.Port())
	if port == 0 && len(Default) > 0 {
		return Default[0]
	}
	return port
}

// Hostname returns the hostname
func (u *URL) Hostname() string {
	return u.U.Hostname()
}

// Query returns the query as a map
func (u *URL) Query() map[string]string {
	m := map[string]string{}
	for k, arr := range u.U.Query() {
		m[k] = arr[0]
	}
	return m
}

// Username returns the Username
func (u *URL) Username() string {
	return u.U.User.Username()
}

// Password returns the password in the url
func (u *URL) Password() string {
	password, _ := u.U.User.Password()
	return password
}

// AddParam adds a query parameter
func (u *URL) AddParam(key, value string) *URL {
	if u.U == nil {
		return u
	}
	q := u.U.Query()
	q.Set(key, value)
	u.U.RawQuery = q.Encode()
	return u
}

// SetParam sets a query parameter
func (u *URL) SetParam(key, value string) *URL {
	if u.U == nil {
		return u
	}
	q := u.U.Query()
	q.Set(key, value)
	u.U.RawQuery = q.Encode()
	return u
}

// GetParam extracts/removes a query parameter
func (u *URL) GetParam(key string) string {
	if u.U == nil {
		return ""
	}
	q := u.U.Query()
	return q.Get(key)
}

// PopParam extracts/removes a query parameter
func (u *URL) PopParam(key string) string {
	if u.U == nil {
		return ""
	}
	q := u.U.Query()
	value := q.Get(key)
	q.Del(key)
	u.U.RawQuery = q.Encode()
	return value
}

// String returs the string instance
func (u *URL) String() string {
	if u.U == nil {
		return u.OrigURL
	}
	return u.U.String()
}
