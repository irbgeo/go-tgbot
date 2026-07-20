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
func (e *APIError) IsForbidden() bool {
	return e != nil && e.Code == 403
}

// IsNotModified reports whether an editMessage* call was a no-op because the
// new content is identical to the current message. Safe to treat as success.
func (e *APIError) IsNotModified() bool {
	return e != nil && strings.Contains(strings.ToLower(e.Description), "not modified")
}

// RetryAfter returns the delay Telegram asks the caller to wait before
// retrying (HTTP 429), and whether such a hint was present.
func (e *APIError) RetryAfter() (time.Duration, bool) {
	if e != nil && e.Parameters != nil && e.Parameters.RetryAfter > 0 {
		return time.Duration(e.Parameters.RetryAfter) * time.Second, true
	}
	return 0, false
}
