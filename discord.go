package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type WebhookPayload struct {
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Title       string       `json:"title,omitempty"`
	Description string       `json:"description,omitempty"`
	Color       int          `json:"color,omitempty"`
	Fields      []EmbedField `json:"fields,omitempty"`
	Footer      *EmbedFooter `json:"footer,omitempty"`
	Author      *EmbedAuthor `json:"author,omitempty"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type EmbedFooter struct {
	Text    string `json:"text"`
	IconURL string `json:"icon_url,omitempty"`
}

type EmbedAuthor struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
	IconURL string `json:"icon_url,omitempty"`
}

var webhookWarningSent = false

func SendWebhook(url string, payload WebhookPayload) error {
	if !strings.HasPrefix(url, "https://discord.com/api/webhooks/") {
		if !webhookWarningSent {
			logger.Printf("Discord webhook URL is not set correctly, skipping webhook send.\n")
			webhookWarningSent = true
		}
		return nil
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	if logger != nil {
		if len(payload.Embeds) > 0 {
			logger.Printf("Sending discord webhook for: %s\n", payload.Embeds[0].Title)
		} else {
			logger.Printf("Sending discord webhook...\n")
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if logger != nil {
			logger.Printf("Error sending webhook: %v\n", err)
		}
		return err
	}
	defer resp.Body.Close()

	return nil
}
