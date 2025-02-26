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
// See https://api.slack.com/reference/messaging/payload for current documentation
type SlackMessage struct {
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	ThreadTS    string            `json:"thread_ts,omitempty"`
	Text        string            `json:"text,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
	Blocks      []SlackBlock      `json:"blocks,omitempty"`
	UnfurlLinks bool              `json:"unfurl_links,omitempty"`
	UnfurlMedia bool              `json:"unfurl_media,omitempty"`
	Mrkdwn      bool              `json:"mrkdwn,omitempty"`
}

// SlackBlock represents a Slack block
// See https://api.slack.com/reference/block-kit/blocks
type SlackBlock struct {
	Type      string        `json:"type"`
	BlockID   string        `json:"block_id,omitempty"`
	Text      *SlackText    `json:"text,omitempty"`
	Fields    []SlackText   `json:"fields,omitempty"`
	Elements  []interface{} `json:"elements,omitempty"`
	Accessory interface{}   `json:"accessory,omitempty"`
	Title     *SlackText    `json:"title,omitempty"`
	ImageURL  string        `json:"image_url,omitempty"`
	AltText   string        `json:"alt_text,omitempty"`
}

// SlackText represents text within a Slack block
type SlackText struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Emoji    bool   `json:"emoji,omitempty"`
	Verbatim bool   `json:"verbatim,omitempty"`
}

// SlackField represents a field in a Slack section
type SlackField struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// SlackSection is a helper struct for creating section blocks
type SlackSection struct {
	Text   string
	Fields []*SlackField
}

// SlackAttachment is a Slack Attachment
// See https://api.slack.com/reference/messaging/attachments
type SlackAttachment struct {
	Color         string        `json:"color,omitempty"`
	Fallback      string        `json:"fallback,omitempty"`
	CallbackID    string        `json:"callback_id,omitempty"`
	ID            int           `json:"id,omitempty"`
	AuthorID      string        `json:"author_id,omitempty"`
	AuthorName    string        `json:"author_name,omitempty"`
	AuthorSubname string        `json:"author_subname,omitempty"`
	AuthorLink    string        `json:"author_link,omitempty"`
	AuthorIcon    string        `json:"author_icon,omitempty"`
	Title         string        `json:"title,omitempty"`
	TitleLink     string        `json:"title_link,omitempty"`
	Pretext       string        `json:"pretext,omitempty"`
	Text          string        `json:"text,omitempty"`
	ImageURL      string        `json:"image_url,omitempty"`
	ThumbURL      string        `json:"thumb_url,omitempty"`
	Footer        string        `json:"footer,omitempty"`
	FooterIcon    string        `json:"footer_icon,omitempty"`
	Fields        []SlackField  `json:"fields,omitempty"`
	Actions       []SlackAction `json:"actions,omitempty"`
	Blocks        []SlackBlock  `json:"blocks,omitempty"`
	MarkdownIn    []string      `json:"mrkdwn_in,omitempty"`
	Ts            string        `json:"ts,omitempty"`
}

// SlackAction represents an interactive action in a Slack attachment
type SlackAction struct {
	Name  string `json:"name"`
	Text  string `json:"text"`
	Type  string `json:"type"`
	Value string `json:"value,omitempty"`
	URL   string `json:"url,omitempty"`
	Style string `json:"style,omitempty"` // primary, danger
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
		fields := make([]SlackText, len(section.Fields))
		for i, f := range section.Fields {
			fields[i] = SlackText{
				Type: "mrkdwn",
				Text: f.Text,
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
