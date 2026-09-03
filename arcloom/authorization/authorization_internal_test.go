package authorization

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewPolicy(t *testing.T) {
	type args struct {
		rules []Rule[int]
	}
	tests := []struct {
		name    string
		args    args
		want    Policy[int]
		wantErr bool
	}{
		{
			name: "one rule",
			args: args{rules: []Rule[int]{func(context.Context, int) (RuleConclusion, error) {
				return Permit, nil
			}}},
			want: Policy[int]{valid: true, rules: []Rule[int]{func(context.Context, int) (RuleConclusion, error) {
				return Permit, nil
			}}},
		},
		{name: "empty", args: args{rules: nil}, want: Policy[int]{}, wantErr: true},
		{name: "nil rule", args: args{rules: []Rule[int]{nil}}, want: Policy[int]{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPolicy(tt.args.rules...)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("NewPolicy() error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got,
				cmp.AllowUnexported(Policy[int]{}),
				cmp.Comparer(func(left, right Rule[int]) bool {
					return (left == nil) == (right == nil)
				}),
			); diff != "" {
				t.Errorf("NewPolicy() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPolicy_Evaluate(t *testing.T) {
	type args struct {
		ctx     context.Context
		subject int
	}
	tests := []struct {
		name    string
		fields  Policy[int]
		args    args
		want    Evaluation[int]
		wantErr bool
	}{
		{
			name: "permit",
			fields: Policy[int]{valid: true, rules: []Rule[int]{func(context.Context, int) (RuleConclusion, error) {
				return Permit, nil
			}}},
			args: args{ctx: context.Background(), subject: 7},
			want: Evaluation[int]{subject: 7, decision: Authorized, valid: true},
		},
		{
			name: "deny overrides",
			fields: Policy[int]{valid: true, rules: []Rule[int]{
				func(context.Context, int) (RuleConclusion, error) { return Unknown, nil },
				func(context.Context, int) (RuleConclusion, error) { return Deny, nil },
			}},
			args: args{ctx: context.Background(), subject: 8},
			want: Evaluation[int]{subject: 8, decision: Denied, valid: true},
		},
		{
			name: "unknown",
			fields: Policy[int]{valid: true, rules: []Rule[int]{func(context.Context, int) (RuleConclusion, error) {
				return Unknown, nil
			}}},
			args: args{ctx: context.Background(), subject: 9},
			want: Evaluation[int]{subject: 9, decision: Undecidable, valid: true},
		},
		{
			name:   "invalid policy",
			fields: Policy[int]{},
			args:   args{ctx: context.Background(), subject: 10},
			want:   Evaluation[int]{subject: 10, decision: Undecidable, valid: true},
		},
		{
			name: "cancelled",
			fields: Policy[int]{valid: true, rules: []Rule[int]{func(ctx context.Context, _ int) (RuleConclusion, error) {
				<-ctx.Done()
				return Permit, nil
			}}},
			args: args{ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(), subject: 11},
			want:    Evaluation[int]{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.fields
			got, err := p.Evaluate(tt.args.ctx, tt.args.subject)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("Policy.Evaluate() error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Evaluation[int]{})); diff != "" {
				t.Errorf("Policy.Evaluate() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvaluation_Subject(t *testing.T) {
	type fields struct {
		subject  int
		decision Decision
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{name: "bound subject", fields: fields{subject: 42, decision: Authorized, valid: true}, want: 42},
		{name: "zero evaluation", fields: fields{}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (Evaluation[int]{subject: tt.fields.subject, decision: tt.fields.decision, valid: tt.fields.valid}).Subject()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Evaluation.Subject() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestEvaluation_Decision(t *testing.T) {
	type fields struct {
		subject  int
		decision Decision
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   Decision
	}{
		{name: "authorized", fields: fields{decision: Authorized, valid: true}, want: Authorized},
		{name: "denied", fields: fields{decision: Denied, valid: true}, want: Denied},
		{name: "undecidable", fields: fields{decision: Undecidable, valid: true}, want: Undecidable},
		{name: "zero evaluation", fields: fields{}, want: Undecidable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (Evaluation[int]{subject: tt.fields.subject, decision: tt.fields.decision, valid: tt.fields.valid}).Decision()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Evaluation.Decision() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
