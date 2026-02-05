package tgbot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const defaultBase = "https://api.telegram.org"

// Client wraps Telegram Bot API calls.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
}

// NewClient creates a new Telegram Bot API client.
func NewClient(token string, opts ...Option) (*Client, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, errors.New("telegram: token is required")
	}
	cl := &Client{
		token:   token,
		baseURL: defaultBase,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cl)
		}
	}
	return cl, nil
}

// GetMe returns the bot's user info.
func (c *Client) GetMe(ctx context.Context) (User, error) {
	return doJSON[User](ctx, c, "getMe", struct{}{})
}

// SendMessage sends a text message.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string, opts *SendMessageOptions) (Message, error) {
	payload := sendMessagePayload{ChatID: chatID, Text: text}
	if opts != nil {
		payload.ParseMode = opts.ParseMode
		payload.DisableWebPagePreview = opts.DisableWebPagePreview
		payload.DisableNotification = opts.DisableNotification
		payload.ReplyToMessageID = opts.ReplyToMessageID
		payload.AllowSendingWithoutReply = opts.AllowSendingWithoutReply
		payload.ReplyMarkup = opts.ReplyMarkup
	}
	return doJSON[Message](ctx, c, "sendMessage", payload)
}

// GetUpdates polls for incoming updates (long polling).
func (c *Client) GetUpdates(ctx context.Context, opts *GetUpdatesOptions) ([]Update, error) {
	payload := getUpdatesPayload{}
	if opts != nil {
		payload.Offset = opts.Offset
		payload.Limit = opts.Limit
		payload.Timeout = opts.Timeout
		payload.AllowedUpdates = opts.AllowedUpdates
	}
	return doJSON[[]Update](ctx, c, "getUpdates", payload)
}

// SetWebhook configures webhook URL.
func (c *Client) SetWebhook(ctx context.Context, url string) (bool, error) {
	payload := map[string]string{"url": url}
	return doJSON[bool](ctx, c, "setWebhook", payload)
}

// DeleteWebhook removes webhook and returns status.
func (c *Client) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) (bool, error) {
	payload := map[string]bool{"drop_pending_updates": dropPendingUpdates}
	return doJSON[bool](ctx, c, "deleteWebhook", payload)
}

// AnswerCallbackQuery answers a callback query from an inline keyboard.
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, opts *AnswerCallbackQueryOptions) (bool, error) {
	payload := answerCallbackQueryPayload{CallbackQueryID: callbackQueryID}
	if opts != nil {
		payload.Text = opts.Text
		payload.ShowAlert = opts.ShowAlert
	}
	return doJSON[bool](ctx, c, "answerCallbackQuery", payload)
}

// EditMessageText edits text of a message.
func (c *Client) EditMessageText(ctx context.Context, chatID int64, messageID int64, text string, opts *EditMessageTextOptions) (Message, error) {
	payload := editMessageTextPayload{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
	}
	if opts != nil {
		payload.ParseMode = opts.ParseMode
		payload.ReplyMarkup = opts.ReplyMarkup
	}
	return doJSON[Message](ctx, c, "editMessageText", payload)
}

// DeleteMessage deletes a message.
func (c *Client) DeleteMessage(ctx context.Context, chatID int64, messageID int64) (bool, error) {
	payload := map[string]int64{"chat_id": chatID, "message_id": messageID}
	return doJSON[bool](ctx, c, "deleteMessage", payload)
}

// SetMyCommands sets the bot's command list.
func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) (bool, error) {
	payload := map[string]any{"commands": commands}
	return doJSON[bool](ctx, c, "setMyCommands", payload)
}

// SendPhoto sends a photo. InputFile can be FileID/URL or a new upload.
func (c *Client) SendPhoto(ctx context.Context, chatID int64, photo InputFile, opts *SendPhotoOptions) (Message, error) {
	if photo.Reader != nil {
		fields := map[string]string{
			"chat_id": fmt.Sprintf("%d", chatID),
		}
		if opts != nil {
			if opts.Caption != "" {
				fields["caption"] = opts.Caption
			}
			if opts.ParseMode != "" {
				fields["parse_mode"] = opts.ParseMode
			}
		}
		files := []formFile{{
			Field:    "photo",
			Reader:   photo.Reader,
			Filename: photo.Filename,
		}}
		return doMultipart[Message](ctx, c, "sendPhoto", fields, files)
	}
	if photo.String() == "" {
		return Message{}, errors.New("telegram: photo file_id or URL is required")
	}
	payload := sendPhotoPayload{
		ChatID: chatID,
		Photo:  photo.String(),
	}
	if opts != nil {
		payload.Caption = opts.Caption
		payload.ParseMode = opts.ParseMode
	}
	return doJSON[Message](ctx, c, "sendPhoto", payload)
}

