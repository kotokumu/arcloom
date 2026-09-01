// Package planapplication owns one invocation-local request to an external
// Actor for an exact authorized Plan revision.
package planapplication

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/arcloom/authorization"
	"github.com/kotokumu/arcloom/controllers/plan"
)

func must[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func TestNewTargetReference(t *testing.T) {
	type args struct {
		contextValue string
		identity     string
	}
	tests := []struct {
		name    string
		args    args
		want    TargetReference
		wantErr bool
	}{
		{name: "valid", args: args{contextValue: "github:repo", identity: "milestone:7"}, want: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}},
		{name: "surrounding whitespace is preserved", args: args{contextValue: " repo ", identity: " 7 "}, want: TargetReference{context: " repo ", identity: " 7 ", valid: true}},
		{name: "empty context", args: args{contextValue: "", identity: "milestone:7"}, wantErr: true},
		{name: "blank identity", args: args{contextValue: "github:repo", identity: " \t"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTargetReference(tt.args.contextValue, tt.args.identity)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(TargetReference{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTargetReference_Context(t *testing.T) {
	type fields struct {
		context  string
		identity string
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "valid", fields: fields{context: "github:repo", identity: "milestone:7", valid: true}, want: "github:repo"},
		{name: "zero", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TargetReference{
				context:  tt.fields.context,
				identity: tt.fields.identity,
				valid:    tt.fields.valid,
			}
			if diff := cmp.Diff(tt.want, tr.Context()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTargetReference_Identity(t *testing.T) {
	type fields struct {
		context  string
		identity string
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "valid", fields: fields{context: "github:repo", identity: "milestone:7", valid: true}, want: "milestone:7"},
		{name: "zero", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := TargetReference{
				context:  tt.fields.context,
				identity: tt.fields.identity,
				valid:    tt.fields.valid,
			}
			if diff := cmp.Diff(tt.want, tr.Identity()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewRevision(t *testing.T) {
	type args struct {
		target   TargetReference
		current  plan.Plan
		proposed plan.Plan
	}
	tests := []struct {
		name    string
		args    args
		want    Revision
		wantErr bool
	}{
		{name: "valid distinct plans", args: args{
			target:   must(NewTargetReference("github:repo", "milestone:7")),
			current:  must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)),
			proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)),
		}, want: Revision{
			target:   must(NewTargetReference("github:repo", "milestone:7")),
			current:  must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)),
			proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, []plan.Task{must(plan.NewTask("task"))}, nil)),
			valid:    true,
		}},
		{name: "zero target", args: args{
			current:  must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
			proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
		}, want: Revision{}, wantErr: true},
		{name: "zero current", args: args{
			target:   must(NewTargetReference("github:repo", "milestone:7")),
			proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
		}, want: Revision{}, wantErr: true},
		{name: "zero proposed", args: args{
			target:  must(NewTargetReference("github:repo", "milestone:7")),
			current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
		}, want: Revision{}, wantErr: true},
		{name: "equal plans", args: args{
			target:   must(NewTargetReference("github:repo", "milestone:7")),
			current:  must(plan.New("same", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
			proposed: must(plan.New("same", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
		}, want: Revision{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRevision(tt.args.target, tt.args.current, tt.args.proposed)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Revision{}, TargetReference{}, plan.Plan{}, plan.Goal{}, plan.AcceptanceCondition{}, plan.Task{}, plan.TargetDate{}), cmp.FilterValues(func(want, got Revision) bool {
				return !want.valid && !got.valid
			}, cmp.Transformer("invalidRevisionShape", func(value Revision) struct {
				TargetContext  string
				TargetIdentity string
				TargetValid    bool
				CurrentValid   bool
				ProposedValid  bool
				RevisionValid  bool
			} {
				return struct {
					TargetContext  string
					TargetIdentity string
					TargetValid    bool
					CurrentValid   bool
					ProposedValid  bool
					RevisionValid  bool
				}{
					TargetContext: value.target.context, TargetIdentity: value.target.identity,
					TargetValid: value.target.valid, CurrentValid: value.current.IsValid(),
					ProposedValid: value.proposed.IsValid(), RevisionValid: value.valid,
				}
			}))); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRevision_Target(t *testing.T) {
	type fields struct {
		target   TargetReference
		current  plan.Plan
		proposed plan.Plan
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   TargetReference
	}{
		{name: "valid", fields: fields{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, valid: true}, want: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}},
		{name: "zero", want: TargetReference{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Revision{
				target:   tt.fields.target,
				current:  tt.fields.current,
				proposed: tt.fields.proposed,
				valid:    tt.fields.valid,
			}
			if diff := cmp.Diff(tt.want, r.Target(), cmp.AllowUnexported(TargetReference{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRevision_Current(t *testing.T) {
	type fields struct {
		target   TargetReference
		current  plan.Plan
		proposed plan.Plan
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   plan.Plan
	}{
		{name: "valid", fields: fields{current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, want: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))},
		{name: "zero", want: plan.Plan{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Revision{
				target:   tt.fields.target,
				current:  tt.fields.current,
				proposed: tt.fields.proposed,
				valid:    tt.fields.valid,
			}
			got := r.Current()
			wantConditions, gotConditions := tt.want.AcceptanceConditions(), got.AcceptanceConditions()
			wantConditionText, gotConditionText := make([]string, len(wantConditions)), make([]string, len(gotConditions))
			for i := range wantConditions {
				wantConditionText[i] = wantConditions[i].Statement()
			}
			for i := range gotConditions {
				gotConditionText[i] = gotConditions[i].Statement()
			}
			wantTasks, gotTasks := tt.want.Tasks(), got.Tasks()
			wantTaskNames, gotTaskNames := make([]string, len(wantTasks)), make([]string, len(gotTasks))
			for i := range wantTasks {
				wantTaskNames[i] = wantTasks[i].Name()
			}
			for i := range gotTasks {
				gotTaskNames[i] = gotTasks[i].Name()
			}
			wantDate, wantHasDate := tt.want.TargetDate()
			gotDate, gotHasDate := got.TargetDate()
			if diff := cmp.Diff(struct {
				Valid             bool
				Name, Goal        string
				Conditions, Tasks []string
				Date              string
				HasDate           bool
			}{tt.want.IsValid(), tt.want.Name(), tt.want.Goal().Text(), wantConditionText, wantTaskNames, wantDate.String(), wantHasDate}, struct {
				Valid             bool
				Name, Goal        string
				Conditions, Tasks []string
				Date              string
				HasDate           bool
			}{got.IsValid(), got.Name(), got.Goal().Text(), gotConditionText, gotTaskNames, gotDate.String(), gotHasDate}); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRevision_Proposed(t *testing.T) {
	type fields struct {
		target   TargetReference
		current  plan.Plan
		proposed plan.Plan
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   plan.Plan
	}{
		{name: "valid", fields: fields{proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, want: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))},
		{name: "zero", want: plan.Plan{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Revision{
				target:   tt.fields.target,
				current:  tt.fields.current,
				proposed: tt.fields.proposed,
				valid:    tt.fields.valid,
			}
			got := r.Proposed()
			wantConditions, gotConditions := tt.want.AcceptanceConditions(), got.AcceptanceConditions()
			wantConditionText, gotConditionText := make([]string, len(wantConditions)), make([]string, len(gotConditions))
			for i := range wantConditions {
				wantConditionText[i] = wantConditions[i].Statement()
			}
			for i := range gotConditions {
				gotConditionText[i] = gotConditions[i].Statement()
			}
			wantTasks, gotTasks := tt.want.Tasks(), got.Tasks()
			wantTaskNames, gotTaskNames := make([]string, len(wantTasks)), make([]string, len(gotTasks))
			for i := range wantTasks {
				wantTaskNames[i] = wantTasks[i].Name()
			}
			for i := range gotTasks {
				gotTaskNames[i] = gotTasks[i].Name()
			}
			wantDate, wantHasDate := tt.want.TargetDate()
			gotDate, gotHasDate := got.TargetDate()
			if diff := cmp.Diff(struct {
				Valid             bool
				Name, Goal        string
				Conditions, Tasks []string
				Date              string
				HasDate           bool
			}{tt.want.IsValid(), tt.want.Name(), tt.want.Goal().Text(), wantConditionText, wantTaskNames, wantDate.String(), wantHasDate}, struct {
				Valid             bool
				Name, Goal        string
				Conditions, Tasks []string
				Date              string
				HasDate           bool
			}{got.IsValid(), got.Name(), got.Goal().Text(), gotConditionText, gotTaskNames, gotDate.String(), gotHasDate}); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRevision_Equal(t *testing.T) {
	type fields struct {
		target   TargetReference
		current  plan.Plan
		proposed plan.Plan
		valid    bool
	}
	type args struct {
		other Revision
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{name: "identical valid revisions", fields: fields{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, args: args{other: Revision{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}}, want: true},
		{name: "invalid versus invalid fails closed", args: args{other: Revision{}}, want: false},
		{name: "valid versus invalid", fields: fields{valid: true}, args: args{other: Revision{}}, want: false},
		{name: "different target", fields: fields{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, args: args{other: Revision{target: TargetReference{context: "github:other", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}}, want: false},
		{name: "different target identity", fields: fields{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, args: args{other: Revision{target: TargetReference{context: "github:repo", identity: "milestone:8", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}}, want: false},
		{name: "different current plan", fields: fields{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, args: args{other: Revision{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("other-current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}}, want: false},
		{name: "different proposed plan", fields: fields{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}, args: args{other: Revision{target: TargetReference{context: "github:repo", identity: "milestone:7", valid: true}, current: must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), proposed: must(plan.New("other-next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)), valid: true}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Revision{
				target:   tt.fields.target,
				current:  tt.fields.current,
				proposed: tt.fields.proposed,
				valid:    tt.fields.valid,
			}
			if diff := cmp.Diff(tt.want, r.Equal(tt.args.other)); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequest_Revision(t *testing.T) {
	type fields struct {
		revision Revision
		valid    bool
	}
	tests := []struct {
		name   string
		fields fields
		want   Revision
	}{
		{name: "valid", fields: fields{revision: Revision{valid: true}, valid: true}, want: Revision{valid: true}},
		{name: "zero", want: Revision{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Request{
				revision: tt.fields.revision,
				valid:    tt.fields.valid,
			}
			got := r.Revision()
			if diff := cmp.Diff(tt.want.Target().Context(), got.Target().Context()); diff != "" {
				t.Errorf("target context mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Target().Identity(), got.Target().Identity()); diff != "" {
				t.Errorf("target identity mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Current().IsValid(), got.Current().IsValid()); diff != "" {
				t.Errorf("current validity mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Current().Name(), got.Current().Name()); diff != "" {
				t.Errorf("current name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Current().Goal().Text(), got.Current().Goal().Text()); diff != "" {
				t.Errorf("current goal mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Proposed().IsValid(), got.Proposed().IsValid()); diff != "" {
				t.Errorf("proposed validity mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Proposed().Name(), got.Proposed().Name()); diff != "" {
				t.Errorf("proposed name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want.Proposed().Goal().Text(), got.Proposed().Goal().Text()); diff != "" {
				t.Errorf("proposed goal mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAcknowledgedReceipt(t *testing.T) {
	type args struct {
		reference string
	}
	tests := []struct {
		name string
		args args
		want ReceiptEvidence
	}{
		{name: "reference", args: args{reference: "provider:1"}, want: ReceiptEvidence{kind: ReceiptAcknowledged, reference: "provider:1"}},
		{name: "empty reference", args: args{reference: ""}, want: ReceiptEvidence{kind: ReceiptAcknowledged}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, AcknowledgedReceipt(tt.args.reference), cmp.AllowUnexported(ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRefusedReceipt(t *testing.T) {
	type args struct {
		reference string
	}
	tests := []struct {
		name string
		args args
		want ReceiptEvidence
	}{
		{name: "reference", args: args{reference: "provider:2"}, want: ReceiptEvidence{kind: ReceiptRefused, reference: "provider:2"}},
		{name: "empty reference", args: args{reference: ""}, want: ReceiptEvidence{kind: ReceiptRefused}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, RefusedReceipt(tt.args.reference), cmp.AllowUnexported(ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNotReceivedReceipt(t *testing.T) {
	tests := []struct {
		name string
		want ReceiptEvidence
	}{
		{name: "zero evidence", want: ReceiptEvidence{kind: KnownNotReceived}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, NotReceivedReceipt(), cmp.AllowUnexported(ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestUncertainReceipt(t *testing.T) {
	tests := []struct {
		name string
		want ReceiptEvidence
	}{
		{name: "uncertain", want: ReceiptEvidence{kind: ReceiptUncertain}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, UncertainReceipt(), cmp.AllowUnexported(ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReceiptEvidence_Kind(t *testing.T) {
	type fields struct {
		kind      ReceiptKind
		reference string
	}
	tests := []struct {
		name   string
		fields fields
		want   ReceiptKind
	}{
		{name: "acknowledged", fields: fields{kind: ReceiptAcknowledged}, want: ReceiptAcknowledged},
		{name: "refused", fields: fields{kind: ReceiptRefused}, want: ReceiptRefused},
		{name: "not received", fields: fields{kind: KnownNotReceived}, want: KnownNotReceived},
		{name: "unknown kind normalizes", fields: fields{kind: 99}, want: ReceiptUncertain},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ReceiptEvidence{
				kind:      tt.fields.kind,
				reference: tt.fields.reference,
			}
			if diff := cmp.Diff(tt.want, e.Kind()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReceiptEvidence_Reference(t *testing.T) {
	type fields struct {
		kind      ReceiptKind
		reference string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
		want1  bool
	}{
		{name: "acknowledged reference", fields: fields{kind: ReceiptAcknowledged, reference: "native:1"}, want: "native:1", want1: true},
		{name: "refused reference", fields: fields{kind: ReceiptRefused, reference: "native:2"}, want: "native:2", want1: true},
		{name: "empty reference", fields: fields{kind: ReceiptAcknowledged}, want: "", want1: false},
		{name: "uncertain has no reference", fields: fields{kind: ReceiptUncertain, reference: "native:3"}, want: "", want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ReceiptEvidence{
				kind:      tt.fields.kind,
				reference: tt.fields.reference,
			}
			got, got1 := e.Reference()
			if diff := cmp.Diff(struct {
				Value   string
				Present bool
			}{tt.want, tt.want1}, struct {
				Value   string
				Present bool
			}{got, got1}); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReceiptEvidence_established(t *testing.T) {
	type fields struct {
		kind      ReceiptKind
		reference string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{name: "acknowledged", fields: fields{kind: ReceiptAcknowledged}, want: true},
		{name: "refused", fields: fields{kind: ReceiptRefused}, want: true},
		{name: "not received", fields: fields{kind: KnownNotReceived}, want: true},
		{name: "uncertain", fields: fields{kind: ReceiptUncertain}, want: false},
		{name: "unknown kind", fields: fields{kind: 99}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := ReceiptEvidence{
				kind:      tt.fields.kind,
				reference: tt.fields.reference,
			}
			if diff := cmp.Diff(tt.want, e.established()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResult_AuthorizationDecision(t *testing.T) {
	type fields struct {
		decision    authorization.Decision
		hasDecision bool
		receipt     ReceiptEvidence
		hasReceipt  bool
		valid       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   authorization.Decision
		want1  bool
	}{
		{name: "decision present", fields: fields{decision: authorization.Denied, hasDecision: true, valid: true}, want: authorization.Denied, want1: true},
		{name: "receipt only", fields: fields{receipt: ReceiptEvidence{kind: ReceiptAcknowledged}, hasReceipt: true, valid: true}, want: authorization.Undecidable, want1: false},
		{name: "invalid", fields: fields{decision: authorization.Denied, hasDecision: true}, want: authorization.Undecidable, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{
				decision:    tt.fields.decision,
				hasDecision: tt.fields.hasDecision,
				receipt:     tt.fields.receipt,
				hasReceipt:  tt.fields.hasReceipt,
				valid:       tt.fields.valid,
			}
			got, got1 := r.AuthorizationDecision()
			if diff := cmp.Diff(struct {
				Decision authorization.Decision
				Present  bool
			}{tt.want, tt.want1}, struct {
				Decision authorization.Decision
				Present  bool
			}{got, got1}); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResult_Receipt(t *testing.T) {
	type fields struct {
		decision    authorization.Decision
		hasDecision bool
		receipt     ReceiptEvidence
		hasReceipt  bool
		valid       bool
	}
	tests := []struct {
		name   string
		fields fields
		want   ReceiptEvidence
		want1  bool
	}{
		{name: "receipt present", fields: fields{receipt: ReceiptEvidence{kind: ReceiptAcknowledged, reference: "native:1"}, hasReceipt: true, valid: true}, want: ReceiptEvidence{kind: ReceiptAcknowledged, reference: "native:1"}, want1: true},
		{name: "decision only", fields: fields{decision: authorization.Denied, hasDecision: true, valid: true}, want: ReceiptEvidence{}, want1: false},
		{name: "invalid", fields: fields{receipt: ReceiptEvidence{kind: ReceiptAcknowledged}, hasReceipt: true}, want: ReceiptEvidence{}, want1: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Result{
				decision:    tt.fields.decision,
				hasDecision: tt.fields.hasDecision,
				receipt:     tt.fields.receipt,
				hasReceipt:  tt.fields.hasReceipt,
				valid:       tt.fields.valid,
			}
			got, got1 := r.Receipt()
			if diff := cmp.Diff(struct {
				Receipt ReceiptEvidence
				Present bool
			}{tt.want, tt.want1}, struct {
				Receipt ReceiptEvidence
				Present bool
			}{got, got1}, cmp.AllowUnexported(ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailure_Error(t *testing.T) {
	type fields struct {
		code FailureCode
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{name: "invalid revision", fields: fields{code: InvalidRevision}, want: "application failure: invalid_revision"},
		{name: "invalid actor", fields: fields{code: InvalidActor}, want: "application failure: invalid_actor"},
		{name: "zero failure", want: "invalid application failure"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Failure{
				code: tt.fields.code,
			}
			if diff := cmp.Diff(tt.want, f.Error()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFailure_Code(t *testing.T) {
	type fields struct {
		code FailureCode
	}
	tests := []struct {
		name   string
		fields fields
		want   FailureCode
	}{
		{name: "invalid revision", fields: fields{code: InvalidRevision}, want: InvalidRevision},
		{name: "invalid actor", fields: fields{code: InvalidActor}, want: InvalidActor},
		{name: "zero", want: FailureCode("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Failure{
				code: tt.fields.code,
			}
			if diff := cmp.Diff(tt.want, f.Code()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_decisionResult(t *testing.T) {
	type args struct {
		decision authorization.Decision
	}
	tests := []struct {
		name string
		args args
		want Result
	}{
		{name: "authorized fails closed", args: args{decision: authorization.Authorized}, want: Result{decision: authorization.Undecidable, hasDecision: true, valid: true}},
		{name: "denied", args: args{decision: authorization.Denied}, want: Result{decision: authorization.Denied, hasDecision: true, valid: true}},
		{name: "undecidable", args: args{decision: authorization.Undecidable}, want: Result{decision: authorization.Undecidable, hasDecision: true, valid: true}},
		{name: "zero fails closed", args: args{decision: authorization.Decision(0)}, want: Result{decision: authorization.Undecidable, hasDecision: true, valid: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, decisionResult(tt.args.decision), cmp.AllowUnexported(Result{}, ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_receiptResult(t *testing.T) {
	type args struct {
		receipt ReceiptEvidence
	}
	tests := []struct {
		name string
		args args
		want Result
	}{
		{name: "established", args: args{receipt: ReceiptEvidence{kind: ReceiptAcknowledged, reference: "native:1"}}, want: Result{receipt: ReceiptEvidence{kind: ReceiptAcknowledged, reference: "native:1"}, hasReceipt: true, valid: true}},
		{name: "uncertain", args: args{receipt: ReceiptEvidence{kind: ReceiptUncertain, reference: "ignored"}}, want: Result{receipt: ReceiptEvidence{kind: ReceiptUncertain}, hasReceipt: true, valid: true}},
		{name: "unknown kind", args: args{receipt: ReceiptEvidence{kind: 99}}, want: Result{receipt: ReceiptEvidence{kind: ReceiptUncertain}, hasReceipt: true, valid: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, receiptResult(tt.args.receipt), cmp.AllowUnexported(Result{}, ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRequestApplication(t *testing.T) {
	type args struct {
		ctx      context.Context
		revision Revision
		policy   authorization.Policy[Revision]
		actor    Actor
	}
	tests := []struct {
		name    string
		args    args
		want    Result
		wantErr bool
	}{
		{name: "nil context", args: args{revision: Revision{}, actor: nil}, wantErr: true},
		{name: "invalid revision", args: args{ctx: context.Background(), revision: Revision{}, actor: func(context.Context, Request) ReceiptEvidence { return UncertainReceipt() }}, wantErr: true},
		{name: "authorized actor receipt", args: args{
			ctx: context.Background(),
			revision: must(NewRevision(
				must(NewTargetReference("github:repo", "milestone:7")),
				must(plan.New("current", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
				must(plan.New("next", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil)),
			)),
			policy: must(authorization.NewPolicy[Revision](func(context.Context, Revision) (authorization.RuleConclusion, error) {
				return authorization.Permit, nil
			})),
			actor: func(context.Context, Request) ReceiptEvidence { return AcknowledgedReceipt("native:1") },
		}, want: Result{receipt: ReceiptEvidence{kind: ReceiptAcknowledged, reference: "native:1"}, hasReceipt: true, valid: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RequestApplication(tt.args.ctx, tt.args.revision, tt.args.policy, tt.args.actor)
			if diff := cmp.Diff(tt.wantErr, err != nil); diff != "" {
				t.Errorf("error presence mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Result{}, ReceiptEvidence{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
