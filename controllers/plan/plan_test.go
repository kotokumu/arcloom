package plan_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
)

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestNewGoal(t *testing.T) {
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		args    args
		want    plan.Goal
		wantErr bool
	}{
		{name: "ascii", args: args{text: "goal"}, want: must(plan.NewGoal("goal"))},
		{name: "unicode and surrounding whitespace", args: args{text: "  ゴール\t"}, want: must(plan.NewGoal("  ゴール\t"))},
		{name: "multiline preserved", args: args{text: "line\r\nnext"}, want: must(plan.NewGoal("line\r\nnext"))},
		{name: "NEL preserved", args: args{text: "line\u0085next"}, want: must(plan.NewGoal("line\u0085next"))},
		{name: "line separator preserved", args: args{text: "line\u2028next"}, want: must(plan.NewGoal("line\u2028next"))},
		{name: "paragraph separator preserved", args: args{text: "line\u2029next"}, want: must(plan.NewGoal("line\u2029next"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.NewGoal(tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewGoal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plan.Goal{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewAcceptanceCondition(t *testing.T) {
	type args struct {
		statement string
	}
	tests := []struct {
		name    string
		args    args
		want    plan.AcceptanceCondition
		wantErr bool
	}{
		{name: "ascii", args: args{statement: "accept"}, want: must(plan.NewAcceptanceCondition("accept"))},
		{name: "unicode and surrounding whitespace", args: args{statement: "  受入\t"}, want: must(plan.NewAcceptanceCondition("  受入\t"))},
		{name: "carriage return preserved", args: args{statement: "a\rb"}, want: must(plan.NewAcceptanceCondition("a\rb"))},
		{name: "line feed preserved", args: args{statement: "a\nb"}, want: must(plan.NewAcceptanceCondition("a\nb"))},
		{name: "NEL preserved", args: args{statement: "a\u0085b"}, want: must(plan.NewAcceptanceCondition("a\u0085b"))},
		{name: "line separator preserved", args: args{statement: "a\u2028b"}, want: must(plan.NewAcceptanceCondition("a\u2028b"))},
		{name: "paragraph separator preserved", args: args{statement: "a\u2029b"}, want: must(plan.NewAcceptanceCondition("a\u2029b"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.NewAcceptanceCondition(tt.args.statement)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewAcceptanceCondition() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plan.AcceptanceCondition{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewTask(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    plan.Task
		wantErr bool
	}{
		{name: "ascii", args: args{name: "task"}, want: must(plan.NewTask("task"))},
		{name: "unicode and surrounding whitespace", args: args{name: "  作業\t"}, want: must(plan.NewTask("  作業\t"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.NewTask(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewTask() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plan.Task{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseTargetDate(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		args    args
		want    plan.TargetDate
		wantErr bool
	}{
		{name: "lower bound", args: args{value: "0001-01-01"}, want: must(plan.ParseTargetDate("0001-01-01"))},
		{name: "upper bound", args: args{value: "9999-12-31"}, want: must(plan.ParseTargetDate("9999-12-31"))},
		{name: "1900", args: args{value: "1900-02-28"}, want: must(plan.ParseTargetDate("1900-02-28"))},
		{name: "2000 leap", args: args{value: "2000-02-29"}, want: must(plan.ParseTargetDate("2000-02-29"))},
		{name: "2028 leap", args: args{value: "2028-02-29"}, want: must(plan.ParseTargetDate("2028-02-29"))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.ParseTargetDate(tt.args.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseTargetDate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plan.TargetDate{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewPlan(t *testing.T) {
	type args struct {
		name       string
		goal       plan.Goal
		conditions []plan.AcceptanceCondition
		tasks      []plan.Task
		targetDate *plan.TargetDate
	}
	tests := []struct {
		name    string
		args    args
		want    plan.Plan
		wantErr bool
	}{
		{name: "minimum", args: args{name: "Plan", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}}, want: must(plan.New("Plan", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))},
		{name: "surrounding whitespace preserved", args: args{name: " Plan ", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}}, want: must(plan.New(" Plan ", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.New(tt.args.name, tt.args.goal, tt.args.conditions, tt.args.tasks, tt.args.targetDate)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(plan.Plan{}, plan.Goal{}, plan.AcceptanceCondition{}, plan.Task{}, plan.TargetDate{})); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanValidityBelongsToPlan(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	constructed := must(plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, nil, nil))
	if diff := cmp.Diff(true, constructed.IsValid()); diff != "" {
		t.Errorf("constructed validity mismatch (-want +got):\n%s", diff)
	}
	var zero plan.Plan
	if diff := cmp.Diff(false, zero.IsValid()); diff != "" {
		t.Errorf("zero validity mismatch (-want +got):\n%s", diff)
	}
}

func TestNewPlanZeroValueBoundaries(t *testing.T) {
	zeroDate := plan.TargetDate{}
	tests := []struct {
		name         string
		goal         plan.Goal
		conditions   []plan.AcceptanceCondition
		date         *plan.TargetDate
		wantCode     plan.ViolationCode
		wantElement  plan.ElementKind
		wantIndex    int
		wantHasIndex bool
	}{
		{name: "zero goal", conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.InvalidText, wantElement: plan.GoalElement},
		{name: "zero acceptance condition", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{{}}, wantCode: plan.InvalidText, wantElement: plan.AcceptanceConditionElement, wantIndex: 0, wantHasIndex: true},
		{name: "zero target date", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, date: &zeroDate, wantCode: plan.InvalidTargetDate, wantElement: plan.TargetDateElement, wantIndex: 0, wantHasIndex: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.New("Plan", tt.goal, tt.conditions, nil, tt.date)
			var validation *plan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(tt.wantCode, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantElement, validation.Element()); diff != "" {
				t.Errorf("element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := validation.Index()
			if diff := cmp.Diff(tt.wantHasIndex, hasIndex); diff != "" {
				t.Errorf("index presence mismatch (-want +got):\n%s", diff)
			}
			if tt.wantHasIndex {
				if diff := cmp.Diff(tt.wantIndex, index); diff != "" {
					t.Errorf("index mismatch (-want +got):\n%s", diff)
				}
			}
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("validity mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Name()); diff != "" {
				t.Errorf("name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Goal().Text()); diff != "" {
				t.Errorf("goal mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(got.AcceptanceConditions())); diff != "" {
				t.Errorf("conditions mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(got.Tasks())); diff != "" {
				t.Errorf("tasks mismatch (-want +got):\n%s", diff)
			}
			date, hasDate := got.TargetDate()
			if diff := cmp.Diff("", date.String()); diff != "" {
				t.Errorf("date mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasDate); diff != "" {
				t.Errorf("date presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewGoalValidation(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{name: "blank", text: " \u2003"},
		{name: "invalid utf8", text: string([]byte{0xff})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.NewGoal(tt.text)
			var validation *plan.ValidationError
			if !errors.As(err, &validation) || validation.Code() != plan.InvalidText || validation.Element() != plan.GoalElement {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff("", got.Text()); diff != "" {
				t.Errorf("zero goal mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewAcceptanceConditionValidation(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{name: "blank", statement: " \u2003"},
		{name: "invalid utf8", statement: string([]byte{0xff})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.NewAcceptanceCondition(tt.statement)
			var validation *plan.ValidationError
			if !errors.As(err, &validation) || validation.Code() != plan.InvalidText || validation.Element() != plan.AcceptanceConditionElement {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff("", got.Statement()); diff != "" {
				t.Errorf("zero condition mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewTaskValidation(t *testing.T) {
	tests := []struct {
		name string
		text string
		code plan.ViolationCode
	}{
		{name: "carriage return", text: "a\rb", code: plan.MultilineName},
		{name: "line feed", text: "a\nb", code: plan.MultilineName},
		{name: "NEL", text: "a\u0085b", code: plan.MultilineName},
		{name: "line separator", text: "a\u2028b", code: plan.MultilineName},
		{name: "paragraph separator", text: "a\u2029b", code: plan.MultilineName},
		{name: "blank", text: " \u2003", code: plan.InvalidText},
		{name: "invalid utf8", text: string([]byte{0xff}), code: plan.InvalidText},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.NewTask(tt.text)
			var validation *plan.ValidationError
			if !errors.As(err, &validation) || validation.Code() != tt.code || validation.Element() != plan.TaskElement {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff("", got.Name()); diff != "" {
				t.Errorf("zero task mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseTargetDateValidation(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "1900 leap day", value: "1900-02-29"},
		{name: "year zero", value: "0000-01-01"},
		{name: "non padded", value: "2028-2-29"},
		{name: "invalid month", value: "2028-13-01"},
		{name: "invalid day", value: "2028-02-30"},
		{name: "whitespace", value: " 2028-02-29"},
		{name: "time suffix", value: "2028-02-29T00:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.ParseTargetDate(tt.value)
			var validation *plan.ValidationError
			if !errors.As(err, &validation) || validation.Code() != plan.InvalidTargetDate || validation.Element() != plan.TargetDateElement {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff("", got.String()); diff != "" {
				t.Errorf("zero date mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanPreservesNearEqualUnicodeInDeclaredOrder(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	nfc := must(plan.NewAcceptanceCondition("caf\u00e9"))
	nfd := must(plan.NewAcceptanceCondition("cafe\u0301"))
	first := must(plan.NewTask("t\u00e9"))
	second := must(plan.NewTask("te\u0301"))
	value := must(plan.New("Plan", goal, []plan.AcceptanceCondition{nfc, nfd}, []plan.Task{first, second}, nil))
	conditions := value.AcceptanceConditions()
	if diff := cmp.Diff([]string{"café", "café"}, []string{conditions[0].Statement(), conditions[1].Statement()}); diff != "" {
		t.Errorf("condition order mismatch (-want +got):\n%s", diff)
	}
	tasks := value.Tasks()
	if diff := cmp.Diff([]string{"té", "té"}, []string{tasks[0].Name(), tasks[1].Name()}); diff != "" {
		t.Errorf("task order mismatch (-want +got):\n%s", diff)
	}
}

func TestNewPlanValidation(t *testing.T) {
	tests := []struct {
		name         string
		planName     string
		goal         plan.Goal
		conditions   []plan.AcceptanceCondition
		tasks        []plan.Task
		wantCode     plan.ViolationCode
		wantElement  plan.ElementKind
		wantIndex    int
		wantHasIndex bool
	}{
		{name: "multiline plan name", planName: "Plan\nName", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.MultilineName, wantElement: plan.PlanNameElement},
		{name: "blank plan name", planName: " \u2003", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.InvalidText, wantElement: plan.PlanNameElement},
		{name: "invalid plan name utf8", planName: string([]byte{0xff}), goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.InvalidText, wantElement: plan.PlanNameElement},
		{name: "carriage return plan name", planName: "Plan\rName", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.MultilineName, wantElement: plan.PlanNameElement},
		{name: "line feed plan name", planName: "Plan\nName", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.MultilineName, wantElement: plan.PlanNameElement},
		{name: "NEL plan name", planName: "Plan\u0085Name", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.MultilineName, wantElement: plan.PlanNameElement},
		{name: "line separator plan name", planName: "Plan\u2028Name", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.MultilineName, wantElement: plan.PlanNameElement},
		{name: "paragraph separator plan name", planName: "Plan\u2029Name", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.MultilineName, wantElement: plan.PlanNameElement},
		{name: "missing condition", planName: "Plan", goal: must(plan.NewGoal("goal")), wantCode: plan.MissingAcceptanceCondition, wantElement: plan.AcceptanceConditionElement},
		{name: "duplicate condition", planName: "Plan", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept")), must(plan.NewAcceptanceCondition("accept"))}, wantCode: plan.DuplicateAcceptanceCondition, wantElement: plan.AcceptanceConditionElement, wantIndex: 1, wantHasIndex: true},
		{name: "invalid condition", planName: "Plan", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("valid")), {}}, wantCode: plan.InvalidText, wantElement: plan.AcceptanceConditionElement, wantIndex: 1, wantHasIndex: true},
		{name: "duplicate task", planName: "Plan", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, tasks: []plan.Task{must(plan.NewTask("task")), must(plan.NewTask("task"))}, wantCode: plan.DuplicateTask, wantElement: plan.TaskElement, wantIndex: 1, wantHasIndex: true},
		{name: "invalid task", planName: "Plan", goal: must(plan.NewGoal("goal")), conditions: []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, tasks: []plan.Task{must(plan.NewTask("valid")), {}}, wantCode: plan.InvalidText, wantElement: plan.TaskElement, wantIndex: 1, wantHasIndex: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := plan.New(tt.planName, tt.goal, tt.conditions, tt.tasks, nil)
			var validation *plan.ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("validation = %v", err)
			}
			if diff := cmp.Diff(tt.wantCode, validation.Code()); diff != "" {
				t.Errorf("code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.wantElement, validation.Element()); diff != "" {
				t.Errorf("element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := validation.Index()
			if diff := cmp.Diff(tt.wantHasIndex, hasIndex); diff != "" {
				t.Errorf("index presence mismatch (-want +got):\n%s", diff)
			}
			if tt.wantHasIndex {
				if diff := cmp.Diff(tt.wantIndex, index); diff != "" {
					t.Errorf("index mismatch (-want +got):\n%s", diff)
				}
			}
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("validity mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Name()); diff != "" {
				t.Errorf("zero name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff("", got.Goal().Text()); diff != "" {
				t.Errorf("zero goal mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(got.AcceptanceConditions())); diff != "" {
				t.Errorf("zero conditions mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(0, len(got.Tasks())); diff != "" {
				t.Errorf("zero tasks mismatch (-want +got):\n%s", diff)
			}
			date, hasDate := got.TargetDate()
			if diff := cmp.Diff("", date.String()); diff != "" {
				t.Errorf("zero date mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasDate); diff != "" {
				t.Errorf("zero date presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestPlanDefensiveCopies(t *testing.T) {
	goal := must(plan.NewGoal("goal"))
	condition := must(plan.NewAcceptanceCondition("accept"))
	task := must(plan.NewTask("task"))
	conditions := []plan.AcceptanceCondition{condition}
	tasks := []plan.Task{task}
	value := must(plan.New("Plan", goal, conditions, tasks, nil))
	conditions[0] = plan.AcceptanceCondition{}
	tasks[0] = plan.Task{}
	gotConditions := value.AcceptanceConditions()
	if diff := cmp.Diff([]string{"accept"}, []string{gotConditions[0].Statement()}); diff != "" {
		t.Errorf("input condition copy mismatch (-want +got):\n%s", diff)
	}
	gotTasks := value.Tasks()
	if diff := cmp.Diff([]string{"task"}, []string{gotTasks[0].Name()}); diff != "" {
		t.Errorf("input task copy mismatch (-want +got):\n%s", diff)
	}
	returnedConditions := value.AcceptanceConditions()
	returnedTasks := value.Tasks()
	returnedConditions[0] = plan.AcceptanceCondition{}
	returnedTasks[0] = plan.Task{}
	gotConditions = value.AcceptanceConditions()
	if diff := cmp.Diff([]string{"accept"}, []string{gotConditions[0].Statement()}); diff != "" {
		t.Errorf("output condition copy mismatch (-want +got):\n%s", diff)
	}
	gotTasks = value.Tasks()
	if diff := cmp.Diff([]string{"task"}, []string{gotTasks[0].Name()}); diff != "" {
		t.Errorf("output task copy mismatch (-want +got):\n%s", diff)
	}
}

func TestAllPlanZeroValueAccessorsArePanicFree(t *testing.T) {
	var goal plan.Goal
	if diff := cmp.Diff("", goal.Text()); diff != "" {
		t.Errorf("zero goal text mismatch (-want +got):\n%s", diff)
	}
	var condition plan.AcceptanceCondition
	if diff := cmp.Diff("", condition.Statement()); diff != "" {
		t.Errorf("zero condition mismatch (-want +got):\n%s", diff)
	}
	var task plan.Task
	if diff := cmp.Diff("", task.Name()); diff != "" {
		t.Errorf("zero task mismatch (-want +got):\n%s", diff)
	}
	var date plan.TargetDate
	if diff := cmp.Diff("", date.String()); diff != "" {
		t.Errorf("zero date mismatch (-want +got):\n%s", diff)
	}
	var value plan.Plan
	if diff := cmp.Diff("", value.Name()); diff != "" {
		t.Errorf("zero plan name mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff("", value.Goal().Text()); diff != "" {
		t.Errorf("zero plan goal mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, len(value.AcceptanceConditions())); diff != "" {
		t.Errorf("zero plan conditions mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(0, len(value.Tasks())); diff != "" {
		t.Errorf("zero plan tasks mismatch (-want +got):\n%s", diff)
	}
	if gotDate, ok := value.TargetDate(); ok || gotDate != (plan.TargetDate{}) {
		t.Errorf("zero plan date = %#v, %v", gotDate, ok)
	}
	var validation *plan.ValidationError
	if diff := cmp.Diff("plan validation error", validation.Error()); diff != "" {
		t.Errorf("nil validation error mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plan.ViolationCode(""), validation.Code()); diff != "" {
		t.Errorf("nil validation code mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(plan.ElementKind(0), validation.Element()); diff != "" {
		t.Errorf("nil validation element mismatch (-want +got):\n%s", diff)
	}
	_, ok := validation.Index()
	if ok {
		t.Errorf("nil validation index presence = %v", ok)
	}
}

func TestPlanNamePreservesSurroundingWhitespace(t *testing.T) {
	value := must(plan.New(" Plan ", must(plan.NewGoal("goal")), []plan.AcceptanceCondition{must(plan.NewAcceptanceCondition("accept"))}, nil, nil))
	if diff := cmp.Diff(" Plan ", value.Name()); diff != "" {
		t.Errorf("name mismatch (-want +got):\n%s", diff)
	}
}

func TestTaskNamePreservesSurroundingWhitespace(t *testing.T) {
	value := must(plan.NewTask(" task "))
	if diff := cmp.Diff(" task ", value.Name()); diff != "" {
		t.Errorf("name mismatch (-want +got):\n%s", diff)
	}
}
