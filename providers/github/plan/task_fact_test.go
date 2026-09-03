package githubplan

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_sameProgressFacts(t *testing.T) {
	type args struct {
		left  taskItemFact
		right taskItemFact
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "same progress facts", args: args{left: taskItemFact{title: "A", state: nativeStateOpen}, right: taskItemFact{title: "A", state: nativeStateOpen}}, want: true},
		{name: "different title", args: args{left: taskItemFact{title: "A", state: nativeStateOpen}, right: taskItemFact{title: "B", state: nativeStateOpen}}, want: false},
		{name: "different state", args: args{left: taskItemFact{title: "A", state: nativeStateOpen}, right: taskItemFact{title: "A", state: nativeStateClosed}}, want: false},
		{name: "different member role", args: args{left: taskItemFact{title: "A", state: nativeStateOpen}, right: taskItemFact{title: "A", state: nativeStateOpen, pullRequest: true}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, sameProgressFacts(tt.args.left, tt.args.right)); diff != "" {
				t.Errorf("sameProgressFacts() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
