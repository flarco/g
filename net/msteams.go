package net

import (
	"bytes"
	"net/http"
	"time"
)

// MsTeamsClient is the MS Teams client
type MsTeamsClient struct {
	WebHookURL string
	TimeOut    time.Duration
}

// MsTeamsMessage is a MS Teams message
// according to https://docs.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/connectors-using
type MsTeamsMessage struct {
	Type            string                  `json:"@type,omitempty"`
	Context         string                  `json:"@context,omitempty"`
	ThemeColor      string                  `json:"themeColor,omitempty"`
	Summary         string                  `json:"summary,omitempty"`
	Sections        []MsTeamsMessageSection `json:"sections,omitempty"`
	PotentialAction []MsTeamsMessageAction  `json:"potentialAction,omitempty"`
}

// MsTeamsMessageSection is a MS Teams message section
type MsTeamsMessageSection struct {
	ActivityTitle    string               `json:"activityTitle,omitempty"`
	ActivitySubtitle string               `json:"activitySubtitle,omitempty"`
	ActivityImage    string               `json:"activityImage,omitempty"`
	Facts            []MsTeamsSectionFact `json:"facts,omitempty"`
	Markdown         bool                 `json:"markdown,omitempty"`
}

// MsTeamsSectionFact is a MS Teams message section fact
type MsTeamsSectionFact struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// MsTeamsMessagePotentialAction is a MS Teams message potential action
type MsTeamsMessagePotentialAction struct {
	Type    string                 `json:"@type,omitempty"`
	Name    string                 `json:"name,omitempty"`
	Inputs  []MsTeamsActionInput   `json:"inputs,omitempty"`
	Actions []MsTeamsMessageAction `json:"actions,omitempty"`
}

// MsTeamsActionInput is a MS Teams action input
type MsTeamsActionInput struct {
	Type          string                     `json:"@type,omitempty"`
	ID            string                     `json:"id,omitempty"`
	IsMultiline   bool                       `json:"isMultiline,omitempty"`
	IsMultiSelect bool                       `json:"isMultiSelect,omitempty"`
	Title         string                     `json:"title,omitempty"`
	Choices       []MsTeamsActionInputChoice `json:"choices,omitempty"`
}

// MsTeamsActionInputChoice is a MS Teams action input choice
type MsTeamsActionInputChoice struct {
	Display string `json:"display,omitempty"`
	Value   string `json:"value,omitempty"`
}

// MsTeamsMessageAction is a MS Teams message action
type MsTeamsMessageAction struct {
	Type   string `json:"@type,omitempty"`
	Name   string `json:"name,omitempty"`
	Target string `json:"target,omitempty"`
}

// Send sends an Ms Teams message
func (c *MsTeamsClient) Send(msg MsTeamsMessage) (err error) {
	msgBody, _ := json.Marshal(msg)
	_, _, err = ClientDo(
		http.MethodPost,
		c.WebHookURL,
		bytes.NewBuffer(msgBody),
		map[string]string{"Content-Type": "application/json"},
		5,
	)
	return
}

// NewMsTeamsCient creates a new ms teams client
func NewMsTeamsCient(url string) *MsTeamsClient {
	return &MsTeamsClient{WebHookURL: url}
}

var example = `
{
	"@type": "MessageCard",
	"@context": "http://schema.org/extensions",
	"themeColor": "0076D7",
	"summary": "Larry Bryant created a new task",
	"sections": [{
			"activityTitle": "![TestImage](https://47a92947.ngrok.io/Content/Images/default.png)Larry Bryant created a new task",
			"activitySubtitle": "On Project Tango",
			"activityImage": "https://teamsnodesample.azurewebsites.net/static/img/image5.png",
			"facts": [{
					"name": "Assigned to",
					"value": "Unassigned"
			}, {
					"name": "Due date",
					"value": "Mon May 01 2017 17:07:18 GMT-0700 (Pacific Daylight Time)"
			}, {
					"name": "Status",
					"value": "Not started"
			}],
			"markdown": true
	}],
	"potentialAction": [{
			"@type": "ActionCard",
			"name": "Add a comment",
			"inputs": [{
					"@type": "TextInput",
					"id": "comment",
					"isMultiline": false,
					"title": "Add a comment here for this task"
			}],
			"actions": [{
					"@type": "HttpPOST",
					"name": "Add comment",
					"target": "http://..."
			}]
	}, {
			"@type": "ActionCard",
			"name": "Set due date",
			"inputs": [{
					"@type": "DateInput",
					"id": "dueDate",
					"title": "Enter a due date for this task"
			}],
			"actions": [{
					"@type": "HttpPOST",
					"name": "Save",
					"target": "http://..."
			}]
	}, {
			"@type": "ActionCard",
			"name": "Change status",
			"inputs": [{
					"@type": "MultichoiceInput",
					"id": "list",
					"title": "Select a status",
					"isMultiSelect": "false",
					"choices": [{
							"display": "In Progress",
							"value": "1"
					}, {
							"display": "Active",
							"value": "2"
					}, {
							"display": "Closed",
							"value": "3"
					}]
			}],
			"actions": [{
					"@type": "HttpPOST",
					"name": "Save",
					"target": "http://..."
			}]
	}]
}
`
