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

// MsTeamsAdaptiveMessage is the envelope for sending an Adaptive Card to a MS Teams webhook.
// The top-level 'type' field should be "message".
// see https://learn.microsoft.com/en-us/connectors/teams/?tabs=text1%2Cdotnet#microsoft-teams-webhook
// see https://adaptivecards.io/samples/
// see https://adaptivecards.microsoft.com/
type MsTeamsAdaptiveMessage struct {
	Type        string                      `json:"type,omitempty"` // Typically "message"
	Attachments []MsTeamsAdaptiveAttachment `json:"attachments,omitempty"`
}

// MsTeamsAdaptiveAttachment represents an attachment in a Teams message,
// specifically for an Adaptive Card.
type MsTeamsAdaptiveAttachment struct {
	ContentType string              `json:"contentType"`          // Should be "application/vnd.microsoft.card.adaptive"
	ContentURL  *string             `json:"contentUrl,omitempty"` // URL to the card payload, mutually exclusive with Content
	Content     AdaptiveCardContent `json:"content"`              // The Adaptive Card itself
}

// AdaptiveCardContent represents the actual Adaptive Card.
// See https://adaptivecards.io/schemas/adaptive-card.json
type AdaptiveCardContent struct {
	Schema                   string                          `json:"$schema,omitempty"`      // e.g., "http://adaptivecards.io/schemas/adaptive-card.json"
	Type                     string                          `json:"type,omitempty"`         // Should be "AdaptiveCard"
	Version                  string                          `json:"version,omitempty"`      // e.g., "1.5", "1.6"
	Body                     []any                           `json:"body,omitempty"`         // Array of card elements (e.g., TextBlock, Image, Container)
	Actions                  []any                           `json:"actions,omitempty"`      // Array of actions (e.g., Action.OpenUrl, Action.Submit)
	Speak                    string                          `json:"speak,omitempty"`        // SSML for speech
	Lang                     string                          `json:"lang,omitempty"`         // Language of the card (e.g., "en-US")
	FallbackText             string                          `json:"fallbackText,omitempty"` // Text to display on clients that don't support Adaptive Cards
	BackgroundImage          *AdaptiveContentBackgroundImage `json:"backgroundImage,omitempty"`
	MinHeight                string                          `json:"minHeight,omitempty"`                // Minimum height of the card (e.g., "150px")
	SelectAction             any                             `json:"selectAction,omitempty"`             // Action to execute when card is selected
	Style                    string                          `json:"style,omitempty"`                    // "default", "emphasis" (style means nothing for the root card, but is valid)
	VerticalContentAlignment string                          `json:"verticalContentAlignment,omitempty"` // "top", "center", "bottom"
	Height                   string                          `json:"height,omitempty"`                   // "auto", "stretch"
	Rtl                      *bool                           `json:"rtl,omitempty"`                      // Right-to-left layout
}

// AdaptiveContentBackgroundImage defines properties for a background image in an Adaptive Card.
type AdaptiveContentBackgroundImage struct {
	URL                 string `json:"url"`
	FillMode            string `json:"fillMode,omitempty"`            // "cover", "repeatHorizontally", "repeatVertically", "repeat"
	HorizontalAlignment string `json:"horizontalAlignment,omitempty"` // "left", "center", "right"
	VerticalAlignment   string `json:"verticalAlignment,omitempty"`   // "top", "center", "bottom"
}

// AdaptiveContentTextBlock adaptive card element
type AdaptiveContentTextBlock struct {
	Type                string            `json:"type"` // Must be "TextBlock"
	Text                string            `json:"text"`
	Color               string            `json:"color,omitempty"`               // "default", "dark", "light", "accent", "good", "warning", "attention"
	FontType            string            `json:"fontType,omitempty"`            // "default", "monospace"
	HorizontalAlignment string            `json:"horizontalAlignment,omitempty"` // "left", "center", "right"
	IsSubtle            *bool             `json:"isSubtle,omitempty"`
	MaxLines            int               `json:"maxLines,omitempty"`
	Size                string            `json:"size,omitempty"`   // "default", "small", "medium", "large", "extraLarge"
	Weight              string            `json:"weight,omitempty"` // "default", "lighter", "bolder"
	Wrap                *bool             `json:"wrap,omitempty"`
	Style               string            `json:"style,omitempty"` // "default", "heading", "paragraph" (note: "paragraph" is default, "heading" from v1.5)
	ID                  string            `json:"id,omitempty"`
	IsVisible           *bool             `json:"isVisible,omitempty"` // Default: true
	Separator           *bool             `json:"separator,omitempty"`
	Spacing             string            `json:"spacing,omitempty"`  // "none", "small", "default", "medium", "large", "extraLarge", "padding"
	Fallback            any               `json:"fallback,omitempty"` // Can be "drop" or another element of the same type
	Requires            map[string]string `json:"requires,omitempty"` // Host version requirements
}

