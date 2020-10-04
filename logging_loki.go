package gutil

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
)

type LokiLine struct {
	labels map[string]string
	text   string
}

var (
	lokiClient = http.Client{Timeout: 2 * time.Second}
	lokiMux    sync.Mutex
)

type lokiValue []interface{}

type lokiStream struct {
	Stream map[string]string `json:"stream"` // labels
	Values []lokiValue       `json:"values"` // log lines
}

type lokiEvent struct {
	Streams []lokiStream `json:"streams"`
}

func lokiSendBatch(URL string, batch []LokiLine) {
	// ensures only 1 instance of lokiSendBatch is running
	lokiMux.Lock()
	defer lokiMux.Unlock()

	processLine := func(line LokiLine) {
		ts := cast.ToString(time.Now().UnixNano())
		value := []interface{}{ts, line.text}
		e := lokiEvent{
			[]lokiStream{{Stream: line.labels, Values: []lokiValue{value}}},
		}
		str := Marshal(e)
		body := strings.NewReader(str)
		req, err := http.NewRequest("POST", URL, body)
		if err != nil {
			if IsDebugLow() {
				println(err.Error() + ". could not POST @ " + URL)
			}
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := lokiClient.Do(req)
		if err != nil {
			if IsDebugLow() {
				println(err.Error() + ". could not perform request")
			}
			return
		}

		if resp.StatusCode >= 300 || resp.StatusCode < 200 {
			respBytes, _ := ioutil.ReadAll(resp.Body)
			if IsDebugLow() {
				println(fmt.Errorf("Unexpected Response %d: %s : %s => %s", resp.StatusCode, resp.Status, string(respBytes), str).Error())
			}
			return
		}
	}

	for _, line := range batch {
		processLine(line)
	}
}
