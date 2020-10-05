package gutil

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cast"
)

type lokiLine struct {
	labels map[string]string
	text   string
}

type lokiBatch struct {
	labels map[string]string
	texts  []string
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

func lokiSendBatch(URL string, batch lokiBatch) {
	// ensures only 1 instance of lokiSendBatch is running
	lokiMux.Lock()
	defer lokiMux.Unlock()

	values := []lokiValue{}
	for _, text := range batch.texts {
		ts := cast.ToString(time.Now().UnixNano())
		values = append(values, []interface{}{ts, text})
	}
	e := lokiEvent{
		[]lokiStream{{Stream: batch.labels, Values: values}},
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

// GetLokiHook returns a log hook to a Grafana Loki instance
// to have dynamic label values, set a key in `labels` with a empty string value
func GetLokiHook(URL string, labels map[string]string) (hook *LogHook) {
	lokiChan := make(chan lokiLine, 100000)

	// add level as dynamic label
	labels["level"] = ""

	hookFunc := func(text string, args ...interface{}) {
		data := M()
		dataKeys := []string{}
		newArgs := []interface{}{}
		for _, val := range args {
			switch val.(type) {
			case map[string]interface{}:
				for k, v := range val.(map[string]interface{}) {
					data[k] = v
					dataKeys = append(dataKeys, k)
				}
			default:
				newArgs = append(newArgs, val)
			}
		}

		newLabels := map[string]string{}
		for k, v := range labels {
			if v != "" {
				newLabels[k] = v
			}
		}

		sort.Strings(dataKeys)
		text = F(text, newArgs...)
		for _, k := range dataKeys {
			v := data[k]
			vS := ""
			switch v.(type) {
			case *time.Time:
				t := v.(*time.Time)
				if t == nil {
					vS = ""
				}
			default:
				vS = cast.ToString(v)
			}

			if vS == "" {
				continue
			} else if _, ok := labels[k]; ok {
				newLabels[k] = vS
			} else {
				text = F("%s %s=%s", text, k, vS)
			}
		}
		lokiChan <- lokiLine{newLabels, text}
	}

	hook = NewLogHook(DebugLevel, hookFunc)
	hook.labels = labels
	hook.queue = lokiChan
	go func() {
		defer close(lokiChan)

		batches := map[string]lokiBatch{}
		for {
			select {
			case line := <-lokiChan:
				key := fmt.Sprint(line.labels)
				if b, ok := batches[key]; ok {
					b.texts = append(b.texts, line.text)
					batches[key] = b
				} else {
					batches[key] = lokiBatch{line.labels, []string{line.text}}
				}
			default:
				for _, b := range batches {
					// println(F("lokiSendBatch - %d - %#v", len(b.texts), b.labels))
					lokiSendBatch(URL, b)
				}
				batches = map[string]lokiBatch{}
			}
		}
	}()
	return
}
