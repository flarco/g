package cacheserver

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	g "github.com/flarco/g"
)

// CacheClient is our cache type
type CacheClient struct {
	client *http.Client
	URL    string
}

// NewCacheClient creates a new client instance
func NewCacheClient(URL string) (c *CacheClient) {
	return &CacheClient{
		client: &http.Client{Timeout: 3 * time.Second},
		URL:    URL,
	}
}

// GetBytes gets a key/value pair
func (c *CacheClient) GetBytes(key string) (respBytes []byte, err error) {
	req, _ := http.NewRequest("GET", c.URL+key, nil)
	resp, err := c.client.Do(req)
	if err != nil {
		err = g.Error(err, "could not perform request")
		return
	}

	respBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err = g.Error(err, "could not read from GET request body")
	}

	if len(respBytes) == 0 {
		respBytes = nil
	}

	return
}

// GetStr gets a key/value pair
func (c *CacheClient) GetStr(key string) (val string, err error) {
	respBytes, err := c.GetBytes(key)
	if err != nil {
		err = g.Error(err, "could not read from GET request body")
		return
	} else if respBytes == nil {
		return
	}
	val = string(respBytes)
	return
}

// SetStr saves a key/value pair
func (c *CacheClient) SetStr(key string, valueStr string) (err error) {
	req, _ := http.NewRequest("PUT", c.URL+key, strings.NewReader(valueStr))
	_, err = c.client.Do(req)
	if err != nil {
		err = g.Error(err, "could not perform PUT request")
		return
	}
	return
}

// Set saves a key/value pair
func (c *CacheClient) Set(key string, value interface{}) (err error) {
	jsonString := g.Marshal(value)
	err = c.SetStr(key, jsonString)
	if err != nil {
		err = g.Error(err, "could not set")
	}
	return
}

// Get gets a key/value pair and unmarshals
func (c *CacheClient) Get(key string, objPtr interface{}) (err error) {
	respBytes, err := c.GetBytes(key)
	if err != nil {
		err = g.Error(err, "could not read from GET request body")
		return
	} else if respBytes == nil {
		return
	}

	err = g.Unmarshal(string(respBytes), objPtr)
	if err != nil {
		err = g.Error(err, "could not unmarshal")
	}

	return
}

// Remove delete a key
func (c *CacheClient) Remove(key string) (err error) {
	req, _ := http.NewRequest("DELETE", c.URL+key, nil)
	_, err = c.client.Do(req)
	if err != nil {
		err = g.Error(err, "could not perform DELETE request")
		return
	}
	return
}

// SetEx saves a key/value pair and expires after seconds
func (c *CacheClient) SetEx(key string, value interface{}, expire int) (err error) {

	err = c.Set(key, value)
	if err != nil {
		err = g.Error(err, "could not set value")
		return
	}

	time.AfterFunc(
		time.Until(time.Now().Add(time.Duration(expire)*time.Second)),
		func() { c.Remove(key) },
	)

	return
}

// Pop gets a key/value pair and unmarshals and deletes
func (c *CacheClient) Pop(key string, objPtr interface{}) (err error) {
	err = c.Get(key, objPtr)
	if err != nil {
		err = g.Error(err, "could not pop")
		return
	}
	err = c.Remove(key)
	if err != nil {
		err = g.Error(err, "could not remove")
	}
	return
}

// PopStr gets a key/value pair and unmarshals and deletes
func (c *CacheClient) PopStr(key string) (val string, err error) {
	val, err = c.GetStr(key)
	if err != nil {
		err = g.Error(err, "could not pop")
		return
	}
	err = c.Remove(key)
	if err != nil {
		err = g.Error(err, "could not remove")
	}
	return
}
