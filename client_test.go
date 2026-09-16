package tgbot

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
