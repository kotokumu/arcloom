package controlruntime

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type controlledSchedulerClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*controlledSchedulerTimer
	armed  chan struct{}
}

func (c *controlledSchedulerClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *controlledSchedulerClock) NewTimer(delay time.Duration) schedulerTimer {
	c.mu.Lock()
	defer c.mu.Unlock()
	timer := &controlledSchedulerTimer{
		clock:  c,
		due:    c.now.Add(delay),
		events: make(chan time.Time, 1),
		active: true,
	}
	c.timers = append(c.timers, timer)
	if c.armed != nil {
		c.armed <- struct{}{}
	}
	return timer
}

func (c *controlledSchedulerClock) Advance(delay time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(delay)
	now := c.now
	for _, timer := range c.timers {
		if timer.active && !timer.due.After(now) {
			timer.active = false
			timer.events <- now
		}
	}
	c.mu.Unlock()
}

type controlledSchedulerTimer struct {
	clock  *controlledSchedulerClock
	due    time.Time
	events chan time.Time
	active bool
}

func (t *controlledSchedulerTimer) Events() <-chan time.Time {
	return t.events
}

func (t *controlledSchedulerTimer) Reset(delay time.Duration) {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.due = t.clock.now.Add(delay)
	t.active = true
}

func (t *controlledSchedulerTimer) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.active = false
}

func TestControllerCommitsDirectivesOnlyAfterCompletionPublication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan TargetIdentity, 2)
	attempt := Attempt[string](func(_ context.Context, target TargetIdentity) (string, Directive, error) {
		started <- target
		return target.Key(), ImmediateReevaluation(), nil
	})
	controller, err := Start(ctx, 1, attempt)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	first, _ := NewTargetIdentity("test", "first")
	second, _ := NewTargetIdentity("test", "second")
	if err := controller.Request(context.Background(), first); err != nil {
		t.Fatalf("first Request() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first Attempt did not start")
	}
	if err := controller.Request(context.Background(), second); err != nil {
		t.Fatalf("second Request() error = %v", err)
	}

	cancel()
	if err := controller.Wait(); !errors.Is(err, context.Canceled) {
		t.Errorf("Wait() error = %v, want context.Canceled", err)
	}
	if got := len(started); got != 0 {
		t.Errorf("Attempts started before Completion publication = %d, want 0", got)
	}
	if _, ok := <-controller.Reports(); ok {
		t.Error("Reports remained open after Wait")
	}
}

func TestControllerFollowsAwaitAndImmediateDirectives(t *testing.T) {
	tests := []struct {
		name      string
		directive Directive
		wantCalls int64
	}{
		{name: "await", directive: AwaitAnotherRequest(), wantCalls: 1},
		{name: "immediate", directive: ImmediateReevaluation(), wantCalls: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			var calls atomic.Int64
			started := make(chan int64, 2)
			attempt := Attempt[int64](func(context.Context, TargetIdentity) (int64, Directive, error) {
				call := calls.Add(1)
				started <- call
				if call == 1 {
					return call, tt.directive, nil
				}
				return call, AwaitAnotherRequest(), nil
			})
			controller, err := Start(ctx, 1, attempt)
			if err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			target, _ := NewTargetIdentity("test", tt.name)
			if err := controller.Request(context.Background(), target); err != nil {
				t.Fatalf("Request() error = %v", err)
			}
			<-started
			<-controller.Reports()
			if tt.wantCalls == 2 {
				select {
				case <-started:
				case <-time.After(time.Second):
					t.Fatal("Immediate directive did not start another Attempt")
				}
				<-controller.Reports()
			}
			cancel()
			_ = controller.Wait()
			if got := calls.Load(); got != tt.wantCalls {
				t.Errorf("Attempt calls = %d, want %d", got, tt.wantCalls)
			}
		})
	}
}

func TestControllerDelayedDirectiveAndEarlyRequestDoNotDuplicateEligibility(t *testing.T) {
	tests := []struct {
		name         string
		requestEarly bool
	}{
		{name: "delay expires"},
		{name: "request arrives early", requestEarly: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			clock := &controlledSchedulerClock{now: time.Unix(100, 0), armed: make(chan struct{}, 1)}
			started := make(chan int, 3)
			call := 0
			delayed, _ := DelayedReevaluation(time.Minute)
			attempt := Attempt[int](func(context.Context, TargetIdentity) (int, Directive, error) {
				call++
				started <- call
				if call == 1 {
					return call, delayed, nil
				}
				return call, AwaitAnotherRequest(), nil
			})
			controller, err := start(ctx, 1, attempt, clock)
			if err != nil {
				t.Fatalf("start() error = %v", err)
			}
			target, _ := NewTargetIdentity("test", tt.name)
			if err := controller.Request(context.Background(), target); err != nil {
				t.Fatalf("Request() error = %v", err)
			}
			<-started
			<-controller.Reports()
			select {
			case <-clock.armed:
			case <-time.After(time.Second):
				t.Fatal("delayed eligibility was not armed")
			}

			if tt.requestEarly {
				if err := controller.Request(context.Background(), target); err != nil {
					t.Fatalf("early Request() error = %v", err)
				}
			} else {
				clock.Advance(time.Minute)
			}
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("second Attempt did not start")
			}
			<-controller.Reports()

			clock.Advance(time.Hour)
			cancel()
			_ = controller.Wait()
			if call != 2 {
				t.Errorf("Attempt calls = %d, want 2", call)
			}
		})
	}
}
