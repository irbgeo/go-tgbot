package tgbot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestKeyboard(t *testing.T) {
	require.Nil(t, InlineKeyboard(), "Keyboard() with no rows should be nil")

	kb := InlineKeyboard(
		Row(Button("Yes", "y"), Button("No", "n")),
		Row(URLButton("Open", "https://example.com")),
	)
	require.NotNil(t, kb)
	require.Len(t, kb.InlineKeyboard, 2)
	require.Equal(t, "Yes", kb.InlineKeyboard[0][0].Text)
	require.Equal(t, "y", kb.InlineKeyboard[0][0].CallbackData)
	require.Equal(t, "https://example.com", kb.InlineKeyboard[1][0].URL)
}

func TestReplyKeyboard(t *testing.T) {
	require.Nil(t, ReplyKeyboard(), "ReplyKeyboard() with no rows should be nil")

	kb := ReplyKeyboard(
		ReplyRow(TextButton("Yes"), TextButton("No")),
		ReplyRow(TextButton("Menu")),
	)
	require.NotNil(t, kb)
	require.Len(t, kb.Keyboard, 2)
	require.Equal(t, "Yes", kb.Keyboard[0][0].Text)
	require.Equal(t, "No", kb.Keyboard[0][1].Text)
	require.Equal(t, "Menu", kb.Keyboard[1][0].Text)
}

func TestUpdateAccessors(t *testing.T) {
	msg := Update{Message: &Message{Chat: Chat{ID: 5}, From: &User{ID: 7}, Text: "hi"}}
	require.Equal(t, int64(7), msg.SenderID())
	require.Equal(t, int64(5), msg.ChatID())

	cb := Update{CallbackQuery: &CallbackQuery{From: User{ID: 9}, Message: &Message{Chat: Chat{ID: 5}}, Data: "x"}}
	require.Equal(t, int64(9), cb.SenderID())
	require.Equal(t, int64(5), cb.ChatID())

	require.Zero(t, (Update{}).SenderID(), "empty update should be 0")
	require.Zero(t, (Update{}).ChatID(), "empty update should be 0")
}

func TestCallbackQueryAccessors(t *testing.T) {
	cb := &CallbackQuery{From: User{ID: 9}, Message: &Message{MessageID: 42, Chat: Chat{ID: 5}}, Data: "x"}
	require.Equal(t, int64(9), cb.SenderID())
	require.Equal(t, int64(5), cb.ChatID())
	require.Equal(t, int64(42), cb.MessageID())

	// No attached message: sender is still known, chat/message are 0.
	noMsg := &CallbackQuery{From: User{ID: 9}}
	require.Equal(t, int64(9), noMsg.SenderID())
	require.Zero(t, noMsg.ChatID())
	require.Zero(t, noMsg.MessageID())

	// Nil-safe.
	var nilCb *CallbackQuery
	require.Zero(t, nilCb.SenderID())
	require.Zero(t, nilCb.ChatID())
	require.Zero(t, nilCb.MessageID())
}

func TestCommand(t *testing.T) {
	tests := []struct {
		text  string
		want  string
		isCmd bool
	}{
		{"/start", "start", true},
		{"/start@my_bot", "start", true},
		{"/menu arg1 arg2", "menu", true},
		{"hello", "", false},
		{"  /help  ", "help", true},
		{"/", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := Update{Message: &Message{Text: tt.text}}.Command()
		require.Equal(t, tt.want, got, "Command(%q) text", tt.text)
		require.Equal(t, tt.isCmd, ok, "Command(%q) isCmd", tt.text)
	}
	_, ok := (Update{}).Command()
	require.False(t, ok, "non-message update should not be a command")
}

func TestAPIErrorHelpers(t *testing.T) {
	require.True(t, (&APIError{Code: 403, Description: "Forbidden: bot was blocked by the user"}).IsForbidden(), "403 should be forbidden")
	require.False(t, (&APIError{Code: 400}).IsForbidden(), "400 should not be forbidden")
	require.True(t, (&APIError{Code: 400, Description: "Bad Request: message is not modified"}).IsNotModified(), "should detect not-modified")

	d, ok := (&APIError{Code: 429, Parameters: &ResponseParameters{RetryAfter: 5}}).RetryAfter()
	require.True(t, ok)
	require.Equal(t, 5*time.Second, d)

	_, ok = (&APIError{Code: 500}).RetryAfter()
	require.False(t, ok, "no retry hint should be false")

	// Nil-safe.
	var e *APIError
	require.False(t, e.IsForbidden(), "nil APIError helpers should be false")
	require.False(t, e.IsNotModified(), "nil APIError helpers should be false")
}
