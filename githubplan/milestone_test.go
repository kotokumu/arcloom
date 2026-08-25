package githubplan

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_decodeNativeState(t *testing.T) {
	type args struct {
		raw json.RawMessage
	}
	tests := []struct {
		name string
		args args
		want nativeState
	}{
		{name: "open", args: args{raw: json.RawMessage(`"open"`)}, want: nativeStateOpen},
		{name: "closed", args: args{raw: json.RawMessage(`"closed"`)}, want: nativeStateClosed},
		{name: "missing", args: args{raw: nil}, want: nativeStateUnknown},
		{name: "malformed", args: args{raw: json.RawMessage(`{"`)}, want: nativeStateUnknown},
		{name: "unsupported", args: args{raw: json.RawMessage(`"paused"`)}, want: nativeStateUnknown},
		{name: "zero value", args: args{raw: json.RawMessage(`0`)}, want: nativeStateUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, decodeNativeState(tt.args.raw)); diff != "" {
				t.Errorf("decodeNativeState() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
