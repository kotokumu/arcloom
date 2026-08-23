package reconciliation_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/reconciliation"
)

func TestNewResultEmpty(t *testing.T) {
	type args struct {
		differences []int
		unavailable []int
	}
	tests := []struct {
		name string
		args args
		want reconciliation.Result[int, int]
	}{
		{name: "nil inputs", args: args{differences: nil, unavailable: nil}, want: reconciliation.NewResult[int, int](nil, nil)},
		{name: "empty inputs", args: args{differences: []int{}, unavailable: []int{}}, want: reconciliation.NewResult[int, int]([]int{}, []int{})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconciliation.NewResult[int, int](tt.args.differences, tt.args.unavailable)
			if diff := cmp.Diff(reconciliation.Satisfied, got.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), got.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{}, got.Differences()); diff != "" {
				t.Errorf("differences mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{}, got.UnavailableInformation()); diff != "" {
				t.Errorf("unavailable mismatch (-want +got):\n%s", diff)
			}

			for i := range tt.args.differences {
				tt.args.differences[i] = 99
			}
			for i := range tt.args.unavailable {
				tt.args.unavailable[i] = 99
			}
			if diff := cmp.Diff([]int{}, got.Differences()); diff != "" {
				t.Errorf("input difference snapshot mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{}, got.UnavailableInformation()); diff != "" {
				t.Errorf("input unavailable snapshot mismatch (-want +got):\n%s", diff)
			}

			returnedDifferences := got.Differences()
			for i := range returnedDifferences {
				returnedDifferences[i] = 77
			}
			returnedUnavailable := got.UnavailableInformation()
			for i := range returnedUnavailable {
				returnedUnavailable[i] = 88
			}
			if diff := cmp.Diff([]int{}, got.Differences()); diff != "" {
				t.Errorf("output difference snapshot mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{}, got.UnavailableInformation()); diff != "" {
				t.Errorf("output unavailable snapshot mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(reconciliation.Satisfied, got.Determination()); diff != "" {
				t.Errorf("determination after mutation mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResultDifferences(t *testing.T) {
	type args struct {
		differences []int
		unavailable []int
	}
	tests := []struct {
		name string
		args args
		want reconciliation.Result[int, int]
	}{
		{name: "difference only", args: args{differences: []int{1, 2}, unavailable: nil}, want: reconciliation.NewResult[int, int]([]int{1, 2}, nil)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconciliation.NewResult[int, int](tt.args.differences, tt.args.unavailable)
			if diff := cmp.Diff(reconciliation.NotSatisfied, got.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), got.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{1, 2}, got.Differences()); diff != "" {
				t.Errorf("differences mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{}, got.UnavailableInformation()); diff != "" {
				t.Errorf("unavailable mismatch (-want +got):\n%s", diff)
			}
			tt.args.differences[0] = 99
			tt.args.differences[1] = 98
			if diff := cmp.Diff([]int{1, 2}, got.Differences()); diff != "" {
				t.Errorf("input snapshot mismatch (-want +got):\n%s", diff)
			}
			returned := got.Differences()
			returned[0] = 77
			if diff := cmp.Diff([]int{1, 2}, got.Differences()); diff != "" {
				t.Errorf("output snapshot mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResultUnavailable(t *testing.T) {
	type args struct {
		differences []int
		unavailable []int
	}
	tests := []struct {
		name string
		args args
		want reconciliation.Result[int, int]
	}{
		{name: "unavailable only", args: args{differences: nil, unavailable: []int{3, 4}}, want: reconciliation.NewResult[int, int](nil, []int{3, 4})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconciliation.NewResult[int, int](tt.args.differences, tt.args.unavailable)
			if diff := cmp.Diff(reconciliation.Undecidable, got.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), got.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{}, got.Differences()); diff != "" {
				t.Errorf("differences mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{3, 4}, got.UnavailableInformation()); diff != "" {
				t.Errorf("unavailable mismatch (-want +got):\n%s", diff)
			}
			tt.args.unavailable[0] = 99
			tt.args.unavailable[1] = 98
			if diff := cmp.Diff([]int{3, 4}, got.UnavailableInformation()); diff != "" {
				t.Errorf("input snapshot mismatch (-want +got):\n%s", diff)
			}
			returned := got.UnavailableInformation()
			returned[0] = 77
			if diff := cmp.Diff([]int{3, 4}, got.UnavailableInformation()); diff != "" {
				t.Errorf("output snapshot mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewResultMixed(t *testing.T) {
	type args struct {
		differences []int
		unavailable []int
	}
	tests := []struct {
		name string
		args args
		want reconciliation.Result[int, int]
	}{
		{name: "mixed evidence", args: args{differences: []int{1}, unavailable: []int{2}}, want: reconciliation.NewResult[int, int]([]int{1}, []int{2})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reconciliation.NewResult[int, int](tt.args.differences, tt.args.unavailable)
			if diff := cmp.Diff(reconciliation.Undecidable, got.Determination()); diff != "" {
				t.Errorf("determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Determination(), got.Determination()); diff != "" {
				t.Errorf("scaffold determination mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{1}, got.Differences()); diff != "" {
				t.Errorf("differences mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff([]int{2}, got.UnavailableInformation()); diff != "" {
				t.Errorf("unavailable mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResult_Determination(t *testing.T) {
	tests := []struct {
		name string
		r    reconciliation.Result[int, int]
		want reconciliation.Determination
	}{
		{name: "zero result", r: reconciliation.Result[int, int]{}, want: reconciliation.Determination(0)},
		{name: "valid empty result", r: reconciliation.NewResult[int, int](nil, nil), want: reconciliation.Satisfied},
		{name: "difference only", r: reconciliation.NewResult[int, int]([]int{1}, nil), want: reconciliation.NotSatisfied},
		{name: "unavailable only", r: reconciliation.NewResult[int, int](nil, []int{1}), want: reconciliation.Undecidable},
		{name: "mixed evidence", r: reconciliation.NewResult[int, int]([]int{1}, []int{2}), want: reconciliation.Undecidable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Determination()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Result[D, U].Determination() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResult_Differences(t *testing.T) {
	tests := []struct {
		name string
		r    reconciliation.Result[int, int]
		want []int
	}{
		{name: "zero result", r: reconciliation.Result[int, int]{}, want: nil},
		{name: "valid empty result", r: reconciliation.NewResult[int, int](nil, nil), want: []int{}},
		{name: "evidence", r: reconciliation.NewResult[int, int]([]int{1, 2}, nil), want: []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Differences()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Result[D, U].Differences() mismatch (-want +got):\n%s", diff)
			}
			for i := range got {
				got[i] = 99
			}
			if diff := cmp.Diff(tt.want, tt.r.Differences()); diff != "" {
				t.Errorf("Result[D, U].Differences() output snapshot mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResult_UnavailableInformation(t *testing.T) {
	tests := []struct {
		name string
		r    reconciliation.Result[int, int]
		want []int
	}{
		{name: "zero result", r: reconciliation.Result[int, int]{}, want: nil},
		{name: "valid empty result", r: reconciliation.NewResult[int, int](nil, nil), want: []int{}},
		{name: "evidence", r: reconciliation.NewResult[int, int](nil, []int{1, 2}), want: []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.UnavailableInformation()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Result[D, U].UnavailableInformation() mismatch (-want +got):\n%s", diff)
			}
			for i := range got {
				got[i] = 99
			}
			if diff := cmp.Diff(tt.want, tt.r.UnavailableInformation()); diff != "" {
				t.Errorf("Result[D, U].UnavailableInformation() output snapshot mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
