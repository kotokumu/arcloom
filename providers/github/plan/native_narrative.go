package githubplan

import (
	"fmt"
	"strings"

	"github.com/kotokumu/arcloom/controllers/plan"
)

const (
	narrativeGoalHeading       = "## Goal"
	narrativeConditionsHeading = "## Acceptance Conditions"
	narrativeTargetDateHeading = "## Target Date"
	narrativeMemberPrefix      = "### "
)

type narrativeValueKind uint8

const (
	narrativeGoalValue narrativeValueKind = iota + 1
	narrativeConditionValue
)

type narrativeConditionsBoundary uint8

const (
	narrativeEndBoundary narrativeConditionsBoundary = iota + 1
	narrativeTargetBoundary
	narrativeIncompleteBoundary
)

type narrativeTextFact struct {
	value     string
	available bool
}

type narrativeConditionsFacts struct {
	members  []string
	complete bool
}

type narrativeTargetState uint8

const (
	narrativeTargetUnavailable narrativeTargetState = iota
	narrativeTargetAbsent
	narrativeTargetPresent
)

type narrativeTargetFact struct {
	state narrativeTargetState
	value string
}

type narrativeFacts struct {
	goal       narrativeTextFact
	conditions narrativeConditionsFacts
	targetDate narrativeTargetFact
}

type narrativeLine struct {
	start      int
	end        int
	terminated bool
}

func (s scheme) narrativeFor(value plan.Plan) (string, bool) {
	if hasReservedNarrativeLine(value.Goal().Text(), narrativeGoalValue) {
		return "", false
	}
	for _, condition := range value.AcceptanceConditions() {
		if hasReservedNarrativeLine(condition.Statement(), narrativeConditionValue) {
			return "", false
		}
	}
	var builder strings.Builder
	builder.WriteString(narrativeGoalHeading + "\n\n")
	builder.WriteString(value.Goal().Text())
	builder.WriteString("\n\n" + narrativeConditionsHeading)
	for index, condition := range value.AcceptanceConditions() {
		fmt.Fprintf(&builder, "\n\n%s%d\n\n", narrativeMemberPrefix, index+1)
		builder.WriteString(condition.Statement())
	}
	builder.WriteByte('\n')
	if s == issueScheme {
		if date, ok := value.TargetDate(); ok {
			builder.WriteString("\n" + narrativeTargetDateHeading + "\n\n")
			builder.WriteString(date.String())
			builder.WriteByte('\n')
		}
	}
	return builder.String(), true
}

func hasReservedNarrativeLine(value string, kind narrativeValueKind) bool {
	for _, line := range strings.Split(value, "\n") {
		if line == narrativeGoalHeading || line == narrativeConditionsHeading || line == narrativeTargetDateHeading {
			return true
		}
		if kind == narrativeConditionValue && strings.HasPrefix(line, narrativeMemberPrefix) {
			return true
		}
	}
	return false
}

