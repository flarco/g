package net

import (
	"net/url"
	"strings"

	"github.com/spf13/cast"
)

// URL is a url instance
type URL struct {
	U          *url.URL
	OrigURL    string
	extraHosts string // stripped seed-list hosts (MongoDB), re-inserted by String()
}

// NewURL creates a new URL instance
func NewURL(urlStr string) (*URL, error) {
	if strings.Contains(urlStr, `:\`) {
		urlStr = strings.ReplaceAll(urlStr, `\`, `/`) // windows path fix
	}

	// reduce MongoDB-style multi-host authorities to their first host so
	// url.Parse accepts them (rejected under Go 1.26+); OrigURL keeps the full URL
	parseStr, extraHosts := reduceMultiHostAuthority(urlStr)

	u, err := url.Parse(parseStr)
	if err != nil {
		return &URL{OrigURL: urlStr}, err
	}
	return &URL{U: u, OrigURL: urlStr, extraHosts: extraHosts}, nil
}

// reduceMultiHostAuthority collapses a comma-separated multi-host authority
// (MongoDB seed lists) to its first host, returning the reduced URL and the
// stripped extra hosts (comma-prefixed). Non-multi-host URLs pass through.
func reduceMultiHostAuthority(urlStr string) (reduced, extraHosts string) {
	sep := "://"
	i := strings.Index(urlStr, sep)
	if i < 0 {
		return urlStr, ""
	}
	rest := urlStr[i+len(sep):]

	end := len(rest)
	if j := strings.IndexAny(rest, "/?#"); j >= 0 {
		end = j
	}
	authority := rest[:end]

	userinfo := ""
	hosts := authority
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		userinfo = authority[:at+1]
		hosts = authority[at+1:]
	}

	comma := strings.Index(hosts, ",")
	if comma < 0 {
		return urlStr, ""
	}

	return urlStr[:i+len(sep)] + userinfo + hosts[:comma] + rest[end:], hosts[comma:]
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
	s := u.U.String()
	if u.extraHosts != "" && u.U.Host != "" {
		if idx := strings.Index(s, u.U.Host); idx >= 0 {
			insertAt := idx + len(u.U.Host)
			s = s[:insertAt] + u.extraHosts + s[insertAt:]
		}
	}
	return s
}
