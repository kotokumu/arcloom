package plansnapshot_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

func TestObservationFailure(t *testing.T) {
	failure := plansnapshot.UnavailableObservationFailure()
	if diff := cmp.Diff(plansnapshot.ObservationUnavailable, failure.Code()); diff != "" {
		t.Errorf("failure code mismatch (-want +got):\n%s", diff)
	}
	var zero plansnapshot.ObservationFailure
	if diff := cmp.Diff(plansnapshot.ObservationFailureCode(0), zero.Code()); diff != "" {
		t.Errorf("zero failure code mismatch (-want +got):\n%s", diff)
	}
}

func TestObserve(t *testing.T) {
	success := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Unknown, nil))))
	ctxCanceled, cancel := context.WithCancel(context.Background())
	cancel()
	type args struct {
		ctx      context.Context
		observer plansnapshot.Observer
	}
	tests := []struct {
		name    string
		args    args
		want    plansnapshot.Snapshot
		wantErr bool
	}{
		{name: "nil context", args: args{observer: func(context.Context) (plansnapshot.Snapshot, error) { return success, nil }}, want: plansnapshot.Snapshot{}, wantErr: true},
		{name: "entry cancellation", args: args{ctx: ctxCanceled, observer: func(context.Context) (plansnapshot.Snapshot, error) { return success, nil }}, want: plansnapshot.Snapshot{}, wantErr: true},
		{name: "nil observer", args: args{ctx: context.Background()}, want: plansnapshot.Snapshot{}, wantErr: true},
		{name: "successful snapshot", args: args{ctx: context.Background(), observer: func(context.Context) (plansnapshot.Snapshot, error) { return success, nil }}, want: success},
		{name: "stable unavailable failure", args: args{ctx: context.Background(), observer: func(context.Context) (plansnapshot.Snapshot, error) {
			return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
		}}, want: plansnapshot.Snapshot{}, wantErr: true},
		{name: "valid snapshot with stable error", args: args{ctx: context.Background(), observer: func(context.Context) (plansnapshot.Snapshot, error) {
			return success, plansnapshot.UnavailableObservationFailure()
		}}, want: plansnapshot.Snapshot{}, wantErr: true},
		{name: "valid snapshot with unknown error", args: args{ctx: context.Background(), observer: func(context.Context) (plansnapshot.Snapshot, error) {
			return success, errors.New("provider detail")
		}}, want: plansnapshot.Snapshot{}, wantErr: true},
		{name: "zero snapshot result", args: args{ctx: context.Background(), observer: func(context.Context) (plansnapshot.Snapshot, error) { return plansnapshot.Snapshot{}, nil }}, want: plansnapshot.Snapshot{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plansnapshot.Observe(tt.args.ctx, tt.args.observer)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			gotProgress, gotHasProgress := got.Progress()
			wantProgress, wantHasProgress := tt.want.Progress()
			if diff := cmp.Diff(wantHasProgress, gotHasProgress); diff != "" {
				t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantProgress.OverallState(), gotProgress.OverallState()); diff != "" {
				t.Errorf("overall state mismatch (-want +got):\n%s", diff)
			}
			wantTasks := wantProgress.Tasks()
			gotTasks := gotProgress.Tasks()
			wantNames := make([]string, len(wantTasks))
			gotNames := make([]string, len(gotTasks))
			wantStates := make([]plansnapshot.ProgressState, len(wantTasks))
			gotStates := make([]plansnapshot.ProgressState, len(gotTasks))
			for i := range wantTasks {
				wantNames[i] = wantTasks[i].Name()
				wantStates[i] = wantTasks[i].State()
			}
			for i := range gotTasks {
				gotNames[i] = gotTasks[i].Name()
				gotStates[i] = gotTasks[i].State()
			}
			if diff := cmp.Diff(wantNames, gotNames); diff != "" {
				t.Errorf("task names mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantStates, gotStates); diff != "" {
				t.Errorf("task states mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantProgress.MembershipComplete(), gotProgress.MembershipComplete()); diff != "" {
				t.Errorf("completeness mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObserveStableFailureCode(t *testing.T) {
	got, err := plansnapshot.Observe(context.Background(), func(context.Context) (plansnapshot.Snapshot, error) {
		return plansnapshot.Snapshot{}, plansnapshot.UnavailableObservationFailure()
	})
	var failure plansnapshot.ObservationFailure
	if diff := cmp.Diff(true, errors.As(err, &failure)); diff != "" {
		t.Errorf("stable failure type mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plansnapshot.ObservationUnavailable, failure.Code()); diff != "" {
		t.Errorf("stable failure code mismatch (-want +got):\n%s", diff)
	}
	_, hasProgress := got.Progress()
	if diff := cmp.Diff(false, hasProgress); diff != "" {
		t.Errorf("contradictory result snapshot mismatch (-want +got):\n%s", diff)
	}
}

func TestObserveEntryCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := plansnapshot.Observe(ctx, func(context.Context) (plansnapshot.Snapshot, error) {
		return plansnapshot.Snapshot{}, nil
	})
	if diff := cmp.Diff(true, errors.Is(err, context.Canceled)); diff != "" {
		t.Errorf("cancellation mismatch (-want +got):\n%s", diff)
	}
	_, hasProgress := got.Progress()
	if diff := cmp.Diff(false, hasProgress); diff != "" {
		t.Errorf("cancelled snapshot mismatch (-want +got):\n%s", diff)
	}
}

func TestObserveRejectsUnknownErrorsAndInvalidFailures(t *testing.T) {
	unknown := errors.New("provider detail")
	type args struct{ returned error }
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "unknown error", args: args{returned: unknown}, want: true},
		{name: "zero observation failure", args: args{returned: plansnapshot.ObservationFailure{}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := plansnapshot.Observe(context.Background(), func(context.Context) (plansnapshot.Snapshot, error) { return plansnapshot.Snapshot{}, tt.args.returned })
			if diff := cmp.Diff(tt.want, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, errors.Is(err, unknown)); diff != "" {
				t.Errorf("provider error leakage mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObserveCancellationInFlight(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	result := make(chan struct {
		snapshot plansnapshot.Snapshot
		err      error
	})
	go func() {
		snapshot, err := plansnapshot.Observe(ctx, func(ctx context.Context) (plansnapshot.Snapshot, error) {
			close(started)
			<-ctx.Done()
			return plansnapshot.Snapshot{}, ctx.Err()
		})
		result <- struct {
			snapshot plansnapshot.Snapshot
			err      error
		}{snapshot: snapshot, err: err}
	}()
	<-started
	cancel()
	got := <-result
	if diff := cmp.Diff(true, got.err == context.Canceled); diff != "" {
		t.Errorf("in-flight cancellation mismatch (-want +got):\n%s", diff)
	}
	_, hasProgress := got.snapshot.Progress()
	if diff := cmp.Diff(false, hasProgress); diff != "" {
		t.Errorf("cancelled snapshot presence mismatch (-want +got):\n%s", diff)
	}
}

func TestObserveConcurrentDistinctResultsAndCancellationIsolation(t *testing.T) {
	ctxCanceled, cancel := context.WithCancel(context.Background())
	cancelStarted := make(chan struct{})
	successStarted := make(chan struct{})
	release := make(chan struct{})
	success := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Open, nil))))
	type result struct {
		snapshot     plansnapshot.Snapshot
		err          error
		wantCanceled bool
		wantNil      bool
		wantState    plansnapshot.ProgressState
		wantHas      bool
		wantComplete bool
	}
	results := make(chan result, 2)
	go func() {
		snapshot, err := plansnapshot.Observe(ctxCanceled, func(ctx context.Context) (plansnapshot.Snapshot, error) {
			close(cancelStarted)
			<-ctx.Done()
			return plansnapshot.Snapshot{}, context.Canceled
		})
		results <- result{snapshot: snapshot, err: err, wantCanceled: true, wantNil: false}
	}()
	go func() {
		snapshot, err := plansnapshot.Observe(context.Background(), func(context.Context) (plansnapshot.Snapshot, error) {
			close(successStarted)
			<-release
			return success, nil
		})
		results <- result{snapshot: snapshot, err: err, wantCanceled: false, wantNil: true, wantState: plansnapshot.Open, wantHas: true, wantComplete: true}
	}()
	<-cancelStarted
	<-successStarted
	cancel()
	close(release)
	first := <-results
	second := <-results
	for _, got := range []result{first, second} {
		if diff := cmp.Diff(got.wantCanceled, got.err == context.Canceled); diff != "" {
			t.Errorf("concurrent cancellation identity mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(got.wantNil, got.err == nil); diff != "" {
			t.Errorf("concurrent nil error mismatch (-want +got):\n%s", diff)
		}
		gotProgress, gotHasProgress := got.snapshot.Progress()
		if diff := cmp.Diff(got.wantHas, gotHasProgress); diff != "" {
			t.Errorf("progress presence mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(got.wantState, gotProgress.OverallState()); diff != "" {
			t.Errorf("progress state mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(got.wantComplete, gotProgress.MembershipComplete()); diff != "" {
			t.Errorf("progress completeness mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestObserveConcurrentSuccessfulDistinctSnapshots(t *testing.T) {
	firstSnapshot := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Open, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("first", plansnapshot.Open))}))))
	secondSnapshot := must(plansnapshot.WithoutCurrent(must(plansnapshot.CompleteProgress(plansnapshot.Closed, []plansnapshot.TaskProgress{must(plansnapshot.NewTaskProgress("second", plansnapshot.Closed))}))))
	firstStarted := make(chan struct{})
	secondStarted := make(chan struct{})
	release := make(chan struct{})
	type result struct {
		snapshot plansnapshot.Snapshot
		err      error
		want     plansnapshot.Snapshot
	}
	results := make(chan result, 2)
	go func() {
		snapshot, err := plansnapshot.Observe(context.Background(), func(context.Context) (plansnapshot.Snapshot, error) {
			close(firstStarted)
			<-release
			return firstSnapshot, nil
		})
		results <- result{snapshot: snapshot, err: err, want: firstSnapshot}
	}()
	go func() {
		snapshot, err := plansnapshot.Observe(context.Background(), func(context.Context) (plansnapshot.Snapshot, error) {
			close(secondStarted)
			<-release
			return secondSnapshot, nil
		})
		results <- result{snapshot: snapshot, err: err, want: secondSnapshot}
	}()
	<-firstStarted
	<-secondStarted
	close(release)
	first := <-results
	second := <-results
	for _, got := range []result{first, second} {
		if diff := cmp.Diff(nil, got.err); diff != "" {
			t.Errorf("concurrent error mismatch (-want +got):\n%s", diff)
		}
		gotProgress, gotHasProgress := got.snapshot.Progress()
		wantProgress, wantHasProgress := got.want.Progress()
		if diff := cmp.Diff(wantHasProgress, gotHasProgress); diff != "" {
			t.Errorf("concurrent progress presence mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantProgress.OverallState(), gotProgress.OverallState()); diff != "" {
			t.Errorf("concurrent overall state mismatch (-want +got):\n%s", diff)
		}
		wantTasks := wantProgress.Tasks()
		gotTasks := gotProgress.Tasks()
		wantNames := make([]string, len(wantTasks))
		gotNames := make([]string, len(gotTasks))
		wantStates := make([]plansnapshot.ProgressState, len(wantTasks))
		gotStates := make([]plansnapshot.ProgressState, len(gotTasks))
		for i := range wantTasks {
			wantNames[i] = wantTasks[i].Name()
			wantStates[i] = wantTasks[i].State()
		}
		for i := range gotTasks {
			gotNames[i] = gotTasks[i].Name()
			gotStates[i] = gotTasks[i].State()
		}
		if diff := cmp.Diff(wantNames, gotNames); diff != "" {
			t.Errorf("concurrent task names mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantStates, gotStates); diff != "" {
			t.Errorf("concurrent task states mismatch (-want +got):\n%s", diff)
		}
		if diff := cmp.Diff(wantProgress.MembershipComplete(), gotProgress.MembershipComplete()); diff != "" {
			t.Errorf("concurrent completeness mismatch (-want +got):\n%s", diff)
		}
	}
}
