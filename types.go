package telegram

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

func (f InputFile) String() string {
	if f.FileID != "" {
		return f.FileID
	}
	return f.URL
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
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

type sendMessagePayload struct {
	ChatID                   int64  `json:"chat_id"`
	Text                     string `json:"text"`
	ParseMode                string `json:"parse_mode,omitempty"`
	DisableWebPagePreview    bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification      bool   `json:"disable_notification,omitempty"`
	ReplyToMessageID         int64  `json:"reply_to_message_id,omitempty"`
	AllowSendingWithoutReply bool   `json:"allow_sending_without_reply,omitempty"`
}

// SendMessageOptions configures SendMessage.
type SendMessageOptions struct {
	ParseMode                string
	DisableWebPagePreview    bool
	DisableNotification      bool
	ReplyToMessageID         int64
	AllowSendingWithoutReply bool
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
