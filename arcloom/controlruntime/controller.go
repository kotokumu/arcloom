package controlruntime

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidContext       = errors.New("invalid control context")
	ErrInvalidConcurrency   = errors.New("invalid concurrency bound")
	ErrInvalidAttempt       = errors.New("invalid attempt")
	ErrControllerNotStarted = errors.New("controller not started")
)

// Controller is one concurrent-safe, caller-scoped control lifecycle. It owns
// only disposable scheduling and Report-publication state. A Controller must
// not be copied after Start; its zero value is unusable.
type Controller[R any] struct {
	ctx      context.Context
	reports  chan Report[R]
	done     chan struct{}
	attempt  Attempt[R]
	requests chan requestSubmission
	maximum  int
	clock    schedulerClock
}

type requestSubmission struct {
	ctx      context.Context
	target   TargetIdentity
	accepted chan error
}

// Start validates configuration and starts one Controller lifecycle. The caller
// ends that lifecycle by ending ctx and may then use Wait to observe ctx.Err().
func Start[R any](ctx context.Context, maxConcurrent int, attempt Attempt[R]) (*Controller[R], error) {
	if ctx == nil {
		return nil, ErrInvalidContext
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxConcurrent <= 0 {
		return nil, ErrInvalidConcurrency
	}
	if attempt == nil {
		return nil, ErrInvalidAttempt
	}
	return start(ctx, maxConcurrent, attempt, systemSchedulerClock{})
}

func start[R any](ctx context.Context, maxConcurrent int, attempt Attempt[R], clock schedulerClock) (*Controller[R], error) {
	controller := &Controller[R]{
		ctx:      ctx,
		reports:  make(chan Report[R]),
		done:     make(chan struct{}),
		attempt:  attempt,
		requests: make(chan requestSubmission),
		maximum:  maxConcurrent,
		clock:    clock,
	}
	go controller.run()
	return controller, nil
}

// Request synchronously submits one identity-only wake-up. ctx bounds only this
// submission; after Request returns nil, ending ctx neither cancels nor supplies
// evidence to the accepted Attempt. If both contexts have ended before
// acceptance, the Controller lifecycle error takes precedence. Request is safe
// for concurrent callers and rejects the zero Controller and invalid identity.
func (c *Controller[R]) Request(ctx context.Context, target TargetIdentity) error {
	if c == nil || c.ctx == nil {
		return ErrControllerNotStarted
	}
	if ctx == nil {
		return ErrInvalidContext
	}
	if !target.isValid() {
		return ErrInvalidTargetIdentity
	}
	if err := c.ctx.Err(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	submission := requestSubmission{
		ctx:      ctx,
		target:   target,
		accepted: make(chan error, 1),
	}
	select {
	case c.requests <- submission:
		return <-submission.accepted
	case <-c.ctx.Done():
		return c.ctx.Err()
	case <-ctx.Done():
		if err := c.ctx.Err(); err != nil {
			return err
		}
		return ctx.Err()
	}
}

// Reports returns the one stable Report stream for this lifecycle. Normal
// operation publishes every returned Attempt outcome exactly once. At most one
// Report awaits publication; while it does, no new Attempt starts, although
// Requests remain accepted and Active Attempts may return. Publication commits
// a Completion before its Directive creates eligibility. Lifecycle cancellation
// may instead discard the unpublished Report and unapplied Directive, so a
// stopped consumer cannot prevent shutdown. Reports is safe for concurrent calls
// and returns nil for the zero Controller.
func (c *Controller[R]) Reports() <-chan Report[R] {
	if c == nil {
		return nil
	}
	return c.reports
}

// Wait returns after every Active Attempt has returned and Reports has closed.
// Pending and Delayed work and Report consumption add no shutdown wait. Repeated
// and concurrent calls return the same Controller lifecycle context error. The
// zero Controller returns ErrControllerNotStarted.
func (c *Controller[R]) Wait() error {
	if c == nil || c.ctx == nil {
		return ErrControllerNotStarted
	}
	<-c.done
	return c.ctx.Err()
}

func (c *Controller[R]) run() {
	defer close(c.done)
	defer close(c.reports)

	type targetState struct {
		active  bool
		pending bool
		delayed bool
		due     time.Time
	}
	type attemptReturn struct {
		target    TargetIdentity
		result    R
		directive Directive
		err       error
	}
	type prospectiveReport struct {
		report    Report[R]
		target    TargetIdentity
		directive Directive
		completed bool
	}

	states := map[TargetIdentity]*targetState{}
	returns := make(chan attemptReturn, c.maximum)
	active := 0
	stopping := false
	var prospective *prospectiveReport
	var delayTimer schedulerTimer
	var delayEvents <-chan time.Time
	var scheduledDue time.Time

	stopDelayTimer := func() {
		if delayTimer != nil {
			delayTimer.Stop()
		}
		delayEvents = nil
		scheduledDue = time.Time{}
	}
	beginStopping := func() {
		stopping = true
		prospective = nil
		for _, state := range states {
			state.pending = false
			state.delayed = false
		}
		stopDelayTimer()
	}
	refreshDelayTimer := func() {
		var earliest time.Time
		for _, state := range states {
			if state.delayed && (earliest.IsZero() || state.due.Before(earliest)) {
				earliest = state.due
			}
		}
		if earliest.IsZero() {
			stopDelayTimer()
			return
		}
		if delayEvents != nil && earliest.Equal(scheduledDue) {
			return
		}
		remaining := earliest.Sub(c.clock.Now())
		if remaining < 0 {
			remaining = 0
		}
		if delayTimer == nil {
			delayTimer = c.clock.NewTimer(remaining)
		} else {
			delayTimer.Reset(remaining)
		}
		delayEvents = delayTimer.Events()
		scheduledDue = earliest
	}
	defer stopDelayTimer()

	for {
		if !stopping && c.ctx.Err() != nil {
			beginStopping()
		}
		if stopping && active == 0 {
			return
		}
		if !stopping && prospective == nil {
			for active < c.maximum {
				var target TargetIdentity
				var state *targetState
				for candidate, candidateState := range states {
					if candidateState.pending && !candidateState.active {
						target = candidate
						state = candidateState
						break
					}
				}
				if state == nil {
					break
				}
				state.pending = false
				state.active = true
				active++
				go func(target TargetIdentity) {
					result, directive, err := c.attempt(c.ctx, target)
					returns <- attemptReturn{target: target, result: result, directive: directive, err: err}
				}(target)
			}
		}
		if !stopping {
			refreshDelayTimer()
		}

		var publication chan Report[R]
		var report Report[R]
		var returnedEvents <-chan attemptReturn
		var lifecycleDone <-chan struct{}
		var requestEvents <-chan requestSubmission
		if prospective != nil && !stopping {
			publication = c.reports
			report = prospective.report
		} else {
			returnedEvents = returns
		}
		if !stopping {
			lifecycleDone = c.ctx.Done()
			requestEvents = c.requests
		}

		select {
		case <-lifecycleDone:
			beginStopping()
		case submission := <-requestEvents:
			if err := c.ctx.Err(); err != nil {
				submission.accepted <- err
				beginStopping()
				continue
			}
			if err := submission.ctx.Err(); err != nil {
				submission.accepted <- err
				continue
			}
			state := states[submission.target]
			if state == nil {
				state = &targetState{}
				states[submission.target] = state
			}
			state.delayed = false
			state.pending = true
			submission.accepted <- nil
		case returned := <-returnedEvents:
			active--
			state := states[returned.target]
			if state != nil {
				state.active = false
			}
			if c.ctx.Err() != nil {
				beginStopping()
				continue
			}
			if stopping {
				continue
			}
			switch {
			case returned.err != nil:
				prospective = &prospectiveReport{
					report: newFailureReport[R](returned.target, newFailure(TargetAttemptFailed, returned.err)),
					target: returned.target,
				}
			case !returned.directive.isValid():
				prospective = &prospectiveReport{
					report: newFailureReport[R](returned.target, newFailure(ControlDirectiveRejected, ErrInvalidDirective)),
					target: returned.target,
				}
			default:
				prospective = &prospectiveReport{
					report:    newCompletionReport(returned.target, returned.result, returned.directive),
					target:    returned.target,
					directive: returned.directive,
					completed: true,
				}
			}
		case publication <- report:
			state := states[prospective.target]
			if prospective.completed && state != nil && !state.pending {
				switch prospective.directive.Kind() {
				case ReevaluateImmediately:
					state.pending = true
				case ReevaluateAfterDelay:
					delay, _ := prospective.directive.Delay()
					state.delayed = true
					state.due = c.clock.Now().Add(delay)
				}
			}
			if state != nil && !state.active && !state.pending && !state.delayed {
				delete(states, prospective.target)
			}
			prospective = nil
		case <-delayEvents:
			delayEvents = nil
			scheduledDue = time.Time{}
			now := c.clock.Now()
			for _, state := range states {
				if state.delayed && !state.due.After(now) {
					state.delayed = false
					state.pending = true
				}
			}
		}
	}
}
