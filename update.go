package tgbot

import "strings"

// SenderID returns the Telegram user id behind the update (the sender of a
// message or the presser of an inline button), or 0 when unknown.
func (s Update) SenderID() int64 {
	switch {
	case s.CallbackQuery != nil:
		return s.CallbackQuery.SenderID()
	case s.Message != nil && s.Message.From != nil:
		return s.Message.From.ID
	}
	return 0
}

// ChatID returns the chat the update belongs to (the message chat, or the chat
// of the message an inline button is attached to), or 0 when unknown.
func (s Update) ChatID() int64 {
	if s.Message != nil {
		return s.Message.Chat.ID
	}
	return s.CallbackQuery.ChatID() // nil-safe
}

// SenderID returns the user id of the tutor who pressed the inline button, or 0
// when unknown.
func (s *CallbackQuery) SenderID() int64 {
	if s == nil {
		return 0
	}
	return s.From.ID
}

// ChatID returns the chat id of the message the button is attached to, or 0
// when the callback carries no message.
func (s *CallbackQuery) ChatID() int64 {
	if s == nil || s.Message == nil {
		return 0
	}
	return s.Message.Chat.ID
}

// MessageID returns the id of the message the button is attached to (used to
// edit it in place), or 0 when the callback carries no message.
func (s *CallbackQuery) MessageID() int64 {
	if s == nil || s.Message == nil {
		return 0
	}
	return s.Message.MessageID
}

// Command returns the bot command in a message without the leading slash (and
// without any "@botname" suffix or arguments), e.g. "/start@bot foo" -> "start".
// The bool is false when the message is not a command.
func (s Update) Command() (string, bool) {
	if s.Message == nil {
		return "", false
	}
	text := strings.TrimSpace(s.Message.Text)
	if !strings.HasPrefix(text, "/") {
		return "", false
	}
	cmd := text[1:]
	if i := strings.IndexAny(cmd, " @"); i >= 0 {
		cmd = cmd[:i]
	}
	if cmd == "" {
		return "", false
	}
	return cmd, true
}
