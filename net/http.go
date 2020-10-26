package net

import (
	"bufio"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/flarco/gutil"
)

// DownloadFile downloads a file
func DownloadFile(url string, filepath string) (err error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return gutil.Error(err, "Unable to Create file "+filepath)
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return gutil.Error(err, "Unable to Reach URL: "+url)
	}
	defer resp.Body.Close()

	// Check server response
	if resp.StatusCode != http.StatusOK {
		return gutil.Error(err, gutil.F("Bad Status '%s' from URL %s", resp.Status, url))
	}

	// Writer the body to file
	bw, err := io.Copy(out, resp.Body)
	if err != nil || bw == 0 {
		return gutil.Error(err, "Unable to write to file "+filepath)
	}

	return nil
}

// ClientDoStream Http client method execution returning a reader
func ClientDoStream(method, URL string, body io.Reader, headers map[string]string) (resp *http.Response, reader io.Reader, err error) {
	gutil.Trace("%s -> %s", method, URL)
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, gutil.Error(err, "could not %s @ %s", method, URL)
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
		err = gutil.Error(err, "could not perform request")
		return
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		respBytes, _ := ioutil.ReadAll(resp.Body)
		err = gutil.Error("Unexpected Response %d: %s. %s", resp.StatusCode, resp.Status, string(respBytes))
		return
	}

	reader = bufio.NewReader(resp.Body)

	return
}

// ClientDo Http client method execution
func ClientDo(method, URL string, body io.Reader, headers map[string]string, timeOut ...int) (resp *http.Response, respBytes []byte, err error) {
	to := 3600 * time.Second
	if len(timeOut) > 0 {
		to = time.Duration(timeOut[0]) * time.Second
	}

	gutil.Trace("%s -> %s", method, URL)
	req, err := http.NewRequest(method, URL, body)
	if err != nil {
		return nil, nil, gutil.Error(err, "could not %s @ %s", method, URL)
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
		err = gutil.Error(err, "could not perform request")
		return
	}

	respBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = gutil.Error(err, "could not read from request body")
	}

	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		err = gutil.Error("Unexpected Response %d: %s. %s", resp.StatusCode, resp.Status, string(respBytes))
		return
	}

	return
}
