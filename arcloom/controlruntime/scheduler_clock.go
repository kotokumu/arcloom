package controlruntime

import "time"

type schedulerClock interface {
	Now() time.Time
	NewTimer(time.Duration) schedulerTimer
}

type schedulerTimer interface {
	Events() <-chan time.Time
	Reset(time.Duration)
	Stop()
}

type systemSchedulerClock struct{}

func (systemSchedulerClock) Now() time.Time {
	return time.Now()
}

func (systemSchedulerClock) NewTimer(delay time.Duration) schedulerTimer {
	return &systemSchedulerTimer{timer: time.NewTimer(delay)}
}

type systemSchedulerTimer struct {
	timer *time.Timer
}

func (t *systemSchedulerTimer) Events() <-chan time.Time {
	return t.timer.C
}

func (t *systemSchedulerTimer) Reset(delay time.Duration) {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
	t.timer.Reset(delay)
}

func (t *systemSchedulerTimer) Stop() {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
}
