package tgbot

// Button builds an inline keyboard button that fires a callback query with the
// given data.
func Button(text, data string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, CallbackData: data}
}

// URLButton builds an inline keyboard button that opens a URL.
func URLButton(text, url string) InlineKeyboardButton {
	return InlineKeyboardButton{Text: text, URL: url}
}

// Row groups buttons into a single keyboard row. It reads well inside Keyboard:
//
//	Keyboard(
//	    Row(Button("Yes", "y"), Button("No", "n")),
//	    Row(Button("Menu", "menu")),
//	)
func Row(buttons ...InlineKeyboardButton) []InlineKeyboardButton {
	return buttons
}

// Keyboard builds an inline keyboard markup from rows of buttons. It returns nil
// when no rows are given, so it can be passed straight to a "no keyboard" call.
func Keyboard(rows ...[]InlineKeyboardButton) *InlineKeyboardMarkup {
	if len(rows) == 0 {
		return nil
	}
	return &InlineKeyboardMarkup{InlineKeyboard: rows}
}
