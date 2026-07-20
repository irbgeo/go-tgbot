package tgbot

import (
	"testing"
	"time"
)

func TestKeyboard(t *testing.T) {
	if Keyboard() != nil {
		t.Fatalf("Keyboard() with no rows should be nil")
	}
	kb := Keyboard(
		Row(Button("Yes", "y"), Button("No", "n")),
		Row(URLButton("Open", "https://example.com")),
	)
	if kb == nil || len(kb.InlineKeyboard) != 2 {
		t.Fatalf("expected 2 rows, got %+v", kb)
	}
	if kb.InlineKeyboard[0][0].Text != "Yes" || kb.InlineKeyboard[0][0].CallbackData != "y" {
		t.Fatalf("button 0 wrong: %+v", kb.InlineKeyboard[0][0])
	}
	if kb.InlineKeyboard[1][0].URL != "https://example.com" {
		t.Fatalf("url button wrong: %+v", kb.InlineKeyboard[1][0])
	}
}

func TestUpdateAccessors(t *testing.T) {
	msg := Update{Message: &Message{Chat: Chat{ID: 5}, From: &User{ID: 7}, Text: "hi"}}
	if msg.SenderID() != 7 || msg.ChatID() != 5 {
		t.Fatalf("message accessors: sender=%d chat=%d", msg.SenderID(), msg.ChatID())
	}
	cb := Update{CallbackQuery: &CallbackQuery{From: User{ID: 9}, Message: &Message{Chat: Chat{ID: 5}}, Data: "x"}}
	if cb.SenderID() != 9 || cb.ChatID() != 5 {
		t.Fatalf("callback accessors: sender=%d chat=%d", cb.SenderID(), cb.ChatID())
	}
	if (Update{}).SenderID() != 0 || (Update{}).ChatID() != 0 {
		t.Fatalf("empty update should be 0/0")
	}
}

func TestCommand(t *testing.T) {
	tests := []struct {
		text  string
		want  string
		isCmd bool
	}{
		{"/start", "start", true},
		{"/start@my_bot", "start", true},
		{"/menu arg1 arg2", "menu", true},
		{"hello", "", false},
		{"  /help  ", "help", true},
		{"/", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, ok := Update{Message: &Message{Text: tt.text}}.Command()
		if got != tt.want || ok != tt.isCmd {
			t.Errorf("Command(%q) = (%q,%v), want (%q,%v)", tt.text, got, ok, tt.want, tt.isCmd)
		}
	}
	if _, ok := (Update{}).Command(); ok {
		t.Errorf("non-message update should not be a command")
	}
}

func TestAPIErrorHelpers(t *testing.T) {
	if !(&APIError{Code: 403, Description: "Forbidden: bot was blocked by the user"}).IsForbidden() {
		t.Errorf("403 should be forbidden")
	}
	if (&APIError{Code: 400}).IsForbidden() {
		t.Errorf("400 should not be forbidden")
	}
	if !(&APIError{Code: 400, Description: "Bad Request: message is not modified"}).IsNotModified() {
		t.Errorf("should detect not-modified")
	}
	d, ok := (&APIError{Code: 429, Parameters: &ResponseParameters{RetryAfter: 5}}).RetryAfter()
	if !ok || d != 5*time.Second {
		t.Errorf("RetryAfter = (%s,%v), want (5s,true)", d, ok)
	}
	if _, ok := (&APIError{Code: 500}).RetryAfter(); ok {
		t.Errorf("no retry hint should be false")
	}
	// Nil-safe.
	var e *APIError
	if e.IsForbidden() || e.IsNotModified() {
		t.Errorf("nil APIError helpers should be false")
	}
}
