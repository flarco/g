package net

import (
	"bytes"
	"net/http"

	"github.com/flarco/g"
)

const defaultSlackTimeout = 5

// SlackClient is the slack client
type SlackClient struct {
	WebHookURL string
	UserName   string
	Channel    string
}

// SlackMessage is a Slack message
// See https://api.slack.com/docs/messages/builder for basic
// See https://app.slack.com/block-kit-builder for advanced
type SlackMessage struct {
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Type        string            `json:"type,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment is a Slack Attachement
type SlackAttachment struct {
	Color         string `json:"color,omitempty"`
	Fallback      string `json:"fallback,omitempty"`
	CallbackID    string `json:"callback_id,omitempty"`
	ID            int    `json:"id,omitempty"`
	AuthorID      string `json:"author_id,omitempty"`
	AuthorName    string `json:"author_name,omitempty"`
	AuthorSubname string `json:"author_subname,omitempty"`
	AuthorLink    string `json:"author_link,omitempty"`
	AuthorIcon    string `json:"author_icon,omitempty"`
	Title         string `json:"title,omitempty"`
	TitleLink     string `json:"title_link,omitempty"`
	Pretext       string `json:"pretext,omitempty"`
	Text          string `json:"text,omitempty"`
	ImageURL      string `json:"image_url,omitempty"`
	ThumbURL      string `json:"thumb_url,omitempty"`
	// Fields and actions are not defined.
	MarkdownIn []string `json:"mrkdwn_in,omitempty"`
	Ts         int      `json:"ts,omitempty"`
}

// NewSlackClient creates a new slack client
func NewSlackClient(url string) *SlackClient {
	return &SlackClient{WebHookURL: url}
}

// Send sends a slack message
func (sc *SlackClient) Send(msg SlackMessage) (err error) {

	msgBody, _ := json.Marshal(msg)
	_, respBytes, err := ClientDo(
		http.MethodPost,
		sc.WebHookURL,
		bytes.NewBuffer(msgBody),
		map[string]string{"Content-Type": "application/json"},
		5,
	)

	if string(respBytes) != "ok" {
		return g.Error("Non-ok response returned from Slack")
	}
	return nil
}