func (s scheme) establishNarrativeFacts(content textFact) narrativeFacts {
	if !content.available {
		return narrativeFacts{}
	}
	source := content.value
	goalLines := exactNarrativeLines(source, narrativeGoalHeading)
	conditionLines := exactNarrativeLines(source, narrativeConditionsHeading)
	targetLines := exactNarrativeLines(source, narrativeTargetDateHeading)
	if len(goalLines) != 1 || len(conditionLines) != 1 || goalLines[0].start >= conditionLines[0].start {
		return narrativeFacts{}
	}
	if s == milestoneScheme && len(targetLines) != 0 {
		return narrativeFacts{}
	}
	if s == issueScheme && (len(targetLines) > 1 || len(targetLines) == 1 && targetLines[0].start <= conditionLines[0].start) {
		return narrativeFacts{}
	}

	goalOpening := narrativeGoalHeading + "\n\n"
	goalStart := goalLines[0].start + len(goalOpening)
	if !strings.HasPrefix(source[goalLines[0].start:], goalOpening) ||
		goalStart > conditionLines[0].start ||
		!strings.HasSuffix(source[goalStart:conditionLines[0].start], "\n\n") {
		return narrativeFacts{}
	}
	goalFramed := source[goalStart:conditionLines[0].start]
	facts := narrativeFacts{
		goal: narrativeTextFact{value: goalFramed[:len(goalFramed)-2], available: true},
	}

	conditionsStart := conditionLines[0].start + len(narrativeConditionsHeading)
	conditionsEnd := len(source)
	hasTarget := s == issueScheme && len(targetLines) == 1
	if hasTarget {
		conditionsEnd = targetLines[0].start
	}
	boundary := narrativeEndBoundary
	if hasTarget {
		boundary = narrativeIncompleteBoundary
		if strings.HasPrefix(source[conditionsEnd:], narrativeTargetDateHeading+"\n\n") {
			boundary = narrativeTargetBoundary
		}
	}
	facts.conditions = establishNarrativeConditions(source[conditionsStart:conditionsEnd], boundary)

	if s != issueScheme {
		return facts
	}
	if !hasTarget {
		facts.targetDate.state = narrativeTargetAbsent
		return facts
	}
	target := targetLines[0]
	if !strings.HasSuffix(source[:target.start], "\n\n") ||
		boundary != narrativeTargetBoundary ||
		!strings.HasSuffix(source, "\n") {
		return facts
	}
	dateStart := target.start + len(narrativeTargetDateHeading+"\n\n")
	if dateStart > len(source)-1 {
		return facts
	}
	facts.targetDate = narrativeTargetFact{
		state: narrativeTargetPresent,
		value: source[dateStart : len(source)-1],
	}
	return facts
}

func establishNarrativeConditions(section string, boundary narrativeConditionsBoundary) narrativeConditionsFacts {
	zeroForm := "\n"
	closing := "\n"
	if boundary == narrativeTargetBoundary {
		zeroForm = "\n\n"
		closing = "\n\n"
	}
	if boundary != narrativeIncompleteBoundary && section == zeroForm {
		return narrativeConditionsFacts{complete: true}
	}
	markers := narrativeMemberLines(section)
	if len(markers) == 0 || markers[0].start != 2 {
		return narrativeConditionsFacts{}
	}
	members := make([]string, 0, len(markers))
	for index, marker := range markers {
		if !marker.terminated || section[marker.start:marker.end] != fmt.Sprintf("%s%d", narrativeMemberPrefix, index+1) ||
			marker.end+1 >= len(section) || section[marker.end+1] != '\n' {
			return narrativeConditionsFacts{members: members}
		}
		memberStart := marker.end + 2
		memberEnd := len(section)
		memberClosing := closing
		if index+1 < len(markers) {
			memberEnd = markers[index+1].start
			memberClosing = "\n\n"
		} else if boundary == narrativeIncompleteBoundary {
			return narrativeConditionsFacts{members: members}
		}
		framed := section[memberStart:memberEnd]
		if !strings.HasSuffix(framed, memberClosing) {
			return narrativeConditionsFacts{members: members}
		}
		members = append(members, framed[:len(framed)-len(memberClosing)])
	}
	return narrativeConditionsFacts{members: members, complete: true}
}

func exactNarrativeLines(source, exact string) []narrativeLine {
	lines := narrativeLines(source)
	result := make([]narrativeLine, 0, 1)
	for _, line := range lines {
		if source[line.start:line.end] == exact {
			result = append(result, line)
		}
	}
	return result
}

func narrativeMemberLines(source string) []narrativeLine {
	lines := narrativeLines(source)
	result := make([]narrativeLine, 0)
	for _, line := range lines {
		if strings.HasPrefix(source[line.start:line.end], narrativeMemberPrefix) {
			result = append(result, line)
		}
	}
	return result
}

func narrativeLines(source string) []narrativeLine {
	lines := make([]narrativeLine, 0, strings.Count(source, "\n")+1)
	for start := 0; ; {
		offset := strings.IndexByte(source[start:], '\n')
		if offset < 0 {
			lines = append(lines, narrativeLine{start: start, end: len(source)})
			return lines
		}
		end := start + offset
		lines = append(lines, narrativeLine{start: start, end: end, terminated: true})
		start = end + 1
		if start == len(source) {
			lines = append(lines, narrativeLine{start: start, end: start})
			return lines
		}
	}
}
