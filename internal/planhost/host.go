// Package planhost implements one disposable reference Host invocation. Plan
// decisions remain in the Plan Controller; this package owns only operations.
package planhost

import (
	"context"
	"errors"
	"sync"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan/assessmentdelivery"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
)

var ErrHostNotStarted = errors.New("plan host not started")

// Host coordinates intake, sole Report consumption, and fail-stop lifecycle.
// Do not copy after Start. Methods are concurrent-safe; zero/nil is not started.
type Host struct {
	parent     context.Context
	ctx        context.Context
	cancel     context.CancelFunc
	controller *controlruntime.Controller[planattempt.AttemptResult]
	identity   controlruntime.TargetIdentity
	recipient  assessmentdelivery.Recipient
	reports    chan controlruntime.Report[planattempt.AttemptResult]
	done       chan struct{}
	mu         sync.Mutex
	terminal   error
}

// Start validates context, binding, then recipient before starting any work.
// It starts no Attempt until Trigger. The binding alone supplies target identity.
func Start(ctx context.Context, binding planattempt.PlanAttemptBinding, recipient assessmentdelivery.Recipient) (*Host, error) {
	if ctx == nil {
		return nil, controlruntime.ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if binding.Identity().Kind() == "" || binding.Identity().Key() == "" || binding.Attempt() == nil {
		return nil, planattempt.ErrInvalidPlanAttemptBinding
	}
	if recipient == nil {
		return nil, assessmentdelivery.ErrInvalidRecipient
	}
	child, cancel := context.WithCancel(ctx)
	controller, err := controlruntime.Start(child, 1, binding.Attempt())
	if err != nil {
		cancel()
		return nil, err
	}
	host := &Host{parent: ctx, ctx: child, cancel: cancel, controller: controller, identity: binding.Identity(), recipient: recipient, reports: make(chan controlruntime.Report[planattempt.AttemptResult], 1), done: make(chan struct{})}
	go host.run()
	return host, nil
}

// Trigger submits only the bound identity. ctx bounds acceptance, not already
// accepted work. A committed terminal error wins over submission cancellation.
func (h *Host) Trigger(ctx context.Context) error {
	if h == nil || h.parent == nil {
		return ErrHostNotStarted
	}
	if ctx == nil {
		return controlruntime.ErrInvalidContext
	}
	if err := h.outcome(); err != nil {
		return err
	}
	err := h.controller.Request(ctx, h.identity)
	if terminal := h.outcome(); terminal != nil {
		return terminal
	}
	return err
}

// Reports is the stable, sole processed-evidence stream in Controller order.
// An assessed Report is exposed only after handoff success. One queued Report
// and one being handled bound evidence state. Cancellation can discard pending
// evidence even after delivery; published evidence remains drainable on close.
func (h *Host) Reports() <-chan controlruntime.Report[planattempt.AttemptResult] {
	if h == nil {
		return nil
	}
	return h.reports
}

// Wait awaits active cooperative boundaries and Controller exit, not a reader.
// Concurrent and repeated calls return the same terminal error.
func (h *Host) Wait() error {
	if h == nil || h.parent == nil {
		return ErrHostNotStarted
	}
	<-h.done
	return h.outcome()
}

func (h *Host) outcome() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.terminal != nil {
		return h.terminal
	}
	return h.parent.Err()
}

func (h *Host) stop(err error) {
	h.mu.Lock()
	if h.terminal == nil {
		h.terminal = h.parent.Err()
		if h.terminal == nil {
			h.terminal = err
		}
	}
	h.mu.Unlock()
	h.cancel()
}

func (h *Host) run() {
	defer close(h.done)
	defer close(h.reports)
	defer func() { h.cancel(); _ = h.controller.Wait() }()
	for {
		if err := h.ctx.Err(); err != nil {
			h.stop(err)
			return
		}
		select {
		case <-h.ctx.Done():
			h.stop(h.ctx.Err())
			return
		case report, ok := <-h.controller.Reports():
			if !ok {
				h.stop(h.controller.Wait())
				return
			}
			// Qualification belongs to Plan Assessment Delivery, never to this pump.
			if err := assessmentdelivery.Deliver(h.ctx, report, h.recipient); err != nil {
				h.stop(err)
				return
			}
			if err := h.ctx.Err(); err != nil {
				h.stop(err)
				return
			}
			select {
			case h.reports <- report:
			case <-h.ctx.Done():
				h.stop(h.ctx.Err())
				return
			}
		}
	}
}
