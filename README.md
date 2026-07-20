# go-tgbot

A small Go client for the [Telegram Bot API](https://core.telegram.org/bots/api).

It has **no external dependencies**. It uses only the Go standard library
(`net/http`, `encoding/json`, `mime/multipart`).

- Import path: `github.com/irbgeo/go-tgbot`
- Package name: `tgbot`
- Go version: 1.25+

## Install

```bash
go get github.com/irbgeo/go-tgbot
```

Then import it:

```go
import tgbot "github.com/irbgeo/go-tgbot"
```

## Quick start

First you need a bot token. You get it from [@BotFather](https://t.me/BotFather).

```go
package main

import (
	"context"
	"log"

	tgbot "github.com/irbgeo/go-tgbot"
)

func main() {
	client, err := tgbot.NewClient("YOUR_BOT_TOKEN")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	me, err := client.GetMe(ctx)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bot username: @%s", me.Username)

	_, err = client.SendMessage(ctx, 123456789, "Hello!", nil)
	if err != nil {
		log.Fatal(err)
	}
}
```

Every method takes a `context.Context`. You can use it to set a timeout or to
cancel a call.

## Create a client

`NewClient` needs a token. It returns an error if the token is empty.

```go
client, err := tgbot.NewClient("YOUR_BOT_TOKEN")
```

You can pass options to change the default behavior.

### WithHTTPClient

Use your own `*http.Client`. This is useful to set a different timeout or a proxy.
The default client has a 15 second timeout.

```go
hc := &http.Client{Timeout: 30 * time.Second}
client, err := tgbot.NewClient("TOKEN", tgbot.WithHTTPClient(hc))
```

### WithBaseURL

Change the API base URL. The value is the API root **without** the bot token.
This is useful for tests or for a local Telegram Bot API server.

```go
client, err := tgbot.NewClient("TOKEN", tgbot.WithBaseURL("https://api.telegram.org"))
```

## Getting updates

There are two ways to get messages from users: **long polling** and **webhooks**.

### Long polling

Call `GetUpdates` in a loop. Save the last `UpdateID` and add `1` to use it as
the next `Offset`. This way you do not get the same update twice.

```go
var offset int64

for {
	updates, err := client.GetUpdates(ctx, &tgbot.GetUpdatesOptions{
		Offset:  offset,
		Timeout: 30, // long polling timeout in seconds
	})
	if err != nil {
		log.Println("getUpdates error:", err)
		time.Sleep(time.Second)
		continue
	}

	for _, u := range updates {
		offset = u.UpdateID + 1

		if u.Message != nil {
			log.Printf("message from %d: %s", u.Message.Chat.ID, u.Message.Text)
		}
		if u.CallbackQuery != nil {
			log.Printf("callback data: %s", u.CallbackQuery.Data)
		}
	}
}
```

`GetUpdatesOptions` fields:

| Field            | Type       | Meaning                                          |
|------------------|------------|--------------------------------------------------|
| `Offset`         | `int64`    | First update ID to return. Use `lastID + 1`.     |
| `Limit`          | `int`      | Max number of updates (1–100).                   |
| `Timeout`        | `int`      | Long polling timeout in seconds.                 |
| `AllowedUpdates` | `[]string` | Update types you want (e.g. `["message"]`).      |

### Poll helper (recommended)

Writing the polling loop by hand is easy to get wrong. The `Poll` method does it
for you. It tracks the offset, handles the `429` rate limit, backs off on errors,
and stops cleanly when the context is cancelled.

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
defer stop()

err := client.Poll(ctx, tgbot.PollOptions{
	Timeout:        30, // long polling timeout in seconds
	AllowedUpdates: []string{"message", "callback_query"},
	OnError: func(err error) {
		log.Println("poll error:", err)
	},
}, func(ctx context.Context, u tgbot.Update) error {
	log.Printf("update from chat %d", u.ChatID())
	return nil
})
if err != nil {
	log.Fatal(err)
}
```

`Poll` returns `nil` when the context is cancelled, so the code above shuts down
on Ctrl+C. A handler error does **not** stop the loop — it is only reported
through `OnError`.

`PollOptions` fields:

| Field            | Type            | Meaning                                              |
|------------------|-----------------|------------------------------------------------------|
| `Timeout`        | `int`           | Long polling timeout in seconds. Default `10`. Keep it below the HTTP client timeout. |
| `Limit`          | `int`           | Max updates per call (`0` = server default).         |
| `AllowedUpdates` | `[]string`      | Update types you want (`nil` = all but `chat_member`). |
| `Backoff`        | `time.Duration` | Pause after a transient error. Default `3s`.         |
| `OnError`        | `func(error)`   | Called on a non-fatal error. The loop keeps running. |

### Webhooks

Tell Telegram to send updates to your URL.

```go
ok, err := client.SetWebhook(ctx, "https://example.com/my-bot-hook")
```

To stop the webhook:

```go
ok, err := client.DeleteWebhook(ctx, true) // true drops pending updates
```

When you use a webhook, Telegram sends an `Update` as JSON in the request body.
You decode it yourself:

```go
func handler(w http.ResponseWriter, r *http.Request) {
	var update tgbot.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	// handle update...
	w.WriteHeader(http.StatusOK)
}
```

Note: use **either** polling **or** a webhook, not both.

### Reading an update

An `Update` can hold a message or a callback query. Instead of checking the
nested fields by hand, use these helper methods. They work for both kinds and
return `0` or `false` when the value is not present.

```go
func handle(ctx context.Context, u tgbot.Update) error {
	chatID := u.ChatID()     // chat the update belongs to
	userID := u.SenderID()   // user who sent it or tapped the button

	// Command() strips the leading slash, the "@botname" part, and arguments.
	if cmd, ok := u.Command(); ok {
		switch cmd {
		case "start":
			_, err := client.SendMessage(ctx, chatID, "Welcome!", nil)
			return err
		case "help":
			_, err := client.SendMessage(ctx, chatID, "Send me any text.", nil)
			return err
		}
	}

	if u.CallbackQuery != nil {
		// Acknowledge the tap so the button stops its loading spinner.
		_, _ = client.AnswerCallback(ctx, u.CallbackQuery.ID)
	}

	log.Printf("update from user %d in chat %d", userID, chatID)
	return nil
}
```

| Method       | Returns          | Meaning                                                |
|--------------|------------------|--------------------------------------------------------|
| `ChatID()`   | `int64`          | Chat ID of the message or the callback's message.      |
| `SenderID()` | `int64`          | User ID of the sender or the button presser.           |
| `Command()`  | `(string, bool)` | Bot command without `/`, `@botname`, or arguments.     |

## Sending messages

```go
msg, err := client.SendMessage(ctx, chatID, "Hello", nil)
```

To change the behavior, pass `*SendMessageOptions`:

```go
msg, err := client.SendMessage(ctx, chatID, "*Bold* text", &tgbot.SendMessageOptions{
	ParseMode:             tgbot.ParseModeMarkdownV2,
	DisableWebPagePreview:  true,
	DisableNotification:    true,
	ReplyToMessageID:       someMessageID,
})
```

For the parse mode you can pass a plain string or use one of the constants:
`tgbot.ParseModeHTML`, `tgbot.ParseModeMarkdownV2`, `tgbot.ParseModeMarkdown`.

`SendMessageOptions` fields:

| Field                      | Type     | Meaning                                            |
|----------------------------|----------|----------------------------------------------------|
| `ParseMode`                | `string` | `"Markdown"`, `"MarkdownV2"`, or `"HTML"`.          |
| `DisableWebPagePreview`    | `bool`   | Hide the link preview.                              |
| `DisableNotification`      | `bool`   | Send without a sound.                               |
| `ReplyToMessageID`         | `int64`  | Reply to this message.                              |
| `AllowSendingWithoutReply` | `bool`   | Send even if the reply target is gone.             |
| `ReplyMarkup`              | `any`    | A keyboard (see below).                             |

### Edit a message

```go
_, err := client.EditMessageText(ctx, chatID, messageID, "New text", nil)
```

### Delete a message

```go
ok, err := client.DeleteMessage(ctx, chatID, messageID)
```

## Keyboards

`ReplyMarkup` accepts one of three types: `InlineKeyboardMarkup`,
`ReplyKeyboardMarkup`, or `ReplyKeyboardRemove`. Pass it by value.

### Inline keyboard

Buttons under the message. They send a callback or open a URL.

The library gives you small builder functions so you do not have to write the
nested slices by hand:

- `tgbot.Button(text, data)` — a button that sends `data` as a callback.
- `tgbot.URLButton(text, url)` — a button that opens a URL.
- `tgbot.Row(buttons...)` — one row of buttons.
- `tgbot.InlineKeyboard(rows...)` — the full markup. Returns `nil` when you pass
  no rows, so you can use it for a "no keyboard" case too.

```go
kb := tgbot.InlineKeyboard(
	tgbot.Row(
		tgbot.Button("Yes", "vote_yes"),
		tgbot.Button("No", "vote_no"),
	),
	tgbot.Row(
		tgbot.URLButton("Open site", "https://example.com"),
	),
)

_, err := client.SendMessage(ctx, chatID, "Do you agree?", &tgbot.SendMessageOptions{
	ReplyMarkup: kb,
})
```

The builders are optional. You can still build `tgbot.InlineKeyboardMarkup` by
hand if you prefer.

When the user taps an inline button, you get an `Update` with a
`CallbackQuery`. You should answer it, or the button keeps a loading state.

```go
// Full control:
_, err := client.AnswerCallbackQuery(ctx, query.ID, &tgbot.AnswerCallbackQueryOptions{
	Text:      "Thanks for your vote!",
	ShowAlert: false, // true shows a popup instead of a small toast
})

// Or just dismiss the spinner (the common case):
_, err = client.AnswerCallback(ctx, query.ID)
```

### Reply keyboard

A custom keyboard that replaces the normal keyboard. It has its own builders:

- `tgbot.TextButton(text)` — a button that sends its text back as a message.
- `tgbot.ReplyRow(buttons...)` — one row of buttons.
- `tgbot.ReplyKeyboard(rows...)` — the full markup. Returns `nil` for no rows.

```go
kb := tgbot.ReplyKeyboard(
	tgbot.ReplyRow(tgbot.TextButton("Help"), tgbot.TextButton("Settings")),
	tgbot.ReplyRow(tgbot.TextButton("Menu")),
)

_, err := client.SendMessage(ctx, chatID, "Choose:", &tgbot.SendMessageOptions{
	ReplyMarkup: kb,
})
```

`ReplyKeyboard` does not set the extra flags. If you need them, set them on the
returned value or build the struct by hand:

```go
kb.ResizeKeyboard = true
kb.OneTimeKeyboard = true
```

### Remove a reply keyboard

```go
_, err := client.SendMessage(ctx, chatID, "Done", &tgbot.SendMessageOptions{
	ReplyMarkup: tgbot.ReplyKeyboardRemove{RemoveKeyboard: true},
})
```

## Sending files

`SendPhoto` and `SendDocument` use the `InputFile` type. You can send a file in
three ways:

1. **Upload bytes** — set `Reader` (and `Filename`). The client uses multipart.
2. **By URL** — set `URL`. Telegram downloads the file.
3. **By file_id** — set `FileID`. Reuse a file that is already on Telegram.

### Upload a new photo

```go
f, err := os.Open("cat.jpg")
if err != nil {
	log.Fatal(err)
}
defer f.Close()

_, err = client.SendPhoto(ctx, chatID, tgbot.InputFile{
	Reader:   f,
	Filename: "cat.jpg",
}, &tgbot.SendPhotoOptions{
	Caption: "My cat",
})
```

### Send a photo by URL

```go
_, err := client.SendPhoto(ctx, chatID, tgbot.InputFile{
	URL: "https://example.com/cat.jpg",
}, nil)
```

### Send a document by file_id

```go
_, err := client.SendDocument(ctx, chatID, tgbot.InputFile{
	FileID: "BQACAgIAAxk...",
}, nil)
```

Both `SendPhotoOptions` and `SendDocumentOptions` have the same fields:

| Field       | Type     | Meaning                       |
|-------------|----------|-------------------------------|
| `Caption`   | `string` | Text under the file.          |
| `ParseMode` | `string` | Parse mode for the caption.   |

## Bot commands

Set the command list shown in the Telegram menu.

```go
ok, err := client.SetMyCommands(ctx, []tgbot.BotCommand{
	{Command: "start", Description: "Start the bot"},
	{Command: "help", Description: "Show help"},
})
```

## Error handling

When the Telegram API returns an error, the method returns an `*APIError`.
It has the HTTP-like error code and the description from Telegram. It can also
hold extra info in `Parameters` (for example, how long to wait).

Use `errors.As` to read it:

```go
_, err := client.SendMessage(ctx, chatID, "Hi", nil)
if err != nil {
	var apiErr *tgbot.APIError
	if errors.As(err, &apiErr) {
		log.Printf("telegram error %d: %s", apiErr.Code, apiErr.Description)

		// Rate limit: Telegram asks you to wait.
		if d, ok := apiErr.RetryAfter(); ok {
			time.Sleep(d)
		}
		return
	}
	// Other errors: network, JSON, context, etc.
	log.Println("request failed:", err)
}
```

`*APIError` has helper methods for the common cases, so you do not have to match
on codes or text yourself:

| Method            | Returns              | Meaning                                                        |
|-------------------|----------------------|----------------------------------------------------------------|
| `IsForbidden()`   | `bool`               | HTTP `403`. In a private chat the user blocked the bot.        |
| `IsNotModified()` | `bool`               | An `EditMessage*` call changed nothing. Safe to treat as OK.   |
| `RetryAfter()`    | `(time.Duration, bool)` | How long to wait after a `429`. `false` when there is no hint. |

All three are safe to call on a `nil` `*APIError`. Example:

```go
_, err := client.EditMessageText(ctx, chatID, messageID, text, nil)
var apiErr *tgbot.APIError
if errors.As(err, &apiErr) && apiErr.IsNotModified() {
	err = nil // the text was already the same
}
```

`ResponseParameters` fields:

| Field             | Type    | Meaning                                        |
|-------------------|---------|------------------------------------------------|
| `RetryAfter`      | `int`   | Wait this many seconds before you try again.   |
| `MigrateToChatID` | `int64` | The group became a supergroup with this new ID.|

## API reference

| Method                | What it does                                  |
|-----------------------|-----------------------------------------------|
| `GetMe`               | Get the bot's own info.                       |
| `SendMessage`         | Send a text message.                          |
| `EditMessageText`     | Edit the text of a message.                   |
| `DeleteMessage`       | Delete a message.                             |
| `GetUpdates`          | Poll for new updates (long polling).          |
| `Poll`                | Run the polling loop for you (offset, backoff, rate limit). |
| `SetWebhook`          | Set the webhook URL.                          |
| `DeleteWebhook`       | Remove the webhook.                           |
| `AnswerCallbackQuery` | Answer an inline button tap.                  |
| `AnswerCallback`      | Answer a button tap with no popup (shortcut). |
| `SendPhoto`           | Send a photo.                                 |
| `SendDocument`        | Send a document.                              |
| `SetMyCommands`       | Set the bot's command list.                   |

### Helper functions and methods

| Helper                                    | What it does                                        |
|-------------------------------------------|-----------------------------------------------------|
| `Button` / `URLButton`                    | Build one inline button (callback or URL).          |
| `Row` / `InlineKeyboard`                  | Build inline keyboard rows and markup.              |
| `TextButton`                              | Build one reply keyboard button.                    |
| `ReplyRow` / `ReplyKeyboard`              | Build reply keyboard rows and markup.               |
| `Update.ChatID` / `.SenderID`             | Read the chat and user IDs from any update.         |
| `Update.Command`                          | Read the bot command from a message.                |
| `APIError.IsForbidden` / `.IsNotModified` | Check common API error cases.                        |
| `APIError.RetryAfter`                     | Read the `429` wait time.                            |

## License

MIT — see the [LICENSE](LICENSE) file.
