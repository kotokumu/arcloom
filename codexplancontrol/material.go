package codexplancontrol

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/plancontrol"
)

type planMaterial struct {
	Name                 string   `json:"name"`
	Goal                 string   `json:"goal"`
	AcceptanceConditions []string `json:"acceptanceConditions"`
	Tasks                []string `json:"tasks"`
	TargetDate           *string  `json:"targetDate"`
}

type assessmentMaterial struct {
	CurrentPlan  planMaterial `json:"currentPlan"`
	Observations string       `json:"observations"`
}

func encodeAssessmentMaterial(current plan.Plan, observations string) ([]byte, error) {
	material := assessmentMaterial{CurrentPlan: planMaterialFrom(current), Observations: observations}
	encoded, err := json.Marshal(material)
	if err != nil {
		return nil, errAssessmentUnavailable
	}
	return encoded, nil
}

func planMaterialFrom(value plan.Plan) planMaterial {
	material := planMaterial{
		Name: value.Name(),
		Goal: value.Goal().Text(),
	}
	for _, condition := range value.AcceptanceConditions() {
		material.AcceptanceConditions = append(material.AcceptanceConditions, condition.Statement())
	}
	for _, task := range value.Tasks() {
		material.Tasks = append(material.Tasks, task.Name())
	}
	if targetDate, ok := value.TargetDate(); ok {
		encoded := targetDate.String()
		material.TargetDate = &encoded
	}
	return material
}

func decodeAssessmentOutput(current plan.Plan, text string) (plancontrol.AssessorResponse, error) {
	fields, err := decodeStrictJSONObject([]byte(text), "outcome", "proposedPlan")
	if err != nil {
		return plancontrol.AssessorResponse{}, plancontrol.ErrUntranslatableAIResponse
	}
	var outcome string
	if err := json.Unmarshal(fields["outcome"], &outcome); err != nil {
		return plancontrol.AssessorResponse{}, plancontrol.ErrUntranslatableAIResponse
	}
	proposedPlan := fields["proposedPlan"]

	switch outcome {
	case "complete":
		return responseWithoutProposal(plancontrol.Complete, proposedPlan)
	case "retain":
		return responseWithoutProposal(plancontrol.Retain, proposedPlan)
	case "insufficient_information":
		return responseWithoutProposal(plancontrol.InsufficientInformation, proposedPlan)
	case "revise":
		proposed, err := decodeProposedPlan(proposedPlan)
		if err != nil || current.Equal(proposed) {
			return plancontrol.AssessorResponse{}, plancontrol.ErrUntranslatableAIResponse
		}
		return plancontrol.AssessorResponse{
			Claims:        []plancontrol.Outcome{plancontrol.Revise},
			ProposedPlans: []plan.Plan{proposed},
		}, nil
	default:
		return plancontrol.AssessorResponse{}, plancontrol.ErrUntranslatableAIResponse
	}
}

func responseWithoutProposal(outcome plancontrol.Outcome, rawProposal json.RawMessage) (plancontrol.AssessorResponse, error) {
	if !bytes.Equal(bytes.TrimSpace(rawProposal), []byte("null")) {
		return plancontrol.AssessorResponse{}, plancontrol.ErrUntranslatableAIResponse
	}
	return plancontrol.AssessorResponse{Claims: []plancontrol.Outcome{outcome}}, nil
}

func decodeProposedPlan(encoded json.RawMessage) (plan.Plan, error) {
	fields, err := decodeStrictJSONObject(
		encoded,
		"name",
		"goal",
		"acceptanceConditions",
		"tasks",
		"targetDate",
	)
	if err != nil {
		return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
	}
	var material planMaterial
	if json.Unmarshal(fields["name"], &material.Name) != nil ||
		json.Unmarshal(fields["goal"], &material.Goal) != nil ||
		json.Unmarshal(fields["acceptanceConditions"], &material.AcceptanceConditions) != nil ||
		json.Unmarshal(fields["tasks"], &material.Tasks) != nil ||
		json.Unmarshal(fields["targetDate"], &material.TargetDate) != nil {
		return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
	}

	goal, err := plan.NewGoal(material.Goal)
	if err != nil {
		return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
	}
	conditions := make([]plan.AcceptanceCondition, 0, len(material.AcceptanceConditions))
	for _, statement := range material.AcceptanceConditions {
		condition, err := plan.NewAcceptanceCondition(statement)
		if err != nil {
			return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
		}
		conditions = append(conditions, condition)
	}
	tasks := make([]plan.Task, 0, len(material.Tasks))
	for _, name := range material.Tasks {
		task, err := plan.NewTask(name)
		if err != nil {
			return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
		}
		tasks = append(tasks, task)
	}
	var targetDate *plan.TargetDate
	if material.TargetDate != nil {
		parsed, err := plan.ParseTargetDate(*material.TargetDate)
		if err != nil {
			return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
		}
		targetDate = &parsed
	}
	proposed, err := plan.New(material.Name, goal, conditions, tasks, targetDate)
	if err != nil {
		return plan.Plan{}, plancontrol.ErrUntranslatableAIResponse
	}
	return proposed, nil
}

func decodeStrictJSONObject(encoded []byte, fieldNames ...string) (map[string]json.RawMessage, error) {
	allowed := make(map[string]struct{}, len(fieldNames))
	for _, name := range fieldNames {
		allowed[name] = struct{}{}
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, plancontrol.ErrUntranslatableAIResponse
	}
	fields := make(map[string]json.RawMessage, len(fieldNames))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, plancontrol.ErrUntranslatableAIResponse
		}
		name, ok := token.(string)
		if !ok {
			return nil, plancontrol.ErrUntranslatableAIResponse
		}
		if _, ok := allowed[name]; !ok {
			return nil, plancontrol.ErrUntranslatableAIResponse
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, plancontrol.ErrUntranslatableAIResponse
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, plancontrol.ErrUntranslatableAIResponse
		}
		fields[name] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, plancontrol.ErrUntranslatableAIResponse
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, plancontrol.ErrUntranslatableAIResponse
	}
	if len(fields) != len(fieldNames) {
		return nil, plancontrol.ErrUntranslatableAIResponse
	}
	return fields, nil
}
