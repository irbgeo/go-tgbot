package tgbot

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const dispatchWait = 2 * time.Second

func chatUpdate(id, chatID int64) Update {
	return Update{
		UpdateID: id,
		Message: &Message{
			From: &User{ID: chatID},
			Chat: Chat{ID: chatID},
		},
	}
}

func preCheckoutUpdate(id int64) Update {
	return Update{
		UpdateID:         id,
		PreCheckoutQuery: &PreCheckoutQuery{ID: "pcq"},
	}
}

func receive(t *testing.T, ch <-chan int64) int64 {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(dispatchWait):
		require.FailNow(t, "the update was not handled in time")
		return 0
	}
}

// One slow request must not stop the poll loop: Handle returns at once, and an update from
// another chat is handled while the first is still running.
func TestDispatcher_DifferentChatsRunInParallel(t *testing.T) {
	release := make(chan struct{})
	handled := make(chan int64, 4)
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		if upd.UpdateID == 1 {
			<-release
		}
		handled <- upd.UpdateID
		return nil
	}, nil)

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	require.NoError(t, d.Handle(context.Background(), chatUpdate(2, 200)))

	require.EqualValues(t, 2, receive(t, handled), "the second chat is not blocked by the first")
	close(release)
	require.EqualValues(t, 1, receive(t, handled))
	d.Shutdown(time.Minute)
}

// A pre-checkout query must be answered within 10 seconds, so it never waits in the queue of a
// chat that is busy.
func TestDispatcher_PreCheckoutQueryIsNeverQueued(t *testing.T) {
	release := make(chan struct{})
	handled := make(chan int64, 4)
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		if upd.UpdateID == 1 {
			<-release
		}
		handled <- upd.UpdateID
		return nil
	}, nil)

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	require.NoError(t, d.Handle(context.Background(), preCheckoutUpdate(2)))

	require.EqualValues(t, 2, receive(t, handled))
	close(release)
	require.EqualValues(t, 1, receive(t, handled))
	d.Shutdown(time.Minute)
}

// A user who presses a button and then sends a text must be handled in that order.
func TestDispatcher_UpdatesOfOneChatRunInOrder(t *testing.T) {
	var mu sync.Mutex
	var order []int64
	done := make(chan struct{}, 5)
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		time.Sleep(time.Duration(5-upd.UpdateID) * time.Millisecond) // later updates are faster
		mu.Lock()
		order = append(order, upd.UpdateID)
		mu.Unlock()
		done <- struct{}{}
		return nil
	}, nil)

	for id := int64(1); id <= 5; id++ {
		require.NoError(t, d.Handle(context.Background(), chatUpdate(id, 100)))
	}
	for range 5 {
		<-done // Shutdown would stop the queue, so let the work finish first
	}
	d.Shutdown(time.Minute)

	require.Equal(t, []int64{1, 2, 3, 4, 5}, order)
}

// Stop lets what is running finish and drops what is only queued: draining
// the whole backlog could run far past the stop grace period.
func TestDispatcher_StopDropsQueuedUpdates(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var mu sync.Mutex
	var handled []int64
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		mu.Lock()
		handled = append(handled, upd.UpdateID)
		mu.Unlock()
		if upd.UpdateID == 1 {
			close(started)
			<-release
		}
		return nil
	}, nil)

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	<-started
	require.NoError(t, d.Handle(context.Background(), chatUpdate(2, 100)))
	d.Stop()
	require.NoError(t, d.Handle(context.Background(), chatUpdate(3, 100)), "an update arriving after Stop is dropped, not an error")
	close(release)
	d.Shutdown(time.Minute)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []int64{1}, handled, "only the update that had already started ran")
}

func TestDispatcher_CallbackQueryIsQueuedWithItsChat(t *testing.T) {
	release := make(chan struct{})
	handled := make(chan int64, 4)
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		if upd.UpdateID == 1 {
			<-release
		}
		handled <- upd.UpdateID
		return nil
	}, nil)
	callback := Update{
		UpdateID: 2,
		CallbackQuery: &CallbackQuery{
			From:    User{ID: 100},
			Message: &Message{Chat: Chat{ID: 100}},
		},
	}

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	require.NoError(t, d.Handle(context.Background(), callback))
	release <- struct{}{}

	require.EqualValues(t, 1, receive(t, handled))
	require.EqualValues(t, 2, receive(t, handled))
	d.Shutdown(time.Minute)
}

func TestDispatcher_HandlerErrorGoesToOnError(t *testing.T) {
	wantErr := errors.New("boom")
	errs := make(chan error, 1)
	d := NewDispatcher(func(context.Context, Update) error { return wantErr }, func(err error) { errs <- err })

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	d.Shutdown(time.Minute)

	require.ErrorIs(t, <-errs, wantErr)
}

