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

// maxDownloadSize is Telegram's own documented Bot API ceiling for files a
// bot can download. DownloadFile enforces this on the actual response body,
// independent of any file_size the caller was told beforehand — Telegram
// marks Document.file_size optional, so it cannot be trusted as a hard cap.
const maxDownloadSize = 20 * 1024 * 1024

// Client wraps Telegram Bot API calls.
type Client struct {
	token   string
	baseURL string
	http    *http.Client
	// maxRetryWait is the longest 429 wait the client sits out itself
	// (WithRetryAfter); 0 turns that off.
	maxRetryWait time.Duration
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
func (s *Client) GetMe(ctx context.Context) (User, error) {
	return doJSON[User](ctx, s, "getMe", struct{}{})
}

// SendMessage sends a text message.
func (s *Client) SendMessage(ctx context.Context, chatID int64, text string, opts *SendMessageOptions) (Message, error) {
	payload := sendMessagePayload{ChatID: chatID, Text: text}
	if opts != nil {
		payload.ParseMode = opts.ParseMode
		payload.DisableWebPagePreview = opts.DisableWebPagePreview
		payload.DisableNotification = opts.DisableNotification
		payload.ReplyToMessageID = opts.ReplyToMessageID
		payload.AllowSendingWithoutReply = opts.AllowSendingWithoutReply
		payload.ReplyMarkup = opts.ReplyMarkup
	}
	return doJSON[Message](ctx, s, "sendMessage", payload)
}

// GetUpdates polls for incoming updates (long polling).
func (s *Client) GetUpdates(ctx context.Context, opts *GetUpdatesOptions) ([]Update, error) {
	payload := getUpdatesPayload{}
	if opts != nil {
		payload.Offset = opts.Offset
		payload.Limit = opts.Limit
		payload.Timeout = opts.Timeout
		payload.AllowedUpdates = opts.AllowedUpdates
	}
	return doJSON[[]Update](ctx, s, "getUpdates", payload)
}

// SetWebhook configures webhook URL.
func (s *Client) SetWebhook(ctx context.Context, url string) (bool, error) {
	payload := map[string]string{"url": url}
	return doJSON[bool](ctx, s, "setWebhook", payload)
}

// DeleteWebhook removes webhook and returns status.
func (s *Client) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) (bool, error) {
	payload := map[string]bool{"drop_pending_updates": dropPendingUpdates}
	return doJSON[bool](ctx, s, "deleteWebhook", payload)
}

// AnswerCallbackQuery answers a callback query from an inline keyboard.
func (s *Client) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, opts *AnswerCallbackQueryOptions) (bool, error) {
	payload := answerCallbackQueryPayload{CallbackQueryID: callbackQueryID}
	if opts != nil {
		payload.Text = opts.Text
		payload.ShowAlert = opts.ShowAlert
	}
	return doJSON[bool](ctx, s, "answerCallbackQuery", payload)
}

// EditMessageText edits text of a message.
func (s *Client) EditMessageText(ctx context.Context, chatID int64, messageID int64, text string, opts *EditMessageTextOptions) (Message, error) {
	payload := editMessageTextPayload{
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
	}
	if opts != nil {
		payload.ParseMode = opts.ParseMode
		payload.ReplyMarkup = opts.ReplyMarkup
	}
	return doJSON[Message](ctx, s, "editMessageText", payload)
}

// DeleteMessage deletes a message.
func (s *Client) DeleteMessage(ctx context.Context, chatID int64, messageID int64) (bool, error) {
	payload := map[string]int64{"chat_id": chatID, "message_id": messageID}
	return doJSON[bool](ctx, s, "deleteMessage", payload)
}

// SetMyCommands sets the bot's command list.
func (s *Client) SetMyCommands(ctx context.Context, commands []BotCommand) (bool, error) {
	payload := map[string]any{"commands": commands}
	return doJSON[bool](ctx, s, "setMyCommands", payload)
}

// SendPhoto sends a photo. InputFile can be FileID/URL or a new upload.
func (s *Client) SendPhoto(ctx context.Context, chatID int64, photo InputFile, opts *SendPhotoOptions) (Message, error) {
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
		return doMultipart[Message](ctx, s, "sendPhoto", fields, files)
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
	return doJSON[Message](ctx, s, "sendPhoto", payload)
}

// SendDocument sends a document. InputFile can be FileID/URL or a new upload.
func (s *Client) SendDocument(ctx context.Context, chatID int64, doc InputFile, opts *SendDocumentOptions) (Message, error) {
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
		return doMultipart[Message](ctx, s, "sendDocument", fields, files)
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
	return doJSON[Message](ctx, s, "sendDocument", payload)
}

// GetFile resolves a file_id to a short-lived download path — the first of
// Telegram's two steps to retrieve a file (see DownloadFile for the second).
func (s *Client) GetFile(ctx context.Context, fileID string) (string, error) {
	payload := map[string]string{"file_id": fileID}
	file, err := doJSON[File](ctx, s, "getFile", payload)
	if err != nil {
		return "", err
	}
	return file.FilePath, nil
}

// DownloadFile fetches a file's raw bytes given the path GetFile returned.
// Unlike every other Client method, this is a plain GET against the file
// host (api.telegram.org/file/bot<token>/...), not the Bot API host — so it
// does not go through doJSON.
func (s *Client) DownloadFile(ctx context.Context, filePath string) ([]byte, error) {
	endpoint, err := s.fileEndpoint(filePath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram: new request: %w", err)
	}
	res, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close() // nolint:errcheck
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram: download file: status %d", res.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(res.Body, maxDownloadSize+1))
	if err != nil {
		return nil, fmt.Errorf("telegram: read file: %w", err)
	}
	if len(data) > maxDownloadSize {
		return nil, fmt.Errorf("telegram: file exceeds %d bytes", maxDownloadSize)
	}
	return data, nil
}

