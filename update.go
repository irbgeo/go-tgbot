package tgbot

import "strings"

// SenderID returns the Telegram user id behind the update (the sender of a
// message or the presser of an inline button), or 0 when unknown.
func (u Update) SenderID() int64 {
	switch {
	case u.CallbackQuery != nil:
		return u.CallbackQuery.From.ID
	case u.Message != nil && u.Message.From != nil:
		return u.Message.From.ID
	}
	return 0
}

// ChatID returns the chat the update belongs to (the message chat, or the chat
// of the message an inline button is attached to), or 0 when unknown.
func (u Update) ChatID() int64 {
	switch {
	case u.Message != nil:
		return u.Message.Chat.ID
	case u.CallbackQuery != nil && u.CallbackQuery.Message != nil:
		return u.CallbackQuery.Message.Chat.ID
	}
	return 0
}

// Command returns the bot command in a message without the leading slash (and
// without any "@botname" suffix or arguments), e.g. "/start@bot foo" -> "start".
// The bool is false when the message is not a command.
func (u Update) Command() (string, bool) {
	if u.Message == nil {
		return "", false
	}
	text := strings.TrimSpace(u.Message.Text)
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
