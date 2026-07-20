package tgbot

import (
	"strings"
	"time"
)

// Telegram parse modes for message text.
const (
	ParseModeHTML       = "HTML"
	ParseModeMarkdownV2 = "MarkdownV2"
	ParseModeMarkdown   = "Markdown"
)

// IsForbidden reports whether the API rejected the call with HTTP 403 — in a
// private chat this means the user blocked the bot or their account is
// deactivated, i.e. they can no longer be reached.
func (s *APIError) IsForbidden() bool {
	return s != nil && s.Code == 403
}

// IsNotModified reports whether an editMessage* call was a no-op because the
// new content is identical to the current message. Safe to treat as success.
func (s *APIError) IsNotModified() bool {
	return s != nil && strings.Contains(strings.ToLower(s.Description), "not modified")
}

// RetryAfter returns the delay Telegram asks the caller to wait before
// retrying (HTTP 429), and whether such a hint was present.
func (s *APIError) RetryAfter() (time.Duration, bool) {
	if s != nil && s.Parameters != nil && s.Parameters.RetryAfter > 0 {
		return time.Duration(s.Parameters.RetryAfter) * time.Second, true
	}
	return 0, false
}
