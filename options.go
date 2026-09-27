package tgbot

import (
	"net/http"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		if c != nil {
			cl.http = c
		}
	}
}

// WithBaseURL overrides the Telegram API base URL.
// The value should be the API root without the bot token.
// Example: https://api.telegram.org
func WithBaseURL(url string) Option {
	return func(cl *Client) {
		if url != "" {
			cl.baseURL = url
		}
	}
}

// WithRetryAfter makes the client wait and send a call once more when
// Telegram answers 429 "Too Many Requests" with a wait of at most maxWait. A
// longer wait, or a second 429, is returned to the caller as usual. Off by
// default. Keep maxWait short where an answer has a deadline (a pre-checkout
// query must be answered within 10 seconds).
func WithRetryAfter(maxWait time.Duration) Option {
	return func(cl *Client) {
		cl.maxRetryWait = maxWait
	}
}
