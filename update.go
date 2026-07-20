package tgbot

import "strings"

// SenderID returns the Telegram user id behind the update (the sender of a
// message or the presser of an inline button), or 0 when unknown.
func (s Update) SenderID() int64 {
	switch {
	case s.CallbackQuery != nil:
		return s.CallbackQuery.From.ID
	case s.Message != nil && s.Message.From != nil:
		return s.Message.From.ID
	}
	return 0
}

// ChatID returns the chat the update belongs to (the message chat, or the chat
// of the message an inline button is attached to), or 0 when unknown.
func (s Update) ChatID() int64 {
	switch {
	case s.Message != nil:
		return s.Message.Chat.ID
	case s.CallbackQuery != nil && s.CallbackQuery.Message != nil:
		return s.CallbackQuery.Message.Chat.ID
	}
	return 0
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
