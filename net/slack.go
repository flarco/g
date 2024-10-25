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
	Color       string            `json:"color,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
}

// SlackBlock represents a Slack block
type SlackBlock struct {
	Type     string        `json:"type"`
	Text     *SlackText    `json:"text,omitempty"`
	Fields   []SlackField  `json:"fields,omitempty"`
	Elements []interface{} `json:"elements,omitempty"`
}

// SlackText represents text within a Slack block
type SlackText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// SlackField represents a field in a Slack section
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackSection is a helper struct for creating section blocks
type SlackSection struct {
	Text   string
	Fields []*SlackField
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

// AddSection adds a new section block to the message
func (m *SlackMessage) AddSection(section *SlackSection) {
	block := SlackBlock{
		Type: "section",
	}

	if section.Text != "" {
		block.Text = &SlackText{
			Type: "mrkdwn",
			Text: section.Text,
		}
	}

	if len(section.Fields) > 0 {
		fields := make([]SlackField, len(section.Fields))
		for i, f := range section.Fields {
			fields[i] = SlackField{
				Title: f.Title,
				Value: f.Value,
				Short: f.Short,
			}
		}
		block.Fields = fields
	}

	m.Blocks = append(m.Blocks, block)
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