// SendDocument sends a document. InputFile can be FileID/URL or a new upload.
func (c *Client) SendDocument(ctx context.Context, chatID int64, doc InputFile, opts *SendDocumentOptions) (Message, error) {
	if doc.Reader != nil {
		fields := map[string]string{
			"chat_id": fmt.Sprintf("%d", chatID),
		}
		if opts != nil {
			if opts.Caption != "" {
				fields["caption"] = opts.Caption
			}
			if opts.ParseMode != "" {
				fields["parse_mode"] = opts.ParseMode
			}
		}
		files := []formFile{{
			Field:    "document",
			Reader:   doc.Reader,
			Filename: doc.Filename,
		}}
		return doMultipart[Message](ctx, c, "sendDocument", fields, files)
	}
	if doc.String() == "" {
		return Message{}, errors.New("telegram: document file_id or URL is required")
	}
	payload := sendDocumentPayload{
		ChatID:   chatID,
		Document: doc.String(),
	}
	if opts != nil {
		payload.Caption = opts.Caption
		payload.ParseMode = opts.ParseMode
	}
	return doJSON[Message](ctx, c, "sendDocument", payload)
}

func (c *Client) endpoint(method string) (string, error) {
	if c.baseURL == "" {
		return "", errors.New("telegram: baseURL is empty")
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("telegram: invalid baseURL: %w", err)
	}
	u.Path = path.Join(u.Path, "bot"+c.token, method)
	return u.String(), nil
}

type apiResponse[T any] struct {
	Ok          bool                `json:"ok"`
	Result      T                   `json:"result"`
	Description string              `json:"description"`
	ErrorCode   int                 `json:"error_code"`
	Parameters  *ResponseParameters `json:"parameters"`
}

// APIError represents a Telegram API error.
type APIError struct {
	Code        int
	Description string
	Parameters  *ResponseParameters
}

func (e *APIError) Error() string {
	if e == nil {
		return "telegram: API error"
	}
	if e.Code == 0 {
		return fmt.Sprintf("telegram: %s", e.Description)
	}
	return fmt.Sprintf("telegram: %d %s", e.Code, e.Description)
}

func doJSON[T any](ctx context.Context, c *Client, method string, payload any) (T, error) {
	var zero T
	endpoint, err := c.endpoint(method)
	if err != nil {
		return zero, err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return zero, fmt.Errorf("telegram: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return zero, fmt.Errorf("telegram: request failed: %w", err)
	}
	defer res.Body.Close()

	var apiRes apiResponse[T]
	if err := json.NewDecoder(res.Body).Decode(&apiRes); err != nil {
		return zero, fmt.Errorf("telegram: decode response: %w", err)
	}
	if !apiRes.Ok {
		return zero, &APIError{Code: apiRes.ErrorCode, Description: apiRes.Description, Parameters: apiRes.Parameters}
	}
	return apiRes.Result, nil
}

type formFile struct {
	Field    string
	Reader   io.Reader
	Filename string
}

func doMultipart[T any](ctx context.Context, c *Client, method string, fields map[string]string, files []formFile) (T, error) {
	var zero T
	endpoint, err := c.endpoint(method)
	if err != nil {
		return zero, err
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			return zero, fmt.Errorf("telegram: write field: %w", err)
		}
	}
	for _, f := range files {
		filename := f.Filename
		if filename == "" {
			filename = "upload"
		}
		part, err := writer.CreateFormFile(f.Field, filename)
		if err != nil {
			return zero, fmt.Errorf("telegram: create form file: %w", err)
		}
		if _, err := io.Copy(part, f.Reader); err != nil {
			return zero, fmt.Errorf("telegram: copy file: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return zero, fmt.Errorf("telegram: close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return zero, fmt.Errorf("telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := c.http.Do(req)
	if err != nil {
		return zero, fmt.Errorf("telegram: request failed: %w", err)
	}
	defer res.Body.Close()

	var apiRes apiResponse[T]
	if err := json.NewDecoder(res.Body).Decode(&apiRes); err != nil {
		return zero, fmt.Errorf("telegram: decode response: %w", err)
	}
	if !apiRes.Ok {
		return zero, &APIError{Code: apiRes.ErrorCode, Description: apiRes.Description, Parameters: apiRes.Parameters}
	}
	return apiRes.Result, nil
}
