package controlruntime

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func Test_newFailure(t *testing.T) {
	type args struct {
		kind  FailureKind
		cause error
	}
	tests := []struct {
		name string
		args args
		want Failure
	}{
		{name: "target attempt", args: args{kind: TargetAttemptFailed, cause: ErrInvalidTargetIdentity}, want: Failure{kind: TargetAttemptFailed, cause: ErrInvalidTargetIdentity}},
		{name: "directive rejection", args: args{kind: ControlDirectiveRejected, cause: ErrInvalidDirective}, want: Failure{kind: ControlDirectiveRejected, cause: ErrInvalidDirective}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, newFailure(tt.args.kind, tt.args.cause), cmp.AllowUnexported(Failure{}), cmpopts.EquateErrors()); diff != "" {
				t.Errorf("newFailure() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailure_Kind(t *testing.T) {
	type fields struct {
		kind  FailureKind
		cause error
	}
	tests := []struct {
		name   string
		fields fields
		want   FailureKind
	}{
		{name: "zero failure", fields: fields{}, want: FailureKind(0)},
		{name: "target attempt", fields: fields{kind: TargetAttemptFailed, cause: ErrInvalidTargetIdentity}, want: TargetAttemptFailed},
		{name: "directive rejection", fields: fields{kind: ControlDirectiveRejected, cause: ErrInvalidDirective}, want: ControlDirectiveRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Failure{
				kind:  tt.fields.kind,
				cause: tt.fields.cause,
			}
			if diff := cmp.Diff(tt.want, f.Kind()); diff != "" {
				t.Errorf("Failure.Kind() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailure_Error(t *testing.T) {
	type fields struct {
		kind  FailureKind
		cause error
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero failure", fields: fields{}, want: "invalid attempt failure"},
		{name: "target attempt", fields: fields{kind: TargetAttemptFailed, cause: ErrInvalidTargetIdentity}, want: "target attempt failed: invalid target identity"},
		{name: "directive rejection", fields: fields{kind: ControlDirectiveRejected, cause: ErrInvalidDirective}, want: "control directive rejected: invalid control directive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Failure{
				kind:  tt.fields.kind,
				cause: tt.fields.cause,
			}
			if diff := cmp.Diff(tt.want, f.Error()); diff != "" {
				t.Errorf("Failure.Error() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailure_Unwrap(t *testing.T) {
	type fields struct {
		kind  FailureKind
		cause error
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{name: "zero failure", fields: fields{}, wantErr: false},
		{name: "target cause", fields: fields{kind: TargetAttemptFailed, cause: ErrInvalidTargetIdentity}, wantErr: true},
		{name: "directive cause", fields: fields{kind: ControlDirectiveRejected, cause: ErrInvalidDirective}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Failure{
				kind:  tt.fields.kind,
				cause: tt.fields.cause,
			}
			if err := f.Unwrap(); (err != nil) != tt.wantErr {
				t.Errorf("Failure.Unwrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
