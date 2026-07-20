package tgbot

import "context"

// AnswerCallback acknowledges a callback query with no popup — the common case
// (dismisses the button's loading spinner).
func (s *Client) AnswerCallback(ctx context.Context, callbackQueryID string) (bool, error) {
	return s.AnswerCallbackQuery(ctx, callbackQueryID, nil)
}
