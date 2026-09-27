package tgbot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMessage_UnmarshalsDocumentField(t *testing.T) {
	raw := `{"message_id":1,"date":0,"chat":{"id":1},"document":{"file_id":"abc123","file_name":"notes.docx","mime_type":"application/vnd.openxmlformats-officedocument.wordprocessingml.document","file_size":2048}}`

	var m Message
	require.NoError(t, json.Unmarshal([]byte(raw), &m))

	require.NotNil(t, m.Document)
	require.Equal(t, "abc123", m.Document.FileID)
	require.Equal(t, "notes.docx", m.Document.FileName)
	require.Equal(t, "application/vnd.openxmlformats-officedocument.wordprocessingml.document", m.Document.MimeType)
	require.Equal(t, int64(2048), m.Document.FileSize)
}

func TestMessage_NoDocumentFieldLeavesNilPointer(t *testing.T) {
	var m Message
	require.NoError(t, json.Unmarshal([]byte(`{"message_id":1,"date":0,"chat":{"id":1},"text":"hi"}`), &m))

	require.Nil(t, m.Document)
}

func TestGetFile_ReturnsFilePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/bottest-token/getFile", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"file_id":"abc123","file_path":"documents/file_1.docx","file_size":2048}}`))
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	filePath, err := c.GetFile(context.Background(), "abc123")

	require.NoError(t, err)
	require.Equal(t, "documents/file_1.docx", filePath)
}

func TestGetFile_APIErrorPropagates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: file is too big"}`))
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = c.GetFile(context.Background(), "abc123")

	require.Error(t, err)
}

func TestDownloadFile_ReturnsBytes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/file/bottest-token/documents/file_1.docx", r.URL.Path)
		_, _ = w.Write([]byte("file contents"))
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	data, err := c.DownloadFile(context.Background(), "documents/file_1.docx")

	require.NoError(t, err)
	require.Equal(t, []byte("file contents"), data)
}

func TestDownloadFile_NonOKStatusReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = c.DownloadFile(context.Background(), "documents/missing.docx")

	require.Error(t, err)
}

// TestDownloadFile_OversizedBodyReturnsError is a generic library-level
// safety net, independent of any caller's own size policy: a response body
// bigger than Telegram's own documented Bot API download ceiling must never
// be read fully into memory, regardless of what the file metadata claimed
// (Document.file_size is optional and cannot be trusted as a hard limit).
func TestDownloadFile_OversizedBodyReturnsError(t *testing.T) {
	oversized := bytes.Repeat([]byte("a"), maxDownloadSize+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(oversized)
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	_, err = c.DownloadFile(context.Background(), "documents/huge.docx")

	require.Error(t, err)
}

func TestMessage_UnmarshalsSuccessfulPaymentField(t *testing.T) {
	raw := `{"message_id":1,"date":0,"chat":{"id":1},"successful_payment":{"currency":"XTR","total_amount":10,"invoice_payload":"pa:abc123","telegram_payment_charge_id":"tpc_1","provider_payment_charge_id":""}}`

	var m Message
	require.NoError(t, json.Unmarshal([]byte(raw), &m))

	require.NotNil(t, m.SuccessfulPayment)
	require.Equal(t, "XTR", m.SuccessfulPayment.Currency)
	require.Equal(t, int64(10), m.SuccessfulPayment.TotalAmount)
	require.Equal(t, "pa:abc123", m.SuccessfulPayment.InvoicePayload)
	require.Equal(t, "tpc_1", m.SuccessfulPayment.TelegramPaymentChargeID)
}

func TestUpdate_UnmarshalsPreCheckoutQueryField(t *testing.T) {
	raw := `{"update_id":1,"pre_checkout_query":{"id":"pcq_1","from":{"id":42,"is_bot":false,"first_name":"A"},"currency":"XTR","total_amount":10,"invoice_payload":"pa:abc123"}}`

	var u Update
	require.NoError(t, json.Unmarshal([]byte(raw), &u))

	require.NotNil(t, u.PreCheckoutQuery)
	require.Equal(t, "pcq_1", u.PreCheckoutQuery.ID)
	require.Equal(t, int64(42), u.PreCheckoutQuery.From.ID)
	require.Equal(t, "XTR", u.PreCheckoutQuery.Currency)
	require.Equal(t, int64(10), u.PreCheckoutQuery.TotalAmount)
	require.Equal(t, "pa:abc123", u.PreCheckoutQuery.InvoicePayload)
}

func TestSendInvoice_SendsCorrectRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, float64(100), body["chat_id"])
		require.Equal(t, "Phrase Analysis", body["title"])
		require.Equal(t, "pa:abc123", body["payload"])
		require.Equal(t, "", body["provider_token"])
		require.Equal(t, "XTR", body["currency"])
		prices, _ := body["prices"].([]any)
		require.Len(t, prices, 1)
		price, _ := prices[0].(map[string]any)
		require.Equal(t, float64(10), price["amount"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":55,"date":0,"chat":{"id":100}}}`))
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	msg, err := c.SendInvoice(context.Background(), 100, "Phrase Analysis", "One request", "pa:abc123", "XTR", "",
		[]LabeledPrice{{Label: "Phrase Analysis", Amount: 10}}, nil)

	require.NoError(t, err)
	require.Equal(t, int64(55), msg.MessageID)
}

func TestAnswerPreCheckoutQuery_SendsCorrectRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "pcq_1", body["pre_checkout_query_id"])
		require.Equal(t, true, body["ok"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	ok, err := c.AnswerPreCheckoutQuery(context.Background(), "pcq_1", true, "")

	require.NoError(t, err)
	require.True(t, ok)
}

func TestRefundStarPayment_SendsCorrectRequestBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, float64(42), body["user_id"])
		require.Equal(t, "tpc_1", body["telegram_payment_charge_id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
	}))
	defer server.Close()
	c, err := NewClient("test-token", WithBaseURL(server.URL))
	require.NoError(t, err)

	ok, err := c.RefundStarPayment(context.Background(), 42, "tpc_1")

	require.NoError(t, err)
	require.True(t, ok)
}

