package main

import (
	"time"
)

type Message struct {
	Username       string    `json:"username"`
	MessageContent string    `json:"message_content"`
	Timestamp      time.Time `json:"timestamp"`
	Type           string    `json:"type"` // "user" o "system"
}

func NewUserMessage(username, content string) *Message {
	return &Message{
		Username:       username,
		MessageContent: content,
		Timestamp:      time.Now(),
		Type:           "user",
	}
}

func NewSystemMessage(content string) *Message {
	return &Message{
		Username:       "Sistema",
		MessageContent: content,
		Timestamp:      time.Now(),
		Type:           "system",
	}
}