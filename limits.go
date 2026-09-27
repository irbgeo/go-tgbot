package tgbot

// Telegram's own limits and codes.
const (
	// MaxMessageLength is the longest text one message may hold, in
	// characters (runes). A longer text is refused; see SplitText.
	MaxMessageLength = 4096
	// MaxCallbackDataLength is the longest callback_data of one inline
	// button, in bytes.
	MaxCallbackDataLength = 64
	// CurrencyStars is the currency of a Telegram Stars invoice (SendInvoice
	// with an empty provider token).
	CurrencyStars = "XTR"
)

// SplitText cuts text into parts that each fit one message
// (MaxMessageLength). A cut goes after the last line break in the second
// half of a part, so lines stay whole where possible. Text that fits is
// returned as it is, as the only part.
func SplitText(text string) []string {
	runes := []rune(text)
	parts := make([]string, 0, len(runes)/MaxMessageLength+1)
	for len(runes) > MaxMessageLength {
		cut := MaxMessageLength
		if nl := lastIndexRune(runes[:cut], '\n'); nl > MaxMessageLength/2 {
			cut = nl + 1
		}
		parts = append(parts, string(runes[:cut]))
		runes = runes[cut:]
	}
	return append(parts, string(runes))
}

// lastIndexRune is strings.LastIndexByte for a rune slice; -1 if absent.
func lastIndexRune(runes []rune, target rune) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] == target {
			return i
		}
	}
	return -1
}
