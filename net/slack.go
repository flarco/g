package net

import (
	"bytes"
	"net/http"
	"time"

	"github.com/flarco/gutil"
)

const defaultSlackTimeout = 5 * time.Second

// SlackClient is the slack client
type SlackClient struct {
	WebHookURL string
	UserName   string
	Channel    string
	TimeOut    time.Duration
}

// SlackMessage is a Slack message
// See https://api.slack.com/docs/messages/builder for basic
// See https://app.slack.com/block-kit-builder for advanced
type SlackMessage struct {
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Channel     string            `json:"channel,omitempty"`
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
func (sc *SlackClient) Send(slackRequest SlackMessage) error {
	slackBody, _ := json.Marshal(slackRequest)
	req, err := http.NewRequest(http.MethodPost, sc.WebHookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return gutil.Error(err, "could not build request")
	}
	req.Header.Add("Content-Type", "application/json")
	if sc.TimeOut == 0 {
		sc.TimeOut = defaultSlackTimeout
	}
	client := &http.Client{Timeout: sc.TimeOut}
	resp, err := client.Do(req)
	if err != nil {
		return gutil.Error(err, "could not send request")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return gutil.Error(err, "could not read response body")
	}
	if buf.String() != "ok" {
		return gutil.Error("Non-ok response returned from Slack")
	}
	return nil
}