// A panic in one request must not take the whole bot down with every other user's request.
func TestDispatcher_PanicIsReportedNotFatal(t *testing.T) {
	errs := make(chan error, 1)
	d := NewDispatcher(func(context.Context, Update) error { panic("bad update") }, func(err error) { errs <- err })

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	d.Shutdown(time.Minute)

	require.ErrorContains(t, <-errs, "bad update")
}

// On shutdown the poll context is cancelled, but a request that already started (maybe a paid
// one) must be able to finish and refund, so the handler gets a context that is not cancelled.
func TestDispatcher_HandlerIsNotCancelledByShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	seen := make(chan error, 1)
	d := NewDispatcher(func(hctx context.Context, _ Update) error {
		time.Sleep(20 * time.Millisecond)
		seen <- hctx.Err()
		return nil
	}, nil)

	require.NoError(t, d.Handle(ctx, chatUpdate(1, 100)))
	cancel()
	d.Shutdown(time.Minute)

	require.NoError(t, <-seen)
}

func TestDispatcher_UpdateWithoutAChatStillRuns(t *testing.T) {
	handled := make(chan int64, 1)
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		handled <- upd.UpdateID
		return nil
	}, nil)

	require.NoError(t, d.Handle(context.Background(), Update{UpdateID: 7}))

	require.EqualValues(t, 7, receive(t, handled))
	d.Shutdown(time.Minute)
}

func chatPaymentUpdate(id, chatID int64) Update {
	upd := chatUpdate(id, chatID)
	upd.Message.SuccessfulPayment = &SuccessfulPayment{
		TelegramPaymentChargeID: "tpc",
	}
	return upd
}

// A successful payment is a chat message, but it must not wait behind a
// long request: a queued update is lost at shutdown, and Telegram never
// sends it again, so the user would pay for nothing.
func TestDispatcher_PaymentIsNeverQueued(t *testing.T) {
	release := make(chan struct{})
	handled := make(chan int64, 4)
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		if upd.UpdateID == 1 {
			<-release
		}
		handled <- upd.UpdateID
		return nil
	}, nil)

	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	require.NoError(t, d.Handle(context.Background(), chatPaymentUpdate(2, 100)))

	require.EqualValues(t, 2, receive(t, handled), "the payment runs while the chat is busy")
	close(release)
	require.EqualValues(t, 1, receive(t, handled))
	d.Shutdown(time.Minute)
}

// A payment that arrives after Stop still runs, and Shutdown waits for it.
func TestDispatcher_PaymentAfterStopStillRuns(t *testing.T) {
	var mu sync.Mutex
	var handled []int64
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		time.Sleep(10 * time.Millisecond)
		mu.Lock()
		handled = append(handled, upd.UpdateID)
		mu.Unlock()
		return nil
	}, nil)

	d.Stop()
	require.NoError(t, d.Handle(context.Background(), chatPaymentUpdate(1, 100)))
	d.Shutdown(time.Minute)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []int64{1}, handled)
}

// Past the grace period, Shutdown cancels what still runs, so a paid
// request can refund before the container is killed.
func TestDispatcher_ShutdownCancelsWorkPastTheGrace(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	d := NewDispatcher(func(ctx context.Context, _ Update) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}, nil)
	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	<-started

	d.Shutdown(20 * time.Millisecond)

	select {
	case <-cancelled:
	default:
		require.FailNow(t, "the running update was not cancelled")
	}
}

// Work that ends within the grace period is not cancelled.
func TestDispatcher_ShutdownLetsQuickWorkFinish(t *testing.T) {
	var gotErr error
	d := NewDispatcher(func(ctx context.Context, _ Update) error {
		time.Sleep(10 * time.Millisecond)
		gotErr = ctx.Err()
		return nil
	}, nil)
	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))

	d.Shutdown(dispatchWait)

	require.NoError(t, gotErr)
}

// Updates dropped at shutdown are reported, not silently lost.
func TestDispatcher_StopReportsDroppedUpdates(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	var mu sync.Mutex
	var reported []error
	d := NewDispatcher(func(_ context.Context, upd Update) error {
		if upd.UpdateID == 1 {
			close(started)
			<-release
		}
		return nil
	}, func(err error) {
		mu.Lock()
		reported = append(reported, err)
		mu.Unlock()
	})
	require.NoError(t, d.Handle(context.Background(), chatUpdate(1, 100)))
	<-started
	require.NoError(t, d.Handle(context.Background(), chatUpdate(2, 100)))

	d.Stop()
	close(release)
	d.Shutdown(time.Minute)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, reported, 1)
	require.Contains(t, reported[0].Error(), "update 2")
}
