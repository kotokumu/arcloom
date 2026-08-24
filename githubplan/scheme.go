package githubplan

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kotokumu/arcloom/plan"
	"github.com/kotokumu/arcloom/planrepresentation"
)

// scheme is the closed GitHub-native correspondence selected at construction.
// It owns the differences between Milestone and Issue facts and their Plan
// locations; provider reads and page coherence remain separate concerns.
type scheme uint8

const (
	milestoneScheme scheme = iota + 1
	issueScheme
)

func schemeFor(representation Representation) (scheme, bool) {
	switch representation {
	case MilestoneRepresentation:
		return milestoneScheme, true
	case IssueRepresentation:
		return issueScheme, true
	default:
		return 0, false
	}
}

func (s scheme) payloadShape() payloadShape {
	if s == issueScheme {
		return payloadWithTargetDate
	}
	return payloadWithoutTargetDate
}

func (s scheme) creationRequests(value plan.Plan) ([]Request, bool) {
	tasks := value.Tasks()
	if s == issueScheme {
		if len(tasks) > 100 {
			return nil, false
		}
		construction := newRequestPlanConstruction(1 + len(tasks)*2)
		parent := construction.addIssue(CreateIssueRequest{title: value.Name(), body: s.narrative(value), hasBody: true})
		for _, task := range tasks {
			child := construction.addIssue(CreateIssueRequest{title: task.Name()})
			construction.addSubIssue(parent.number, child.id)
		}
		return construction.snapshot(), true
	}

	construction := newRequestPlanConstruction(1 + len(tasks))
	milestone := CreateMilestoneRequest{title: value.Name(), description: s.narrative(value)}
	if date, ok := value.TargetDate(); ok {
		milestone.dueOn = date.String() + "T00:00:00Z"
		milestone.hasDueOn = true
	}
	milestoneResult := construction.addMilestone(milestone)
	for _, task := range tasks {
		construction.addIssue(CreateIssueRequest{title: task.Name(), milestone: milestoneResult.reference(), hasMilestone: true})
	}
	return construction.snapshot(), true
}

func (s scheme) narrative(value plan.Plan) string {
	var builder strings.Builder
	builder.WriteString(payloadFor(value, s.payloadShape()))
	builder.WriteString("## Goal\n\n")
	builder.WriteString(value.Goal().Text())
	builder.WriteString("\n\n## Acceptance Conditions")
	for index, condition := range value.AcceptanceConditions() {
		fmt.Fprintf(&builder, "\n\n### %d\n\n", index+1)
		builder.WriteString(condition.Statement())
	}
	builder.WriteByte('\n')
	if s == issueScheme {
		if date, ok := value.TargetDate(); ok {
			builder.WriteString("\n## Target Date\n\n")
			builder.WriteString(date.String())
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}

func (s scheme) rootURL(binding observerBinding) *url.URL {
	suffix := "/issues/" + strconv.FormatInt(binding.number.value, 10)
	if s == milestoneScheme {
		suffix = "/milestones/" + strconv.FormatInt(binding.number.value, 10)
	}
	return binding.resourceURL(suffix)
}

func (s scheme) taskURL(binding observerBinding) *url.URL {
	if s == milestoneScheme {
		target := binding.resourceURL("/issues")
		query := target.Query()
		query.Set("milestone", strconv.FormatInt(binding.number.value, 10))
		query.Set("state", "all")
		query.Set("per_page", "100")
		query.Set("page", "1")
		target.RawQuery = query.Encode()
		return target
	}
	target := binding.resourceURL("/issues/" + strconv.FormatInt(binding.number.value, 10) + "/sub_issues")
	query := target.Query()
	query.Set("per_page", "100")
	query.Set("page", "1")
	target.RawQuery = query.Encode()
	return target
}

func (s scheme) decodeRoot(document responseDocument, binding observerBinding) rootFactOutcome {
	if s == milestoneScheme {
		return decodeMilestoneRoot(document, binding.number)
	}
	return decodeIssueRoot(document, binding.number)
}

func (s scheme) project(facts githubFactSet) (planrepresentation.Observation, error) {
	if facts.binding.scheme != s || !facts.root.available || facts.root.fact.number != facts.binding.number.value {
		return planrepresentation.UnavailableObservation(), nil
	}
	name := planrepresentation.UnavailablePlanName()
	if facts.root.fact.title.available {
		name = planrepresentation.ClassifyPlanName(facts.root.fact.title.value)
	}

	payload := unavailablePayloadOutcome()
	if facts.root.fact.content.available {
		payload = decodePayload(facts.root.fact.content.value, s.payloadShape())
	}
	goal, conditions := payloadObservations(payload)
	titles, complete := s.taskTitles(facts.tasks)
	tasks := planrepresentation.IncompleteTasks(titles)
	if complete {
		tasks = planrepresentation.CompleteTasks(titles)
	}

	targetDate := planrepresentation.UnavailableTargetDate()
	if s == milestoneScheme {
		switch facts.root.fact.date.state {
		case dateAbsent:
			targetDate = planrepresentation.AbsentTargetDate()
		case datePresent:
			value := facts.root.fact.date.value
			if len(value) == len("2006-01-02T00:00:00Z") && strings.HasSuffix(value, "T00:00:00Z") {
				value = value[:10]
			}
			targetDate = planrepresentation.ClassifyTargetDate(value)
		}
	} else {
		targetDate = payloadTargetDateObservation(payload)
	}
	return planrepresentation.NewPresentObservation(name, goal, conditions, tasks, targetDate)
}

func (s scheme) taskTitles(set taskFactSet) ([]string, bool) {
	titles := make([]string, 0, len(set.members))
	complete := set.isComplete()
	for _, member := range set.members {
		if member.pullRequest {
			if s == issueScheme {
				complete = false
			}
			continue
		}
		titles = append(titles, member.title)
	}
	return titles, complete
}

func payloadObservations(payload payloadOutcome) (planrepresentation.GoalObservation, planrepresentation.AcceptanceConditionsObservation) {
	goal := planrepresentation.UnavailableGoal()
	if payload.goal.available {
		goal = planrepresentation.ClassifyGoal(payload.goal.value)
	}
	conditions := planrepresentation.IncompleteAcceptanceConditions(nil)
	if payload.conditions.available {
		if payload.conditions.complete {
			conditions = planrepresentation.CompleteAcceptanceConditions(payload.conditions.members)
		} else {
			conditions = planrepresentation.IncompleteAcceptanceConditions(payload.conditions.members)
		}
	}
	return goal, conditions
}

func payloadTargetDateObservation(payload payloadOutcome) planrepresentation.TargetDateObservation {
	switch payload.targetDate.state {
	case payloadTargetAbsent:
		return planrepresentation.AbsentTargetDate()
	case payloadTargetPresent:
		return planrepresentation.ClassifyTargetDate(payload.targetDate.value)
	default:
		return planrepresentation.UnavailableTargetDate()
	}
}

func (b observerBinding) resourceURL(suffix string) *url.URL {
	return &url.URL{
		Scheme: "https",
		Host:   "api.github.com",
		Path:   "/repos/" + b.repository.owner + "/" + b.repository.name + suffix,
	}
}
