package tgbot

import (
	"context"
	"errors"
	"time"
)

// PollOptions configures Client.Poll.
type PollOptions struct {
	// Timeout is the long-poll timeout in seconds. Keep it below the HTTP
	// client timeout (default 15s). Defaults to 10.
	Timeout int
	// Limit caps the number of updates returned per call (0 = server default).
	Limit int
	// AllowedUpdates restricts the update types delivered (nil = all but
	// chat_member).
	AllowedUpdates []string
	// Backoff is the pause after a transient getUpdates failure. Defaults to 3s.
	Backoff time.Duration
	// OnError, if set, is called for a non-fatal error — either a getUpdates
	// failure or an error returned by the handler. The loop keeps running.
	OnError func(error)
}

// Poll long-polls for updates and calls handle for each one, tracking the
// update offset itself. It honors Telegram's RetryAfter (429), backs off on
// transient errors, and returns nil when ctx is cancelled. A handler error
// does not stop the loop (it is reported via OnError).
func (s *Client) Poll(ctx context.Context, o PollOptions, handle func(context.Context, Update) error) error {
	timeout := o.Timeout
	if timeout == 0 {
		timeout = 10
	}
	backoff := o.Backoff
	if backoff == 0 {
		backoff = 3 * time.Second
	}

	var offset int64
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		updates, err := s.GetUpdates(ctx, &GetUpdatesOptions{
			Offset:         offset,
			Limit:          o.Limit,
			Timeout:        timeout,
			AllowedUpdates: o.AllowedUpdates,
		})
		if err != nil {
			if ctx.Err() != nil {
				continue // shutting down; the loop top returns nil
			}
			var apiErr *APIError
			if errors.As(err, &apiErr) {
				if d, ok := apiErr.RetryAfter(); ok {
					sleep(ctx, d)
					continue
				}
			}
			if o.OnError != nil {
				o.OnError(err)
			}
			sleep(ctx, backoff)
			continue
		}
		for _, upd := range updates {
			offset = upd.UpdateID + 1
			if err := handle(ctx, upd); err != nil && o.OnError != nil {
				o.OnError(err)
			}
		}
	}
}

// sleep waits for d or until ctx is cancelled, whichever comes first.
func sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
