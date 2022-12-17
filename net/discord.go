package net

import (
	"bytes"
	"net/http"

	"github.com/flarco/g"
)

// DiscordClient is the discord client
type DiscordClient struct {
	WebHookURL string
}

// https://gist.github.com/Birdie0/78ee79402a4301b1faf412ab5f1cdcf9
type DiscordMessage struct {
	Username  string                `json:"username"`
	AvatarURL string                `json:"avatar_url"`
	Content   string                `json:"content"`
	Embeds    []DiscordMessageEmbed `json:"embeds"`
}

type DiscordMessageEmbed struct {
	Author      map[string]any   `json:"author"`
	Title       string           `json:"title"`
	Url         string           `json:"url"`
	Description string           `json:"description"`
	Color       int              `json:"color"`
	Fields      []map[string]any `json:"fields"`
	Thumbnail   map[string]any   `json:"thumbnail"`
	Image       map[string]any   `json:"image"`
	Footer      map[string]any   `json:"footer"`
}

// NewDiscordClient creates a new discord client
func NewDiscordClient(url string) *DiscordClient {
	return &DiscordClient{WebHookURL: url}
}

// Send sends a slack message
func (sc *DiscordClient) Send(msg DiscordMessage) (err error) {

	msgBody, _ := json.Marshal(msg)
	resp, respBytes, err := ClientDo(
		http.MethodPost,
		sc.WebHookURL,
		bytes.NewBuffer(msgBody),
		map[string]string{"Content-Type": "application/json"},
		5,
	)

	if err != nil {
		return g.Error(err)
	} else if resp.StatusCode >= 400 {
		return g.Error(string(respBytes))
	}
	return nil
}