// deadServer returns the URL of a server that is already closed, so every
// request fails with a network error.
func deadServer() string {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	return server.URL
}

func TestNetworkErrorsHideToken(t *testing.T) {
	const token = "123456:SECRET-token"
	c, err := NewClient(token, WithBaseURL(deadServer()))
	require.NoError(t, err)
	ctx := context.Background()

	_, jsonErr := c.GetMe(ctx)
	_, multipartErr := c.SendDocument(
		ctx,
		1,
		InputFile{
			Reader:   bytes.NewReader([]byte("x")),
			Filename: "a.txt",
		},
		nil,
	)
	_, downloadErr := c.DownloadFile(ctx, "docs/a.txt")

	for name, err := range map[string]error{
		"json":      jsonErr,
		"multipart": multipartErr,
		"download":  downloadErr,
	} {
		require.Error(t, err, name)
		require.NotContains(t, err.Error(), "SECRET", name)
		require.Contains(t, err.Error(), "/bot***/", name)
	}
}

func TestNetworkErrorKeepsCause(t *testing.T) {
	c, err := NewClient("123456:SECRET-token", WithBaseURL(deadServer()))
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = c.GetMe(ctx)
	require.ErrorIs(t, err, context.Canceled, "callers can still check the cause")
	require.NotContains(t, err.Error(), "SECRET")
}

// floodServer answers the first `floods` calls with a 429 asking to wait
// retryAfter seconds, and every later call with ok. It counts the calls.
func floodServer(t *testing.T, floods, retryAfter int) (url string, calls *atomic.Int32) {
	t.Helper()
	calls = &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if int(n) <= floods {
			_, _ = fmt.Fprintf(w, `{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":%d}}`, retryAfter)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":5,"chat":{"id":1}}}`))
	}))
	t.Cleanup(server.Close)
	return server.URL, calls
}

func TestRetryAfter_WaitsAndSendsAgain(t *testing.T) {
	url, calls := floodServer(t, 1, 1)
	c, err := NewClient("test-token", WithBaseURL(url), WithRetryAfter(2*time.Second))
	require.NoError(t, err)

	start := time.Now()
	msg, err := c.SendMessage(context.Background(), 1, "hi", nil)

	require.NoError(t, err)
	require.Equal(t, int64(5), msg.MessageID)
	require.EqualValues(t, 2, calls.Load())
	require.GreaterOrEqual(t, time.Since(start), time.Second, "it waited as Telegram asked")
}

func TestRetryAfter_RetriesUploadsToo(t *testing.T) {
	url, calls := floodServer(t, 1, 1)
	c, err := NewClient("test-token", WithBaseURL(url), WithRetryAfter(2*time.Second))
	require.NoError(t, err)

	_, err = c.SendDocument(
		context.Background(),
		1,
		InputFile{
			Reader:   bytes.NewReader([]byte("x")),
			Filename: "a.txt",
		},
		nil,
	)

	require.NoError(t, err)
	require.EqualValues(t, 2, calls.Load())
}

func TestRetryAfter_OnlyOnce(t *testing.T) {
	url, calls := floodServer(t, 5, 1)
	c, err := NewClient("test-token", WithBaseURL(url), WithRetryAfter(2*time.Second))
	require.NoError(t, err)

	_, err = c.SendMessage(context.Background(), 1, "hi", nil)

	_, asked := RetryAfter(err)
	require.True(t, asked, "the second 429 goes to the caller")
	require.EqualValues(t, 2, calls.Load())
}

func TestRetryAfter_LongWaitGoesToCaller(t *testing.T) {
	url, calls := floodServer(t, 1, 30)
	c, err := NewClient("test-token", WithBaseURL(url), WithRetryAfter(2*time.Second))
	require.NoError(t, err)

	_, err = c.SendMessage(context.Background(), 1, "hi", nil)

	d, asked := RetryAfter(err)
	require.True(t, asked)
	require.Equal(t, 30*time.Second, d)
	require.EqualValues(t, 1, calls.Load())
}

func TestRetryAfter_OffByDefault(t *testing.T) {
	url, calls := floodServer(t, 1, 1)
	c, err := NewClient("test-token", WithBaseURL(url))
	require.NoError(t, err)

	_, err = c.SendMessage(context.Background(), 1, "hi", nil)

	require.Error(t, err)
	require.EqualValues(t, 1, calls.Load())
}

func TestRetryAfter_StopsWhenContextEnds(t *testing.T) {
	url, calls := floodServer(t, 1, 1)
	c, err := NewClient("test-token", WithBaseURL(url), WithRetryAfter(2*time.Second))
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = c.SendMessage(ctx, 1, "hi", nil)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.EqualValues(t, 1, calls.Load())
}