// SendInvoice sends an invoice message. currency and providerToken are
// left to the caller rather than hardcoded, since go-tgbot is a
// general-purpose library — Stars-specific defaults (currency "XTR", empty
// providerToken) belong in the caller.
func (s *Client) SendInvoice(ctx context.Context, chatID int64, title, description, payload, currency, providerToken string, prices []LabeledPrice, opts *SendInvoiceOptions) (Message, error) {
	body := sendInvoicePayload{
		ChatID:        chatID,
		Title:         title,
		Description:   description,
		Payload:       payload,
		ProviderToken: providerToken,
		Currency:      currency,
		Prices:        prices,
	}
	if opts != nil {
		body.ReplyMarkup = opts.ReplyMarkup
	}
	return doJSON[Message](ctx, s, "sendInvoice", body)
}

// AnswerPreCheckoutQuery answers a pre_checkout_query. Must be called
// within 10 seconds of receiving it. Pass ok=false with a human-readable
// errorMessage to reject (Telegram shows it to the user); errorMessage is
// ignored when ok=true.
func (s *Client) AnswerPreCheckoutQuery(ctx context.Context, preCheckoutQueryID string, ok bool, errorMessage string) (bool, error) {
	return doJSON[bool](ctx, s, "answerPreCheckoutQuery", answerPreCheckoutQueryPayload{
		PreCheckoutQueryID: preCheckoutQueryID,
		OK:                 ok,
		ErrorMessage:       errorMessage,
	})
}

// RefundStarPayment reverses a completed Telegram Stars payment.
func (s *Client) RefundStarPayment(ctx context.Context, userID int64, telegramPaymentChargeID string) (bool, error) {
	return doJSON[bool](ctx, s, "refundStarPayment", refundStarPaymentPayload{
		UserID:                  userID,
		TelegramPaymentChargeID: telegramPaymentChargeID,
	})
}

// do sends the request. The bot token is part of every request URL, and
// net/http puts that URL into its errors (*url.Error), so on failure the
// token is replaced with "***" before the error reaches callers and logs.
// The error chain is kept: errors.Is(err, context.DeadlineExceeded) works.
func (s *Client) do(req *http.Request) (*http.Response, error) {
	res, err := s.http.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			urlErr.URL = strings.ReplaceAll(urlErr.URL, s.token, "***")
		}
		return nil, fmt.Errorf("telegram: request failed: %w", err)
	}
	return res, nil
}

func (s *Client) fileEndpoint(filePath string) (string, error) {
	return s.buildURL("file", "bot"+s.token, filePath)
}

func (s *Client) endpoint(method string) (string, error) {
	return s.buildURL("bot"+s.token, method)
}

// buildURL joins segments onto baseURL's path — shared by endpoint (the Bot
// API host) and fileEndpoint (the file-download host), which differ only in
// which segments they join.
func (s *Client) buildURL(segments ...string) (string, error) {
	if s.baseURL == "" {
		return "", errors.New("telegram: baseURL is empty")
	}
	u, err := url.Parse(s.baseURL)
	if err != nil {
		return "", fmt.Errorf("telegram: invalid baseURL: %w", err)
	}
	u.Path = path.Join(append([]string{u.Path}, segments...)...)
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

func (s *APIError) Error() string {
	if s == nil {
		return "telegram: API error"
	}
	if s.Code == 0 {
		return fmt.Sprintf("telegram: %s", s.Description)
	}
	return fmt.Sprintf("telegram: %d %s", s.Code, s.Description)
}

func doJSON[T any](ctx context.Context, c *Client, method string, payload any) (T, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("telegram: marshal payload: %w", err)
	}
	return call[T](
		ctx,
		c,
		apiRequest{
			Method:      method,
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func doMultipart[T any](ctx context.Context, c *Client, method string, fields map[string]string, files []formFile) (T, error) {
	var zero T
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
	return call[T](
		ctx,
		c,
		apiRequest{
			Method:      method,
			ContentType: writer.FormDataContentType(),
			Body:        buf.Bytes(),
		},
	)
}

// call sends one Bot API request. When Telegram asks to wait (429) no longer
// than maxRetryWait, it waits and sends the same body once more — the body
// is kept in memory for that.
func call[T any](ctx context.Context, c *Client, r apiRequest) (T, error) {
	res, err := send[T](ctx, c, r)
	if c.maxRetryWait <= 0 {
		return res, err
	}
	wait, asked := RetryAfter(err)
	if !asked || wait > c.maxRetryWait {
		return res, err
	}
	sleep(ctx, wait)
	if ctx.Err() != nil {
		return res, fmt.Errorf("telegram: waiting to retry %s: %w", r.Method, ctx.Err())
	}
	return send[T](ctx, c, r)
}

func send[T any](ctx context.Context, c *Client, r apiRequest) (T, error) {
	var zero T
	endpoint, err := c.endpoint(r.Method)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(r.Body))
	if err != nil {
		return zero, fmt.Errorf("telegram: new request: %w", err)
	}
	req.Header.Set("Content-Type", r.ContentType)
	res, err := c.do(req)
	if err != nil {
		return zero, err
	}
	defer res.Body.Close() // nolint:errcheck

	var apiRes apiResponse[T]
	if err := json.NewDecoder(res.Body).Decode(&apiRes); err != nil {
		return zero, fmt.Errorf("telegram: decode response: %w", err)
	}
	if !apiRes.Ok {
		return zero, &APIError{
			Code:        apiRes.ErrorCode,
			Description: apiRes.Description,
			Parameters:  apiRes.Parameters,
		}
	}
	return apiRes.Result, nil
}
