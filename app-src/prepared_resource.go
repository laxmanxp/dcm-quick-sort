package main

import "context"

// One owner (the UI thread) calls Take/Cancel/PollDone. The worker owns the
// resource until Take transfers it. A cancelled preparation is never adopted;
// its worker must finish disposal before PollDone reports completion.
type preparedOutcome[T any] struct {
	value T
	err   error
}
type preparedResource[T any] struct {
	ready    chan preparedOutcome[T]
	decision chan bool
	done     chan preparedOutcome[T]
	cancel   context.CancelFunc
	decided  bool
}

func newPreparedResource[T any](build func(context.Context) (T, error), dispose func(T) error) *preparedResource[T] {
	ctx, cancel := context.WithCancel(context.Background())
	p := &preparedResource[T]{ready: make(chan preparedOutcome[T], 1), decision: make(chan bool, 1), done: make(chan preparedOutcome[T], 1), cancel: cancel}
	go func() {
		value, err := build(ctx)
		p.ready <- preparedOutcome[T]{value, err}
		// Exactly one decision comes from the owner. Cancellation may precede build
		// completion; even then the returned partial resource must be disposed.
		if <-p.decision {
			p.done <- preparedOutcome[T]{}
		} else {
			cleanupErr := dispose(value)
			if cleanupErr == nil {
				var zero T
				value = zero
			}
			p.done <- preparedOutcome[T]{value, cleanupErr}
		}
	}()
	return p
}
func (p *preparedResource[T]) Cancel() {
	if p == nil || p.decided {
		return
	}
	p.decided = true
	p.cancel()
	p.decision <- false
}
func (p *preparedResource[T]) Take() (T, error, bool) {
	var zero T
	if p == nil || p.decided {
		return zero, nil, false
	}
	select {
	case out := <-p.ready:
		if out.err != nil {
			p.Cancel()
			return zero, out.err, true
		}
		p.decided = true
		p.decision <- true
		p.cancel()
		return out.value, nil, true
	default:
		return zero, nil, false
	}
}
func (p *preparedResource[T]) PollDone() (preparedOutcome[T], bool) {
	select {
	case out := <-p.done:
		return out, true
	default:
		return preparedOutcome[T]{}, false
	}
}