// AdaptiveContentImage adaptive card element
type AdaptiveContentImage struct {
	Type                string            `json:"type"` // Must be "Image"
	URL                 string            `json:"url"`
	AltText             string            `json:"altText,omitempty"`
	BackgroundColor     string            `json:"backgroundColor,omitempty"`     // New in 1.2
	Height              string            `json:"height,omitempty"`              // "auto", "stretch", or "<N>px"
	HorizontalAlignment string            `json:"horizontalAlignment,omitempty"` // "left", "center", "right"
	SelectAction        any               `json:"selectAction,omitempty"`        // Action
	Size                string            `json:"size,omitempty"`                // "auto", "stretch", "small", "medium", "large"
	Style               string            `json:"style,omitempty"`               // "default", "person" (for rounded images)
	Width               string            `json:"width,omitempty"`               // "<N>px" or "auto" or "stretch"
	ID                  string            `json:"id,omitempty"`
	IsVisible           *bool             `json:"isVisible,omitempty"`
	Separator           *bool             `json:"separator,omitempty"`
	Spacing             string            `json:"spacing,omitempty"`
	Fallback            any               `json:"fallback,omitempty"`
	Requires            map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentActionOpenURL defines an action to open a URL.
type AdaptiveContentActionOpenURL struct {
	Type      string            `json:"type"` // Must be "Action.OpenUrl"
	Title     string            `json:"title,omitempty"`
	URL       string            `json:"url"`
	IconURL   string            `json:"iconUrl,omitempty"`
	Style     string            `json:"style,omitempty"`     // "default", "positive", "destructive"
	Tooltip   string            `json:"tooltip,omitempty"`   // New in 1.2
	IsEnabled *bool             `json:"isEnabled,omitempty"` // New in 1.5, default true
	Mode      string            `json:"mode,omitempty"`      // "primary", "secondary" (New in 1.5 for Universal Actions)
	ID        string            `json:"id,omitempty"`
	Fallback  any               `json:"fallback,omitempty"` // Can be "drop" or another action
	Requires  map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentActionSubmit defines a submit action.
// The data from associated input fields will be submitted.
type AdaptiveContentActionSubmit struct {
	Type             string            `json:"type"` // Must be "Action.Submit"
	Title            string            `json:"title,omitempty"`
	Data             any               `json:"data,omitempty"` // object or string, static data to merge with input data
	IconURL          string            `json:"iconUrl,omitempty"`
	Style            string            `json:"style,omitempty"`
	Tooltip          string            `json:"tooltip,omitempty"`
	IsEnabled        *bool             `json:"isEnabled,omitempty"`
	Mode             string            `json:"mode,omitempty"`
	AssociatedInputs string            `json:"associatedInputs,omitempty"` // "Auto", "None" or comma separated Input IDs (New in 1.3)
	ID               string            `json:"id,omitempty"`
	Fallback         any               `json:"fallback,omitempty"`
	Requires         map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentCodeBlock adaptive card element (New in 1.6)
type AdaptiveContentCodeBlock struct {
	Type     string `json:"type"` // Must be "CodeBlock"
	Code     string `json:"code"`
	Language string `json:"language,omitempty"` // Language hint for syntax highlighting (e.g., "csharp", "javascript", "json")
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentColumnSet adaptive card element
type AdaptiveContentColumnSet struct {
	Type                string                  `json:"type"` // Must be "ColumnSet"
	Columns             []AdaptiveContentColumn `json:"columns,omitempty"`
	SelectAction        any                     `json:"selectAction,omitempty"`
	Style               string                  `json:"style,omitempty"` // "default", "emphasis"
	Bleed               *bool                   `json:"bleed,omitempty"`
	MinHeight           string                  `json:"minHeight,omitempty"`
	HorizontalAlignment string                  `json:"horizontalAlignment,omitempty"` // "left", "center", "right"
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentColumn for use within a ColumnSet
type AdaptiveContentColumn struct {
	Type                     string                          `json:"type,omitempty"` // Should be "Column"
	Items                    []any                           `json:"items,omitempty"`
	BackgroundImage          *AdaptiveContentBackgroundImage `json:"backgroundImage,omitempty"`
	Bleed                    *bool                           `json:"bleed,omitempty"`
	MinHeight                string                          `json:"minHeight,omitempty"`
	Rtl                      *bool                           `json:"rtl,omitempty"`
	Separator                *bool                           `json:"separator,omitempty"`
	Spacing                  string                          `json:"spacing,omitempty"` // "none", "small", "default", "medium", "large", "extraLarge"
	SelectAction             any                             `json:"selectAction,omitempty"`
	Style                    string                          `json:"style,omitempty"`                    // "default", "emphasis", "good", "attention", "warning", "accent"
	VerticalContentAlignment string                          `json:"verticalContentAlignment,omitempty"` // "top", "center", "bottom"
	Width                    string                          `json:"width,omitempty"`                    // "auto", "stretch", number (weight), "<N>px"
	ID                       string                          `json:"id,omitempty"`
	IsVisible                *bool                           `json:"isVisible,omitempty"`
	Fallback                 any                             `json:"fallback,omitempty"`
	Requires                 map[string]string               `json:"requires,omitempty"`
}

// AdaptiveContentContainer adaptive card element
type AdaptiveContentContainer struct {
	Type                     string                          `json:"type"` // Must be "Container"
	Items                    []any                           `json:"items,omitempty"`
	SelectAction             any                             `json:"selectAction,omitempty"`
	Style                    string                          `json:"style,omitempty"`                    // "default", "emphasis", "good", "attention", "warning", "accent"
	VerticalContentAlignment string                          `json:"verticalContentAlignment,omitempty"` // "top", "center", "bottom"
	BackgroundImage          *AdaptiveContentBackgroundImage `json:"backgroundImage,omitempty"`
	Bleed                    *bool                           `json:"bleed,omitempty"`
	MinHeight                string                          `json:"minHeight,omitempty"`
	Rtl                      *bool                           `json:"rtl,omitempty"`
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentFactSet adaptive card element
type AdaptiveContentFactSet struct {
	Type  string                `json:"type"` // Must be "FactSet"
	Facts []AdaptiveContentFact `json:"facts"`
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
	Height    string            `json:"height,omitempty"` // "auto", "stretch"
}

// AdaptiveContentFact for use within a FactSet
type AdaptiveContentFact struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Speak string `json:"speak,omitempty"` // SSML for speech
}

// AdaptiveContentIcon adaptive card element (New in 1.6)
type AdaptiveContentIcon struct {
	Type                string `json:"type"`                          // Must be "Icon"
	Name                string `json:"name"`                          // Name of the icon (e.g., "Default", "Home", "User")
	Svg                 string `json:"svg,omitempty"`                 // SVG path data, mutually exclusive with Name
	Size                string `json:"size,omitempty"`                // "Small", "Default", "Medium", "Large", or "<N>px" (Default is "Default")
	ForegroundColor     string `json:"foregroundColor,omitempty"`     // Color of the icon (e.g., "Default", "Accent")
	BackgroundColor     string `json:"backgroundColor,omitempty"`     // Background color of the icon (e.g., "Default", "Accent")
	HorizontalAlignment string `json:"horizontalAlignment,omitempty"` // "Left", "Center", "Right" (Default is "Left")
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentRichTextBlock adaptive card element
type AdaptiveContentRichTextBlock struct {
	Type                string `json:"type"`                          // Must be "RichTextBlock"
	Inlines             []any  `json:"inlines"`                       // Array of TextRun and potentially other inline elements in future
	HorizontalAlignment string `json:"horizontalAlignment,omitempty"` // "left", "center", "right"
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
}

// AdaptiveContentTextRun for use within a RichTextBlock
type AdaptiveContentTextRun struct {
	Type          string `json:"type"` // Must be "TextRun"
	Text          string `json:"text"`
	Color         string `json:"color,omitempty"`
	FontType      string `json:"fontType,omitempty"`
	Highlight     *bool  `json:"highlight,omitempty"`
	IsSubtle      *bool  `json:"isSubtle,omitempty"`
	Italic        *bool  `json:"italic,omitempty"`
	SelectAction  any    `json:"selectAction,omitempty"`
	Size          string `json:"size,omitempty"`
	Strikethrough *bool  `json:"strikethrough,omitempty"`
	Underline     *bool  `json:"underline,omitempty"`
	Weight        string `json:"weight,omitempty"`
	Superscript   *bool  `json:"superscript,omitempty"` // New in 1.6
	Subscript     *bool  `json:"subscript,omitempty"`   // New in 1.6
}

// AdaptiveContentTable adaptive card element (New in 1.5)
type AdaptiveContentTable struct {
	Type                           string                       `json:"type"` // Must be "Table"
	Columns                        []AdaptiveContentTableColumn `json:"columns,omitempty"`
	Rows                           []AdaptiveContentTableRow    `json:"rows,omitempty"`
	FirstRowAsHeaders              *bool                        `json:"firstRowAsHeaders,omitempty"`              // Default true
	ShowGridLines                  *bool                        `json:"showGridLines,omitempty"`                  // Default true
	GridStyle                      string                       `json:"gridStyle,omitempty"`                      // "default", "accent"
	HorizontalCellContentAlignment string                       `json:"horizontalCellContentAlignment,omitempty"` // "left", "center", "right"
	VerticalCellContentAlignment   string                       `json:"verticalCellContentAlignment,omitempty"`   // "top", "center", "bottom"
	// Common properties
	ID        string            `json:"id,omitempty"`
	IsVisible *bool             `json:"isVisible,omitempty"`
	Separator *bool             `json:"separator,omitempty"`
	Spacing   string            `json:"spacing,omitempty"`
	Fallback  any               `json:"fallback,omitempty"`
	Requires  map[string]string `json:"requires,omitempty"`
	Height    string            `json:"height,omitempty"` // "auto", "stretch"
}

// AdaptiveContentTableColumn for use within a Table
type AdaptiveContentTableColumn struct {
	Width                          string `json:"width,omitempty"`                          // number (weight) or "<N>px"
	HorizontalCellContentAlignment string `json:"horizontalCellContentAlignment,omitempty"` // "left", "center", "right"
	VerticalCellContentAlignment   string `json:"verticalCellContentAlignment,omitempty"`   // "top", "center", "bottom"
}

// AdaptiveContentTableRow for use within a Table
type AdaptiveContentTableRow struct {
	Cells []AdaptiveContentTableCell `json:"cells,omitempty"`
	Style string                     `json:"style,omitempty"` // "default", "accent", "good", "attention", "warning"
}

// AdaptiveContentTableCell for use within a TableRow
type AdaptiveContentTableCell struct {
	Items                    []any                           `json:"items"` // Adaptive card elements
	SelectAction             any                             `json:"selectAction,omitempty"`
	Style                    string                          `json:"style,omitempty"`                    // "default", "accent", "good", "attention", "warning"
	VerticalContentAlignment string                          `json:"verticalContentAlignment,omitempty"` // "top", "center", "bottom"
	Bleed                    *bool                           `json:"bleed,omitempty"`
	Rtl                      *bool                           `json:"rtl,omitempty"`
	BackgroundImage          *AdaptiveContentBackgroundImage `json:"backgroundImage,omitempty"`
	MinHeight                string                          `json:"minHeight,omitempty"`
	Height                   string                          `json:"height,omitempty"` // "auto", "stretch"
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
	Text             string               `json:"text,omitempty"`
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

// SendMessage sends an Ms Teams message
func (c *MsTeamsClient) SendMessage(msg MsTeamsMessage) (err error) {
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

// SendAdaptiveMessage sends an Ms Teams Adaptive Card message
func (c *MsTeamsClient) SendAdaptiveMessage(msg MsTeamsAdaptiveMessage) (err error) {
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

// NewMsTeamsClient creates a new ms teams client
func NewMsTeamsClient(url string) *MsTeamsClient {
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
