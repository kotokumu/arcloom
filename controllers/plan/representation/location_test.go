package planrepresentation_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestAcceptanceConditionInvalid(t *testing.T) {
	type args struct {
		statement string
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Location
		wantErr bool
	}{
		{name: "blank", args: args{statement: " \u2003"}, want: planrepresentation.Location{}, wantErr: true},
		{name: "invalid UTF-8", args: args{statement: string([]byte{0xff})}, want: planrepresentation.Location{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planrepresentation.AcceptanceCondition(tt.args.statement)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AcceptanceCondition() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want.Kind(), got.Kind()); diff != "" {
				t.Errorf("kind mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := got.Member()
			wantMember, wantHasMember := tt.want.Member()
			if diff := cmp.Diff(wantMember, member); diff != "" {
				t.Errorf("member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantHasMember, hasMember); diff != "" {
				t.Errorf("member presence mismatch (-want +got):\n%s", diff)
			}
			var validation *plan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want *plan.ValidationError", err)
			}
			if diff := cmp.Diff(plan.InvalidText, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.AcceptanceConditionElement, validation.Element()); diff != "" {
				t.Errorf("element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := validation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskInvalidText(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Location
		wantErr bool
	}{
		{name: "blank", args: args{name: " \u2003"}, want: planrepresentation.Location{}, wantErr: true},
		{name: "invalid UTF-8", args: args{name: string([]byte{0xff})}, want: planrepresentation.Location{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planrepresentation.Task(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Task() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want.Kind(), got.Kind()); diff != "" {
				t.Errorf("kind mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := got.Member()
			wantMember, wantHasMember := tt.want.Member()
			if diff := cmp.Diff(wantMember, member); diff != "" {
				t.Errorf("member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantHasMember, hasMember); diff != "" {
				t.Errorf("member presence mismatch (-want +got):\n%s", diff)
			}
			var validation *plan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want *plan.ValidationError", err)
			}
			if diff := cmp.Diff(plan.InvalidText, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.TaskElement, validation.Element()); diff != "" {
				t.Errorf("element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := validation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestTaskMultiline(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Location
		wantErr bool
	}{
		{name: "carriage return", args: args{name: "Build\rTest"}, want: planrepresentation.Location{}, wantErr: true},
		{name: "line feed", args: args{name: "Build\nTest"}, want: planrepresentation.Location{}, wantErr: true},
		{name: "NEL", args: args{name: "Build\u0085Test"}, want: planrepresentation.Location{}, wantErr: true},
		{name: "line separator", args: args{name: "Build\u2028Test"}, want: planrepresentation.Location{}, wantErr: true},
		{name: "paragraph separator", args: args{name: "Build\u2029Test"}, want: planrepresentation.Location{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planrepresentation.Task(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Task() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want.Kind(), got.Kind()); diff != "" {
				t.Errorf("kind mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := got.Member()
			wantMember, wantHasMember := tt.want.Member()
			if diff := cmp.Diff(wantMember, member); diff != "" {
				t.Errorf("member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantHasMember, hasMember); diff != "" {
				t.Errorf("member presence mismatch (-want +got):\n%s", diff)
			}
			var validation *plan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want *plan.ValidationError", err)
			}
			if diff := cmp.Diff(plan.MultilineName, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.TaskElement, validation.Element()); diff != "" {
				t.Errorf("element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := validation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLocation_Kind(t *testing.T) {
	tests := []struct {
		name string
		l    planrepresentation.Location
		want planrepresentation.LocationKind
	}{
		{name: "zero", l: planrepresentation.Location{}, want: planrepresentation.LocationKind(0)},
		{name: "root", l: planrepresentation.PlanRoot(), want: planrepresentation.PlanRootLocation},
		{name: "name", l: planrepresentation.PlanName(), want: planrepresentation.PlanNameLocation},
		{name: "goal", l: planrepresentation.Goal(), want: planrepresentation.GoalLocation},
		{name: "acceptance collection", l: planrepresentation.AcceptanceConditions(), want: planrepresentation.AcceptanceConditionCollectionLocation},
		{name: "task collection", l: planrepresentation.Tasks(), want: planrepresentation.TaskCollectionLocation},
		{name: "target date", l: planrepresentation.TargetDate(), want: planrepresentation.TargetDateLocation},
		{name: "acceptance member", l: must(planrepresentation.AcceptanceCondition("A")), want: planrepresentation.AcceptanceConditionMemberLocation},
		{name: "task member", l: must(planrepresentation.Task("Build")), want: planrepresentation.TaskMemberLocation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if diff := cmp.Diff(tt.want, tt.l.Kind()); diff != "" {
				t.Errorf("Location.Kind() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLocation_Member(t *testing.T) {
	tests := []struct {
		name  string
		l     planrepresentation.Location
		want  string
		want1 bool
	}{
		{name: "zero", l: planrepresentation.Location{}, want: "", want1: false},
		{name: "root", l: planrepresentation.PlanRoot(), want: "", want1: false},
		{name: "scalar", l: planrepresentation.PlanName(), want: "", want1: false},
		{name: "acceptance member", l: must(planrepresentation.AcceptanceCondition("A\nB")), want: "A\nB", want1: true},
		{name: "task member", l: must(planrepresentation.Task(" 試験 ")), want: " 試験 ", want1: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.l.Member()
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("Location.Member() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want1, got1); diff != "" {
				t.Errorf("Location.Member() presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLeastUnavailableLocationValid(t *testing.T) {
	type args struct {
		affected []planrepresentation.Location
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Location
		wantErr bool
	}{
		{name: "one scalar", args: args{affected: []planrepresentation.Location{planrepresentation.PlanName()}}, want: planrepresentation.PlanName(), wantErr: false},
		{name: "one collection", args: args{affected: []planrepresentation.Location{planrepresentation.Tasks()}}, want: planrepresentation.Tasks(), wantErr: false},
		{name: "one acceptance member", args: args{affected: []planrepresentation.Location{must(planrepresentation.AcceptanceCondition("A"))}}, want: planrepresentation.AcceptanceConditions(), wantErr: false},
		{name: "one task member", args: args{affected: []planrepresentation.Location{must(planrepresentation.Task("Build"))}}, want: planrepresentation.Tasks(), wantErr: false},
		{name: "acceptance members", args: args{affected: []planrepresentation.Location{must(planrepresentation.AcceptanceCondition("A")), must(planrepresentation.AcceptanceCondition("B"))}}, want: planrepresentation.AcceptanceConditions(), wantErr: false},
		{name: "Task members", args: args{affected: []planrepresentation.Location{must(planrepresentation.Task("Build")), must(planrepresentation.Task("Test"))}}, want: planrepresentation.Tasks(), wantErr: false},
		{name: "two top-level branches", args: args{affected: []planrepresentation.Location{planrepresentation.PlanName(), planrepresentation.Goal()}}, want: planrepresentation.PlanRoot(), wantErr: false},
		{name: "root plus descendant", args: args{affected: []planrepresentation.Location{planrepresentation.PlanRoot(), planrepresentation.TargetDate()}}, want: planrepresentation.PlanRoot(), wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planrepresentation.LeastUnavailableLocation(tt.args.affected...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LeastUnavailableLocation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want.Kind(), got.Kind()); diff != "" {
				t.Errorf("kind mismatch (-want +got):\n%s", diff)
			}
			wantMember, wantHasMember := tt.want.Member()
			gotMember, gotHasMember := got.Member()
			if diff := cmp.Diff(wantMember, gotMember); diff != "" {
				t.Errorf("member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantHasMember, gotHasMember); diff != "" {
				t.Errorf("member presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestLeastUnavailableLocationInvalid(t *testing.T) {
	type args struct {
		affected []planrepresentation.Location
	}
	tests := []struct {
		name    string
		args    args
		want    planrepresentation.Location
		wantErr bool
	}{
		{name: "empty", args: args{affected: nil}, want: planrepresentation.Location{}, wantErr: true},
		{name: "zero location", args: args{affected: []planrepresentation.Location{{}}}, want: planrepresentation.Location{}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := planrepresentation.LeastUnavailableLocation(tt.args.affected...)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LeastUnavailableLocation() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want.Kind(), got.Kind()); diff != "" {
				t.Errorf("kind mismatch (-want +got):\n%s", diff)
			}
			member, hasMember := got.Member()
			wantMember, wantHasMember := tt.want.Member()
			if diff := cmp.Diff(wantMember, member); diff != "" {
				t.Errorf("member mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(wantHasMember, hasMember); diff != "" {
				t.Errorf("member presence mismatch (-want +got):\n%s", diff)
			}
			var validation *planrepresentation.ObservationValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("error = %v, want *ObservationValidationError", err)
			}
		})
	}
}
