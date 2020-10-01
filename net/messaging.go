package net

// Message is a basic protocol for communication
type Message struct {
	ReqID     string                 `json:"req_id"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error"`
	OrigReqID string                 `json:"orig_req_id,omitempty"`
}
