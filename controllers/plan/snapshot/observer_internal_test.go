package plansnapshot

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
)

func TestUnavailableObservationFailure(t *testing.T) {
	tests := []struct {
		name string
		want ObservationFailure
	}{
		{name: "stable unavailable", want: ObservationFailure{code: ObservationUnavailable}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, UnavailableObservationFailure(), cmp.AllowUnexported(ObservationFailure{})); diff != "" {
				t.Errorf("UnavailableObservationFailure mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObservationFailure_Error(t *testing.T) {
	type fields struct {
		code ObservationFailureCode
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "unavailable", fields: fields{code: ObservationUnavailable}, want: "observation unavailable"},
		{name: "zero", fields: fields{}, want: "invalid observation failure"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := ObservationFailure{code: tt.fields.code}
			if diff := cmp.Diff(tt.want, f.Error()); diff != "" {
				t.Errorf("ObservationFailure.Error() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObservationFailure_Code(t *testing.T) {
	type fields struct{ code ObservationFailureCode }
	tests := []struct {
		name   string
		fields fields
		want   ObservationFailureCode
	}{
		{name: "unavailable", fields: fields{code: ObservationUnavailable}, want: ObservationUnavailable},
		{name: "zero", fields: fields{}, want: ObservationFailureCode(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, (ObservationFailure{code: tt.fields.code}).Code()); diff != "" {
				t.Errorf("ObservationFailure.Code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestObserve(t *testing.T) {
	type args struct {
		ctx      context.Context
		observer Observer
	}
	tests := []struct {
		name    string
		args    args
		want    Snapshot
		wantErr bool
	}{
		{name: "nil context", args: args{ctx: nil, observer: func(context.Context) (Snapshot, error) { return Snapshot{}, nil }}, want: Snapshot{}, wantErr: true},
		{name: "valid snapshot", args: args{ctx: context.Background(), observer: func(context.Context) (Snapshot, error) { return Snapshot{valid: true}, nil }}, want: Snapshot{valid: true}},
		{name: "stable failure", args: args{ctx: context.Background(), observer: func(context.Context) (Snapshot, error) { return Snapshot{}, UnavailableObservationFailure() }}, want: Snapshot{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Observe(tt.args.ctx, tt.args.observer)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("Observe() error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Snapshot{}, ProgressEvidence{}, TaskProgress{}, plan.Plan{}, plan.Goal{}, plan.AcceptanceCondition{}, plan.Task{}, plan.TargetDate{}), cmp.Comparer(func(left, right plan.Plan) bool { return (!left.IsValid() && !right.IsValid()) || left.Equal(right) })); diff != "" {
				t.Errorf("Observe() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_validObservationFailure(t *testing.T) {
	type args struct{ err error }
	tests := []struct {
		name  string
		args  args
		want  ObservationFailure
		want1 bool
	}{
		{name: "stable value", args: args{err: UnavailableObservationFailure()}, want: ObservationFailure{code: ObservationUnavailable}, want1: true},
		{name: "unknown error", args: args{err: errors.New("provider detail")}, want: ObservationFailure{}, want1: false},
		{name: "zero failure", args: args{err: ObservationFailure{}}, want: ObservationFailure{}, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := validObservationFailure(tt.args.err)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(ObservationFailure{})); diff != "" {
				t.Errorf("validObservationFailure value mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("validObservationFailure presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
