package main

import (
	"errors"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
)

type targetRecord struct {
	Kind string `json:"kind"`
	Key  string `json:"key"`
}
type planRecord struct {
	Name                 string   `json:"name"`
	Goal                 string   `json:"goal"`
	AcceptanceConditions []string `json:"acceptanceConditions"`
	Tasks                []string `json:"tasks"`
	TargetDate           *string  `json:"targetDate"`
}
type taskProgressRecord struct {
	Name  string `json:"name"`
	State string `json:"state"`
}
type progressRecord struct {
	OverallState       string               `json:"overallState"`
	MembershipComplete bool                 `json:"membershipComplete"`
	Tasks              []taskProgressRecord `json:"tasks"`
}
type assessmentValue struct {
	Outcome      string      `json:"outcome"`
	AssessedPlan planRecord  `json:"assessedPlan"`
	ProposedPlan *planRecord `json:"proposedPlan"`
}
type buildEvidence struct {
	GoVersion string `json:"goVersion"`
	Version   string `json:"version"`
	Revision  string `json:"revision"`
	Modified  string `json:"modified"`
}
type acquisitionRecord struct {
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
}
type provenanceRecord struct {
	TargetURL             string             `json:"targetURL"`
	Snapshot              *acquisitionRecord `json:"snapshotAcquisition"`
	Delivery              *acquisitionRecord `json:"deliveryAcquisition"`
	GitHubAPIVersion      string             `json:"githubAPIVersion"`
	CodexSupportedVersion string             `json:"codexSupportedVersion"`
	// The SDK does not expose observed version metadata. Null is not an assertion
	// that no version validation occurred; it is explicitly unavailable evidence.
	CodexObservedVersion *string       `json:"codexObservedVersion"`
	Model                string        `json:"model"`
	Effort               string        `json:"effort"`
	Executable           string        `json:"executable"`
	WorkingDirectory     string        `json:"workingDirectory"`
	ShutdownGrace        string        `json:"shutdownGrace"`
	Build                buildEvidence `json:"build"`
}
type assessmentRecord struct {
	Type       string           `json:"type"`
	Target     targetRecord     `json:"target"`
	Assessment assessmentValue  `json:"assessment"`
	Provenance provenanceRecord `json:"provenance"`
}
type processedRecord struct {
	Type           string           `json:"type"`
	Target         targetRecord     `json:"target"`
	Classification string           `json:"classification"`
	CurrentPlan    *planRecord      `json:"currentPlan"`
	Progress       *progressRecord  `json:"progress"`
	Assessment     *assessmentValue `json:"assessment"`
	Failure        string           `json:"failure"`
	Directive      string           `json:"directive"`
	Provenance     provenanceRecord `json:"provenance"`
}

func identityRecord(identity controlruntime.TargetIdentity) targetRecord {
	return targetRecord{identity.Kind(), identity.Key()}
}
func encodePlan(value plan.Plan) planRecord {
	result := planRecord{Name: value.Name(), Goal: value.Goal().Text(), AcceptanceConditions: []string{}, Tasks: []string{}}
	for _, condition := range value.AcceptanceConditions() {
		result.AcceptanceConditions = append(result.AcceptanceConditions, condition.Statement())
	}
	for _, task := range value.Tasks() {
		result.Tasks = append(result.Tasks, task.Name())
	}
	if date, ok := value.TargetDate(); ok {
		text := date.String()
		result.TargetDate = &text
	}
	return result
}
func encodeProgress(value plansnapshot.ProgressEvidence) progressRecord {
	result := progressRecord{OverallState: progressStateText(value.OverallState()), MembershipComplete: value.MembershipComplete(), Tasks: []taskProgressRecord{}}
	for _, task := range value.Tasks() {
		result.Tasks = append(result.Tasks, taskProgressRecord{task.Name(), progressStateText(task.State())})
	}
	return result
}
func progressStateText(state plansnapshot.ProgressState) string {
	switch state {
	case plansnapshot.Open:
		return "open"
	case plansnapshot.Closed:
		return "closed"
	default:
		return "unknown"
	}
}
func encodeAssessment(value plancontrol.Assessment) assessmentValue {
	names := map[plancontrol.Outcome]string{plancontrol.Complete: "complete", plancontrol.Retain: "retain", plancontrol.Revise: "revise", plancontrol.InsufficientInformation: "insufficient_information"}
	result := assessmentValue{Outcome: names[value.Outcome()], AssessedPlan: encodePlan(value.AssessedPlan())}
	if proposed, ok := value.ProposedPlan(); ok {
		record := encodePlan(proposed)
		result.ProposedPlan = &record
	}
	return result
}
func encodeProcessedReport(report controlruntime.Report[planattempt.AttemptResult], provenance provenanceRecord) (processedRecord, int) {
	result := processedRecord{Type: "processed_report", Target: identityRecord(report.Target()), Provenance: provenance}
	if failure, ok := report.Failure(); ok {
		result.Classification = "attempt_failure"
		result.Failure = stableFailure(failure)
		return result, 1
	}
	completed, directive, ok := report.Completion()
	if !ok {
		result.Classification = "invalid_report"
		return result, 1
	}
	if directive.Kind() == controlruntime.AwaitRequest {
		result.Directive = "await_request"
	}
	snapshot := completed.Snapshot()
	if current, ok := snapshot.CurrentPlan(); ok {
		record := encodePlan(current)
		result.CurrentPlan = &record
	}
	if progress, ok := snapshot.Progress(); ok {
		record := encodeProgress(progress)
		result.Progress = &record
	}
	if assessment, ok := completed.Assessment(); ok {
		record := encodeAssessment(assessment)
		result.Assessment = &record
		result.Classification = "assessed"
		return result, 0
	}
	result.Classification = "current_plan_not_established"
	return result, 2
}
func stableFailure(failure controlruntime.Failure) string {
	if failure.Kind() == controlruntime.ControlDirectiveRejected {
		return "control_directive_rejected"
	}
	var attempt *planattempt.FailureError
	if errors.As(failure, &attempt) {
		return string(attempt.Code())
	}
	var assessment *plancontrol.FailureError
	if errors.As(failure, &assessment) {
		return string(assessment.Code())
	}
	var snapshot plansnapshot.ObservationFailure
	if errors.As(failure, &snapshot) && snapshot.Code() == plansnapshot.ObservationUnavailable {
		return "observation_unavailable"
	}
	return "target_attempt_failed"
}
func runtimeBuildEvidence() buildEvidence {
	result := buildEvidence{GoVersion: runtime.Version(), Version: "unavailable", Revision: "unavailable", Modified: "unavailable"}
	if info, ok := debug.ReadBuildInfo(); ok {
		result.Version = info.Main.Version
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				result.Revision = setting.Value
			case "vcs.modified":
				result.Modified = setting.Value
			}
		}
	}
	return result
}
