package net

import (
	"bufio"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/flarco/g"
)

// DownloadFile downloads a file
func DownloadFile(url string, filepath string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return g.Error(err, "Unable to Create file "+filepath)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return g.Error(err, "Unable to Reach URL: "+url)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode >= 400 {
		return g.Error("Bad Status '%s' from URL %s", resp.Status, url)
	}

	// Writer the body to file
	bw, err := io.Copy(out, resp.Body)
	if err != nil || bw == 0 {
		return g.Error(err, "Unable to write to file "+filepath)
	}

	return nil
}

// ClientDoStream Http client method execution returning a reader
func ClientDoStream(method, URL string, body io.Reader, headers map[string]string) (resp *http.Response, reader io.Reader, err error) {
	g.Trace("%s -> %s", method, URL)
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, g.Error(err, "could not %s @ %s", method, URL)
	}
	if headers == nil {
		headers = map[string]string{}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := http.Client{}

	resp, err = client.Do(req)
	if err != nil {
		err = g.Error(err, "could not perform request")
		return
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		respBytes, _ := io.ReadAll(resp.Body)
		err = g.Error("Unexpected Response %d: %s. %s", resp.StatusCode, resp.Status, string(respBytes))
		return
	}

	reader = bufio.NewReader(resp.Body)

	return
}

// ClientDo Http client method execution
func ClientDo(method, URL string, body io.Reader, headers map[string]string, timeOut ...int) (resp *http.Response, respBytes []byte, err error) {
	respBytes = []byte("")
	to := 30 * time.Second
	if len(timeOut) > 0 {
		to = time.Duration(timeOut[0]) * time.Second
	}

	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, g.Error(err, "could not %s @ %s", method, URL)
	}
	if headers == nil {
		headers = map[string]string{}
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := http.Client{Timeout: to}

	resp, err = client.Do(req)
	if err != nil {
		if resp == nil {
			switch {
			case strings.Contains(err.Error(), "server misbehaving"):
				errorMsg := g.ErrMsg(err)
				// create fake http response with 502 error code
				resp = &http.Response{
					Status:        "DNS Server Misbehaving",
					StatusCode:    502,
					Body:          io.NopCloser(strings.NewReader(errorMsg)),
					Header:        http.Header{"Content-Type": []string{"text/plain"}},
					Proto:         "HTTP/1.1",
					ProtoMajor:    1,
					ProtoMinor:    1,
					ContentLength: int64(len(errorMsg)),
					Request:       req,
				}
			case strings.Contains(err.Error(), "Client.Timeout"):
				errorMsg := g.ErrMsg(err)
				// create fake http response with 502 error code
				resp = &http.Response{
					Status:        "Request Timed-out",
					StatusCode:    504,
					Body:          io.NopCloser(strings.NewReader(errorMsg)),
					Header:        http.Header{"Content-Type": []string{"text/plain"}},
					Proto:         "HTTP/1.1",
					ProtoMajor:    1,
					ProtoMinor:    1,
					ContentLength: int64(len(errorMsg)),
					Request:       req,
				}
			}
		}
		err = g.Error(err, "could not perform request")
		return
	}

	respBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		err = g.Error(err, "could not read from request body")
	}

	if resp.StatusCode >= 400 || resp.StatusCode < 200 {
		err = g.Error("Unexpected Response %d: %s. %s", resp.StatusCode, resp.Status, string(respBytes))
		return
	}

	return
}
