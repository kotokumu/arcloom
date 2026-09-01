package plan_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kotokumu/arcloom/controllers/plan"
)

func TestValidateNameValid(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "ASCII", args: args{name: "Release"}, wantErr: false},
		{name: "non ASCII with surrounding whitespace", args: args{name: "  リリース\t"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateName(tt.args.name)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateName() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			if standaloneErr != nil {
				t.Fatalf("ValidateName() error = %v, want nil", standaloneErr)
			}
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New(tt.args.name, goal, []plan.AcceptanceCondition{condition}, nil, nil)
			if aggregateErr != nil {
				t.Fatalf("New() error = %v, want nil", aggregateErr)
			}
			if diff := cmp.Diff(tt.args.name, got.Name()); diff != "" {
				t.Errorf("name mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, got.IsValid()); diff != "" {
				t.Errorf("validity mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateNameInvalidText(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "blank", args: args{name: " \u2003"}, wantErr: true},
		{name: "invalid UTF-8", args: args{name: string([]byte{0xff})}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateName(tt.args.name)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateName() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateName() error = %v, want *plan.ValidationError", standaloneErr)
			}
			if diff := cmp.Diff(plan.InvalidText, standaloneValidation.Code()); diff != "" {
				t.Errorf("standalone code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.PlanNameElement, standaloneValidation.Element()); diff != "" {
				t.Errorf("standalone element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := standaloneValidation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("standalone index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("standalone index presence mismatch (-want +got):\n%s", diff)
			}

			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New(tt.args.name, goal, []plan.AcceptanceCondition{condition}, nil, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.InvalidText, aggregateValidation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.PlanNameElement, aggregateValidation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex = aggregateValidation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateNameMultiline(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "carriage return", args: args{name: "Release\rName"}, wantErr: true},
		{name: "line feed", args: args{name: "Release\nName"}, wantErr: true},
		{name: "NEL", args: args{name: "Release\u0085Name"}, wantErr: true},
		{name: "line separator", args: args{name: "Release\u2028Name"}, wantErr: true},
		{name: "paragraph separator", args: args{name: "Release\u2029Name"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateName(tt.args.name)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateName() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateName() error = %v, want *plan.ValidationError", standaloneErr)
			}
			if diff := cmp.Diff(plan.MultilineName, standaloneValidation.Code()); diff != "" {
				t.Errorf("standalone code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.PlanNameElement, standaloneValidation.Element()); diff != "" {
				t.Errorf("standalone element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := standaloneValidation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("standalone index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("standalone index presence mismatch (-want +got):\n%s", diff)
			}

			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New(tt.args.name, goal, []plan.AcceptanceCondition{condition}, nil, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.MultilineName, aggregateValidation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.PlanNameElement, aggregateValidation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex = aggregateValidation.Index()
			if diff := cmp.Diff(-1, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateAcceptanceConditionsEmpty(t *testing.T) {
	type args struct {
		values []plan.AcceptanceCondition
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil", args: args{values: nil}, wantErr: false},
		{name: "empty", args: args{values: []plan.AcceptanceCondition{}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateAcceptanceConditions(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			if standaloneErr != nil {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, want nil", standaloneErr)
			}
			goal := must(plan.NewGoal("goal"))
			got, aggregateErr := plan.New("Plan", goal, tt.args.values, nil, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var validation *plan.ValidationError
			if !errors.As(aggregateErr, &validation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.MissingAcceptanceCondition, validation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.AcceptanceConditionElement, validation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := validation.Index()
			if diff := cmp.Diff(0, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(false, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateAcceptanceConditionsValid(t *testing.T) {
	type args struct {
		values []plan.AcceptanceCondition
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "distinct Unicode code points", args: args{values: []plan.AcceptanceCondition{
			must(plan.NewAcceptanceCondition("é")),
			must(plan.NewAcceptanceCondition("e\u0301")),
		}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateAcceptanceConditions(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			if standaloneErr != nil {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, want nil", standaloneErr)
			}
			goal := must(plan.NewGoal("goal"))
			got, aggregateErr := plan.New("Plan", goal, tt.args.values, nil, nil)
			if aggregateErr != nil {
				t.Fatalf("New() error = %v, want nil", aggregateErr)
			}
			if diff := cmp.Diff(true, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateAcceptanceConditionsDuplicate(t *testing.T) {
	type args struct {
		values []plan.AcceptanceCondition
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "exact duplicate at index one", args: args{values: []plan.AcceptanceCondition{
			must(plan.NewAcceptanceCondition("A")),
			must(plan.NewAcceptanceCondition("A")),
		}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateAcceptanceConditions(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, want *plan.ValidationError", standaloneErr)
			}
			if diff := cmp.Diff(plan.DuplicateAcceptanceCondition, standaloneValidation.Code()); diff != "" {
				t.Errorf("standalone code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.AcceptanceConditionElement, standaloneValidation.Element()); diff != "" {
				t.Errorf("standalone element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := standaloneValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("standalone index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("standalone index presence mismatch (-want +got):\n%s", diff)
			}
			goal := must(plan.NewGoal("goal"))
			got, aggregateErr := plan.New("Plan", goal, tt.args.values, nil, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.DuplicateAcceptanceCondition, aggregateValidation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.AcceptanceConditionElement, aggregateValidation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex = aggregateValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateAcceptanceConditionsZeroElement(t *testing.T) {
	type args struct {
		values []plan.AcceptanceCondition
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "zero element at index one", args: args{values: []plan.AcceptanceCondition{
			must(plan.NewAcceptanceCondition("A")),
			{},
		}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateAcceptanceConditions(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, want *plan.ValidationError", standaloneErr)
			}
			if diff := cmp.Diff(plan.InvalidText, standaloneValidation.Code()); diff != "" {
				t.Errorf("standalone code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.AcceptanceConditionElement, standaloneValidation.Element()); diff != "" {
				t.Errorf("standalone element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := standaloneValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("standalone index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("standalone index presence mismatch (-want +got):\n%s", diff)
			}
			goal := must(plan.NewGoal("goal"))
			got, aggregateErr := plan.New("Plan", goal, tt.args.values, nil, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.InvalidText, aggregateValidation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.AcceptanceConditionElement, aggregateValidation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex = aggregateValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateAcceptanceConditionsSimultaneousViolations(t *testing.T) {
	type args struct {
		values []plan.AcceptanceCondition
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "duplicate and zero element", args: args{values: []plan.AcceptanceCondition{
			must(plan.NewAcceptanceCondition("A")),
			must(plan.NewAcceptanceCondition("A")),
			{},
		}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateAcceptanceConditions(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateAcceptanceConditions() error = %v, want *plan.ValidationError", standaloneErr)
			}
			index, hasIndex := standaloneValidation.Index()
			allowed := (standaloneValidation.Code() == plan.DuplicateAcceptanceCondition &&
				standaloneValidation.Element() == plan.AcceptanceConditionElement && index == 1 && hasIndex) ||
				(standaloneValidation.Code() == plan.InvalidText &&
					standaloneValidation.Element() == plan.AcceptanceConditionElement && index == 2 && hasIndex)
			if diff := cmp.Diff(true, allowed); diff != "" {
				t.Errorf("standalone allowed violation mismatch (-want +got):\n%s", diff)
			}
			goal := must(plan.NewGoal("goal"))
			got, aggregateErr := plan.New("Plan", goal, tt.args.values, nil, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			aggregateIndex, aggregateHasIndex := aggregateValidation.Index()
			if diff := cmp.Diff(standaloneValidation.Code(), aggregateValidation.Code()); diff != "" {
				t.Errorf("standalone and aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(standaloneValidation.Element(), aggregateValidation.Element()); diff != "" {
				t.Errorf("standalone and aggregate element mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(index, aggregateIndex); diff != "" {
				t.Errorf("standalone and aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(hasIndex, aggregateHasIndex); diff != "" {
				t.Errorf("standalone and aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateTasksEmptyAndValid(t *testing.T) {
	type args struct {
		values []plan.Task
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "nil", args: args{values: nil}, wantErr: false},
		{name: "empty", args: args{values: []plan.Task{}}, wantErr: false},
		{name: "distinct Unicode code points", args: args{values: []plan.Task{
			must(plan.NewTask("é")),
			must(plan.NewTask("e\u0301")),
		}}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateTasks(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateTasks() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			if standaloneErr != nil {
				t.Fatalf("ValidateTasks() error = %v, want nil", standaloneErr)
			}
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tt.args.values, nil)
			if aggregateErr != nil {
				t.Fatalf("New() error = %v, want nil", aggregateErr)
			}
			if diff := cmp.Diff(true, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateTasksDuplicate(t *testing.T) {
	type args struct {
		values []plan.Task
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "exact duplicate at index one", args: args{values: []plan.Task{
			must(plan.NewTask("Build")),
			must(plan.NewTask("Build")),
		}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateTasks(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateTasks() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateTasks() error = %v, want *plan.ValidationError", standaloneErr)
			}
			if diff := cmp.Diff(plan.DuplicateTask, standaloneValidation.Code()); diff != "" {
				t.Errorf("standalone code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.TaskElement, standaloneValidation.Element()); diff != "" {
				t.Errorf("standalone element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := standaloneValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("standalone index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("standalone index presence mismatch (-want +got):\n%s", diff)
			}
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tt.args.values, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.DuplicateTask, aggregateValidation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.TaskElement, aggregateValidation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex = aggregateValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateTasksZeroElement(t *testing.T) {
	type args struct {
		values []plan.Task
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "zero element at index one", args: args{values: []plan.Task{
			must(plan.NewTask("Build")),
			{},
		}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateTasks(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateTasks() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateTasks() error = %v, want *plan.ValidationError", standaloneErr)
			}
			if diff := cmp.Diff(plan.InvalidText, standaloneValidation.Code()); diff != "" {
				t.Errorf("standalone code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.TaskElement, standaloneValidation.Element()); diff != "" {
				t.Errorf("standalone element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex := standaloneValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("standalone index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("standalone index presence mismatch (-want +got):\n%s", diff)
			}
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tt.args.values, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			if diff := cmp.Diff(plan.InvalidText, aggregateValidation.Code()); diff != "" {
				t.Errorf("aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(plan.TaskElement, aggregateValidation.Element()); diff != "" {
				t.Errorf("aggregate element mismatch (-want +got):\n%s", diff)
			}
			index, hasIndex = aggregateValidation.Index()
			if diff := cmp.Diff(1, index); diff != "" {
				t.Errorf("aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(true, hasIndex); diff != "" {
				t.Errorf("aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateTasksSimultaneousViolations(t *testing.T) {
	type args struct {
		values []plan.Task
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "duplicate and zero element", args: args{values: []plan.Task{
			must(plan.NewTask("Build")),
			must(plan.NewTask("Build")),
			{},
		}}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			standaloneErr := plan.ValidateTasks(tt.args.values)
			if (standaloneErr != nil) != tt.wantErr {
				t.Fatalf("ValidateTasks() error = %v, wantErr %v", standaloneErr, tt.wantErr)
			}
			var standaloneValidation *plan.ValidationError
			if !errors.As(standaloneErr, &standaloneValidation) {
				t.Fatalf("ValidateTasks() error = %v, want *plan.ValidationError", standaloneErr)
			}
			index, hasIndex := standaloneValidation.Index()
			allowed := (standaloneValidation.Code() == plan.DuplicateTask &&
				standaloneValidation.Element() == plan.TaskElement && index == 1 && hasIndex) ||
				(standaloneValidation.Code() == plan.InvalidText &&
					standaloneValidation.Element() == plan.TaskElement && index == 2 && hasIndex)
			if diff := cmp.Diff(true, allowed); diff != "" {
				t.Errorf("standalone allowed violation mismatch (-want +got):\n%s", diff)
			}
			goal := must(plan.NewGoal("goal"))
			condition := must(plan.NewAcceptanceCondition("accept"))
			got, aggregateErr := plan.New("Plan", goal, []plan.AcceptanceCondition{condition}, tt.args.values, nil)
			if diff := cmp.Diff(false, got.IsValid()); diff != "" {
				t.Errorf("aggregate validity mismatch (-want +got):\n%s", diff)
			}
			var aggregateValidation *plan.ValidationError
			if !errors.As(aggregateErr, &aggregateValidation) {
				t.Fatalf("New() error = %v, want *plan.ValidationError", aggregateErr)
			}
			aggregateIndex, aggregateHasIndex := aggregateValidation.Index()
			if diff := cmp.Diff(standaloneValidation.Code(), aggregateValidation.Code()); diff != "" {
				t.Errorf("standalone and aggregate code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(standaloneValidation.Element(), aggregateValidation.Element()); diff != "" {
				t.Errorf("standalone and aggregate element mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(index, aggregateIndex); diff != "" {
				t.Errorf("standalone and aggregate index mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(hasIndex, aggregateHasIndex); diff != "" {
				t.Errorf("standalone and aggregate index presence mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
