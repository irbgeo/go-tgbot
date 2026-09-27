package tgbot

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Dispatcher hands each update to a handler without blocking the poll loop.
// Without it one slow request would stop the bot from reading anything else,
// and Telegram cancels a payment if the bot does not answer its pre-checkout
// query within 10 seconds. Give Dispatcher.Handle to Client.Poll and call
// Shutdown when Poll returns:
//
//	d := tgbot.NewDispatcher(router.Handle, logError)
//	err := client.Poll(ctx, tgbot.PollOptions{}, d.Handle)
//	d.Shutdown(45 * time.Second)
//
// Updates of one chat run one after another, in order (a mode button and the
// text after it must not swap). Different chats run in parallel. A
// pre-checkout query and a successful payment are never queued: an update
// the bot drops is gone for good, and a payment must never be.
type Dispatcher struct {
	handle  func(context.Context, Update) error
	onError func(error)

	mu     sync.Mutex
	queues map[int64]*updateQueue
	wg     sync.WaitGroup

	stopOnce sync.Once
	stop     chan struct{}

	// abort cancels every running handler (Shutdown, past its grace).
	abortCtx context.Context
	abort    context.CancelFunc
}

// maxPendingPerChat caps how much one chat may queue up. Past that its
// newest updates are dropped: one user must not be able to make the bot
// hold an unbounded backlog. Telegram re-sends nothing the poll loop has
// taken, so a dropped update is lost — which is why payments never queue.
const maxPendingPerChat = 50

// NewDispatcher builds a Dispatcher. onError may be nil.
func NewDispatcher(
	handle func(context.Context, Update) error,
	onError func(error),
) *Dispatcher {
	abortCtx, abort := context.WithCancel(context.Background())
	return &Dispatcher{
		handle:   handle,
		onError:  onError,
		queues:   make(map[int64]*updateQueue),
		stop:     make(chan struct{}),
		abortCtx: abortCtx,
		abort:    abort,
	}
}

// Handle starts the work for upd and returns at once. It is the function to
// give to the poll loop.
func (d *Dispatcher) Handle(ctx context.Context, upd Update) error {
	chatID := upd.ChatID()
	if chatID == 0 || upd.PreCheckoutQuery != nil || carriesPayment(upd) {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			d.run(ctx, upd)
		}()
		return nil
	}

	if d.stopping() {
		return nil // shutting down: nothing new is started
	}

	d.mu.Lock()
	if q, running := d.queues[chatID]; running {
		if len(q.pending) >= maxPendingPerChat {
			d.mu.Unlock()
			d.report(fmt.Errorf("telegram: chat %d has %d updates waiting, dropping update %d", chatID, len(q.pending), upd.UpdateID))
			return nil
		}
		q.pending = append(q.pending, upd)
		d.mu.Unlock()
		return nil
	}
	d.queues[chatID] = &updateQueue{}
	d.mu.Unlock()

	d.wg.Add(1)
	go d.drain(ctx, chatID, upd)
	return nil
}

// Stop tells the dispatcher to start nothing more from a chat queue.
// Updates already running finish; updates waiting in a queue are dropped
// and reported (Telegram does not send them again). Payments still run.
func (d *Dispatcher) Stop() {
	d.stopOnce.Do(func() { close(d.stop) })
}

// Shutdown stops the dispatcher and waits up to grace for running updates.
// Then it cancels their contexts and waits for them to end, so a handler can
// still clean up (for example, refund a paid request it could not finish).
// grace must leave time for that before the process is killed (Docker kills
// a container 10 s after a stop by default).
func (d *Dispatcher) Shutdown(grace time.Duration) {
	d.Stop()
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return
	case <-time.After(grace):
	}
	d.report(fmt.Errorf("telegram: updates still running after %s, cancelling them", grace))
	d.abort()
	<-done
}

// stopping reports whether Stop has been called.
func (d *Dispatcher) stopping() bool {
	select {
	case <-d.stop:
		return true
	default:
		return false
	}
}

// drain runs first, then every update that queued behind it, then removes
// the chat's queue.
func (d *Dispatcher) drain(ctx context.Context, chatID int64, first Update) {
	defer d.wg.Done()
	upd := first
	for {
		d.run(ctx, upd)

		d.mu.Lock()
		q := d.queues[chatID]
		if len(q.pending) == 0 || d.stopping() {
			delete(d.queues, chatID)
			d.mu.Unlock()
			for _, dropped := range q.pending {
				d.report(fmt.Errorf("telegram: shutting down, dropped queued update %d of chat %d", dropped.UpdateID, chatID))
			}
			return
		}
		upd = q.pending[0]
		q.pending = q.pending[1:]
		d.mu.Unlock()
	}
}

// run calls the handler. A panic or an error is only reported: one bad
// request must not stop the others.
func (d *Dispatcher) run(ctx context.Context, upd Update) {
	ctx, release := d.handlerContext(ctx)
	defer release()
	defer func() {
		if p := recover(); p != nil {
			d.report(fmt.Errorf("telegram: update %d panicked: %v", upd.UpdateID, p))
		}
	}()
	if err := d.handle(ctx, upd); err != nil {
		d.report(err)
	}
}

// handlerContext is ctx without its own cancellation, cancelled instead by
// Shutdown past its grace. The poll loop's ctx ends at the first stop
// signal, which must not cut a running request in half. release must be
// called when the handler is done.
func (d *Dispatcher) handlerContext(ctx context.Context) (handlerCtx context.Context, release func()) {
	handlerCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	stopAbort := context.AfterFunc(d.abortCtx, cancel)
	return handlerCtx, func() {
		stopAbort()
		cancel()
	}
}

func (d *Dispatcher) report(err error) {
	if d.onError != nil {
		d.onError(err)
	}
}

// carriesPayment reports whether upd tells the bot about money received.
func carriesPayment(upd Update) bool {
	return upd.Message != nil && upd.Message.SuccessfulPayment != nil
}
