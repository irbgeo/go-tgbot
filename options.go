package telegram

import "net/http"

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
