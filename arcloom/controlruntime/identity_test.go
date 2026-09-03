// Package controlruntime runs target-independent, caller-scoped
// control lifecycles without owning target-specific reconciliation meaning.
package controlruntime

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewTargetIdentity(t *testing.T) {
	type args struct {
		kind string
		key  string
	}
	tests := []struct {
		name    string
		args    args
		want    TargetIdentity
		wantErr bool
	}{
		{name: "ordinary", args: args{kind: "plan", key: "github:owner/repository"}, want: TargetIdentity{kind: "plan", key: "github:owner/repository"}},
		{name: "embedded whitespace", args: args{kind: "plan target", key: "owner / repository"}, want: TargetIdentity{kind: "plan target", key: "owner / repository"}},
		{name: "case is preserved", args: args{kind: "Plan", key: "Owner/Repository"}, want: TargetIdentity{kind: "Plan", key: "Owner/Repository"}},
		{name: "composed unicode is preserved", args: args{kind: "plan", key: "caf\u00e9"}, want: TargetIdentity{kind: "plan", key: "caf\u00e9"}},
		{name: "decomposed unicode is preserved", args: args{kind: "plan", key: "cafe\u0301"}, want: TargetIdentity{kind: "plan", key: "cafe\u0301"}},
		{name: "empty kind", args: args{kind: "", key: "target"}, want: TargetIdentity{}, wantErr: true},
		{name: "empty key", args: args{kind: "plan", key: ""}, want: TargetIdentity{}, wantErr: true},
		{name: "whitespace only kind", args: args{kind: "\t\n", key: "target"}, want: TargetIdentity{}, wantErr: true},
		{name: "whitespace only key", args: args{kind: "plan", key: "\u3000"}, want: TargetIdentity{}, wantErr: true},
		{name: "leading unicode whitespace", args: args{kind: "\u2003plan", key: "target"}, want: TargetIdentity{}, wantErr: true},
		{name: "trailing unicode whitespace", args: args{kind: "plan", key: "target\u00a0"}, want: TargetIdentity{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTargetIdentity(tt.args.kind, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewTargetIdentity() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(TargetIdentity{})); diff != "" {
				t.Errorf("NewTargetIdentity() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTargetIdentity_Kind(t *testing.T) {
	type fields struct {
		kind string
		key  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero identity", fields: fields{}, want: ""},
		{name: "preserved kind", fields: fields{kind: "Plan Target", key: "ignored"}, want: "Plan Target"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TargetIdentity{
				kind: tt.fields.kind,
				key:  tt.fields.key,
			}
			if diff := cmp.Diff(tt.want, tr.Kind()); diff != "" {
				t.Errorf("TargetIdentity.Kind() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTargetIdentity_Key(t *testing.T) {
	type fields struct {
		kind string
		key  string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "zero identity", fields: fields{}, want: ""},
		{name: "preserved key", fields: fields{kind: "ignored", key: "Owner / Repository"}, want: "Owner / Repository"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TargetIdentity{
				kind: tt.fields.kind,
				key:  tt.fields.key,
			}
			if diff := cmp.Diff(tt.want, tr.Key()); diff != "" {
				t.Errorf("TargetIdentity.Key() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
