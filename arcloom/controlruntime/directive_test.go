package controlruntime

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestAwaitAnotherRequest(t *testing.T) {
	tests := []struct {
		name string
		want Directive
	}{
		{name: "constructed await", want: Directive{kind: AwaitRequest}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, AwaitAnotherRequest(), cmp.AllowUnexported(Directive{})); diff != "" {
				t.Errorf("AwaitAnotherRequest() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestImmediateReevaluation(t *testing.T) {
	tests := []struct {
		name string
		want Directive
	}{
		{name: "constructed immediate", want: Directive{kind: ReevaluateImmediately}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, ImmediateReevaluation(), cmp.AllowUnexported(Directive{})); diff != "" {
				t.Errorf("ImmediateReevaluation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDelayedReevaluation(t *testing.T) {
	type args struct {
		delay time.Duration
	}
	tests := []struct {
		name    string
		args    args
		want    Directive
		wantErr bool
	}{
		{name: "one nanosecond", args: args{delay: time.Nanosecond}, want: Directive{kind: ReevaluateAfterDelay, delay: time.Nanosecond}},
		{name: "one hour", args: args{delay: time.Hour}, want: Directive{kind: ReevaluateAfterDelay, delay: time.Hour}},
		{name: "zero", args: args{delay: 0}, want: Directive{}, wantErr: true},
		{name: "negative", args: args{delay: -time.Nanosecond}, want: Directive{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DelayedReevaluation(tt.args.delay)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DelayedReevaluation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Directive{})); diff != "" {
				t.Errorf("DelayedReevaluation() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDirective_Kind(t *testing.T) {
	type fields struct {
		kind  DirectiveKind
		delay time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   DirectiveKind
	}{
		{name: "zero directive", fields: fields{}, want: DirectiveKind(0)},
		{name: "await", fields: fields{kind: AwaitRequest}, want: AwaitRequest},
		{name: "immediate", fields: fields{kind: ReevaluateImmediately}, want: ReevaluateImmediately},
		{name: "delay", fields: fields{kind: ReevaluateAfterDelay, delay: time.Second}, want: ReevaluateAfterDelay},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Directive{
				kind:  tt.fields.kind,
				delay: tt.fields.delay,
			}
			if diff := cmp.Diff(tt.want, d.Kind()); diff != "" {
				t.Errorf("Directive.Kind() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDirective_Delay(t *testing.T) {
	type fields struct {
		kind  DirectiveKind
		delay time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Duration
		want1  bool
	}{
		{name: "zero directive", fields: fields{}, want: 0, want1: false},
		{name: "await", fields: fields{kind: AwaitRequest}, want: 0, want1: false},
		{name: "immediate", fields: fields{kind: ReevaluateImmediately}, want: 0, want1: false},
		{name: "positive delay", fields: fields{kind: ReevaluateAfterDelay, delay: 3 * time.Second}, want: 3 * time.Second, want1: true},
		{name: "unconstructed zero delay", fields: fields{kind: ReevaluateAfterDelay}, want: 0, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Directive{
				kind:  tt.fields.kind,
				delay: tt.fields.delay,
			}
			got, got1 := d.Delay()
			if !cmp.Equal(tt.want, got) {
				t.Errorf("Directive.Delay() got = %v, want %v\ndiff=%s", got, tt.want, cmp.Diff(tt.want, got))
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("Directive.Delay() presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
