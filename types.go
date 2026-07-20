package tgbot

import "io"

// ResponseParameters holds extra Telegram API error info.
type ResponseParameters struct {
	RetryAfter      int   `json:"retry_after"`
	MigrateToChatID int64 `json:"migrate_to_chat_id"`
}

// InputFile represents a file to send (file_id, URL, or upload).
type InputFile struct {
	FileID   string
	URL      string
	Reader   io.Reader
	Filename string
}

func (s InputFile) String() string {
	if s.FileID != "" {
		return s.FileID
	}
	return s.URL
}

// User represents a Telegram user or bot.
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Language  string `json:"language_code"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int64  `json:"message_id"`
	Date      int64  `json:"date"`
	Chat      Chat   `json:"chat"`
	From      *User  `json:"from"`
	Text      string `json:"text"`
	Caption   string `json:"caption"`
}

// Update represents an incoming update.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// CallbackQuery represents a callback from an inline keyboard button.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// InlineKeyboardMarkup represents an inline keyboard.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// ReplyKeyboardMarkup represents a custom keyboard.
type ReplyKeyboardMarkup struct {
	Keyboard        [][]KeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool               `json:"resize_keyboard,omitempty"`
	OneTimeKeyboard bool               `json:"one_time_keyboard,omitempty"`
}

// KeyboardButton represents a button in a reply keyboard.
type KeyboardButton struct {
	Text string `json:"text"`
}

// ReplyKeyboardRemove removes the custom keyboard.
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

// BotCommand represents a bot command.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type sendMessagePayload struct {
	ChatID                   int64  `json:"chat_id"`
	Text                     string `json:"text"`
	ParseMode                string `json:"parse_mode,omitempty"`
	DisableWebPagePreview    bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification      bool   `json:"disable_notification,omitempty"`
	ReplyToMessageID         int64  `json:"reply_to_message_id,omitempty"`
	AllowSendingWithoutReply bool   `json:"allow_sending_without_reply,omitempty"`
	ReplyMarkup              any    `json:"reply_markup,omitempty"`
}

// SendMessageOptions configures SendMessage.
type SendMessageOptions struct {
	ParseMode                string
	DisableWebPagePreview    bool
	DisableNotification      bool
	ReplyToMessageID         int64
	AllowSendingWithoutReply bool
	ReplyMarkup              any // InlineKeyboardMarkup, ReplyKeyboardMarkup, or ReplyKeyboardRemove
}

type getUpdatesPayload struct {
	Offset         int64    `json:"offset,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

// GetUpdatesOptions configures GetUpdates.
type GetUpdatesOptions struct {
	Offset         int64
	Limit          int
	Timeout        int
	AllowedUpdates []string
}

type sendPhotoPayload struct {
	ChatID    int64  `json:"chat_id"`
	Photo     string `json:"photo"`
	Caption   string `json:"caption,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// SendPhotoOptions configures SendPhoto.
type SendPhotoOptions struct {
	Caption   string
	ParseMode string
}

type sendDocumentPayload struct {
	ChatID    int64  `json:"chat_id"`
	Document  string `json:"document"`
	Caption   string `json:"caption,omitempty"`
	ParseMode string `json:"parse_mode,omitempty"`
}

// SendDocumentOptions configures SendDocument.
type SendDocumentOptions struct {
	Caption   string
	ParseMode string
}

type answerCallbackQueryPayload struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

// AnswerCallbackQueryOptions configures AnswerCallbackQuery.
type AnswerCallbackQueryOptions struct {
	Text      string
	ShowAlert bool
}

type editMessageTextPayload struct {
	ChatID      int64  `json:"chat_id"`
	MessageID   int64  `json:"message_id"`
	Text        string `json:"text"`
	ParseMode   string `json:"parse_mode,omitempty"`
	ReplyMarkup any    `json:"reply_markup,omitempty"`
}

// EditMessageTextOptions configures EditMessageText.
type EditMessageTextOptions struct {
	ParseMode   string
	ReplyMarkup *InlineKeyboardMarkup
}
