package githubplan

import (
	"net/url"
	"sort"
	"strconv"

	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/representation"
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

func (s scheme) creationRequests(value plan.Plan) ([]Request, bool) {
	tasks := value.Tasks()
	narrative, representable := s.narrativeFor(value)
	if !representable {
		return nil, false
	}
	if s == issueScheme {
		if len(tasks) > 100 {
			return nil, false
		}
		construction := newRequestPlanConstruction(1 + len(tasks)*2)
		parent := construction.addIssue(CreateIssueRequest{title: value.Name(), body: narrative, hasBody: true})
		for _, task := range tasks {
			child := construction.addIssue(CreateIssueRequest{title: task.Name()})
			construction.addSubIssue(parent.number, child.id)
		}
		return construction.snapshot(), true
	}

	construction := newRequestPlanConstruction(1 + len(tasks))
	milestone := CreateMilestoneRequest{title: value.Name(), description: narrative}
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

	narrative := s.establishNarrativeFacts(facts.root.fact.content)
	goal, conditions := narrativeObservations(narrative)
	titles, complete := s.taskTitles(facts.tasks)
	tasks := planrepresentation.IncompleteTasks(titles)
	if complete {
		tasks = planrepresentation.CompleteTasks(titles)
	}

	targetDate := planrepresentation.UnavailableTargetDate()
	if s == milestoneScheme {
		state, value := facts.root.fact.date.targetDateText()
		switch state {
		case dateAbsent:
			targetDate = planrepresentation.AbsentTargetDate()
		case datePresent:
			targetDate = planrepresentation.ClassifyTargetDate(value)
		}
	} else {
		targetDate = narrativeTargetDateObservation(narrative)
	}
	return planrepresentation.NewPresentObservation(name, goal, conditions, tasks, targetDate)
}

func (s scheme) taskTitles(set taskFactSet) ([]string, bool) {
	titles := make([]string, 0, len(set.members))
	complete := set.isComplete()
	for _, member := range s.taskMembersInPlanOrder(set) {
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

func (s scheme) taskMembersInPlanOrder(set taskFactSet) []taskItemFact {
	members := append([]taskItemFact(nil), set.members...)
	if s == milestoneScheme {
		sort.Slice(members, func(left, right int) bool {
			return members[left].id < members[right].id
		})
	}
	return members
}

func narrativeObservations(facts narrativeFacts) (planrepresentation.GoalObservation, planrepresentation.AcceptanceConditionsObservation) {
	goal := planrepresentation.UnavailableGoal()
	if facts.goal.available {
		goal = planrepresentation.ClassifyGoal(facts.goal.value)
	}
	conditions := planrepresentation.IncompleteAcceptanceConditions(facts.conditions.members)
	if facts.conditions.complete {
		conditions = planrepresentation.CompleteAcceptanceConditions(facts.conditions.members)
	}
	return goal, conditions
}

func narrativeTargetDateObservation(facts narrativeFacts) planrepresentation.TargetDateObservation {
	switch facts.targetDate.state {
	case narrativeTargetAbsent:
		return planrepresentation.AbsentTargetDate()
	case narrativeTargetPresent:
		return planrepresentation.ClassifyTargetDate(facts.targetDate.value)
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
