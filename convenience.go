package tgbot

import "context"

// AnswerCallback acknowledges a callback query with no popup — the common case
// (dismisses the button's loading spinner).
func (c *Client) AnswerCallback(ctx context.Context, callbackQueryID string) (bool, error) {
	return c.AnswerCallbackQuery(ctx, callbackQueryID, nil)
}
