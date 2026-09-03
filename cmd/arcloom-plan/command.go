package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/kotokumu/arcloom/arcloom/controlruntime"
	"github.com/kotokumu/arcloom/controllers/plan/assessmentdelivery"
	"github.com/kotokumu/arcloom/controllers/plan/attempt"
	"github.com/kotokumu/arcloom/controllers/plan/control"
	"github.com/kotokumu/arcloom/controllers/plan/snapshot"
	"github.com/kotokumu/arcloom/internal/planhost"
	"github.com/kotokumu/arcloom/providers/codex/appserver"
	"github.com/kotokumu/arcloom/providers/codex/plancontrol"
	"github.com/kotokumu/arcloom/providers/github/plan"
)

// commandDependencies are concrete outer adapters, not a product Port. Each
// invocation owns its collaborators; output and file reads honor cancellation.
type commandDependencies struct {
	httpClient     *http.Client
	newCodexClient func(string, time.Duration) (codexappserver.Client, error)
	writeRecord    func(context.Context, any) error
	readEvidence   func(context.Context, string) (string, error)
	now            func() time.Time
	build          buildEvidence
}

type commandOptions struct {
	owner, repository, executable, model, effort, workdir, evidencePath string
	milestone                                                           int64
	shutdown                                                            time.Duration
}

func parseOptions(arguments []string) (commandOptions, error) {
	var options commandOptions
	flags := flag.NewFlagSet("arcloom-plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Never echo unknown arguments or credential values.
	flags.StringVar(&options.owner, "owner", "", "GitHub owner")
	flags.StringVar(&options.repository, "repo", "", "GitHub repository")
	flags.Int64Var(&options.milestone, "milestone", 0, "positive milestone number")
	flags.StringVar(&options.executable, "codex", "", "absolute Codex executable")
	flags.DurationVar(&options.shutdown, "shutdown-grace", 0, "positive SDK shutdown bound")
	flags.StringVar(&options.model, "model", "", "Codex model")
	flags.StringVar(&options.effort, "effort", "", "Codex reasoning effort")
	flags.StringVar(&options.workdir, "workdir", "", "existing absolute working directory")
	flags.StringVar(&options.evidencePath, "delivery-evidence", "", "optional current UTF-8 operator evidence")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return commandOptions{}, errors.New("invalid command options")
	}
	return options, nil
}

func runCommand(ctx context.Context, arguments []string, deps commandDependencies) int {
	if ctx == nil {
		return 1
	}
	if ctx.Err() != nil {
		return 130
	}
	options, err := parseOptions(arguments)
	if err != nil {
		return 1
	}
	if deps.httpClient == nil || deps.newCodexClient == nil || deps.writeRecord == nil || deps.readEvidence == nil || deps.now == nil {
		return 1
	}
	repository, err := githubplan.NewRepository(options.owner, options.repository)
	if err != nil {
		return 1
	}
	number, err := githubplan.NewResourceNumber(options.milestone)
	if err != nil {
		return 1
	}
	milestone, err := githubplan.NewMilestoneTarget(repository, number)
	if err != nil {
		return 1
	}
	observer, err := githubplan.NewMilestoneSnapshotObserver(deps.httpClient, milestone)
	if err != nil {
		return 1
	}
	model, err := codexplancontrol.NewModel(options.model)
	if err != nil {
		return 1
	}
	effort, err := codexplancontrol.NewReasoningEffort(options.effort)
	if err != nil {
		return 1
	}
	directory, err := codexplancontrol.NewWorkingDirectory(options.workdir)
	if err != nil {
		return 1
	}
	client, err := deps.newCodexClient(options.executable, options.shutdown)
	if err != nil {
		return 1
	}
	assessor, err := codexplancontrol.NewAssessor(client, codexplancontrol.NewConfiguration(model, effort, directory), encodeDeliveryObservation)
	if err != nil {
		return 1
	}
	key := repository.Owner() + "/" + repository.Name() + "/milestones/" + strconv.FormatInt(options.milestone, 10)
	identity, err := controlruntime.NewTargetIdentity("github-milestone", key)
	if err != nil {
		return 1
	}
	provenance := provenanceRecord{
		TargetURL:        "https://github.com/" + repository.Owner() + "/" + repository.Name() + "/milestone/" + strconv.FormatInt(options.milestone, 10),
		GitHubAPIVersion: githubplan.RESTAPIVersion, CodexSupportedVersion: codexappserver.SupportedCodexVersion,
		Model: options.model, Effort: options.effort, Executable: options.executable, WorkingDirectory: options.workdir, ShutdownGrace: options.shutdown.String(), Build: deps.build,
	}
	target, err := bindObservations(identity, observer, options.evidencePath, deps, &provenance)
	if err != nil {
		return 1
	}
	binding, err := planattempt.NewPlanAttemptBinding(target, assessor)
	if err != nil {
		return 1
	}
	recipient := assessmentdelivery.Recipient(func(ctx context.Context, target controlruntime.TargetIdentity, assessment plancontrol.Assessment) error {
		return deps.writeRecord(ctx, assessmentRecord{Type: "assessment", Target: identityRecord(target), Assessment: encodeAssessment(assessment), Provenance: provenance})
	})
	lifecycle, cancel := context.WithCancel(ctx)
	defer cancel()
	host, err := planhost.Start(lifecycle, binding, recipient)
	if err != nil {
		return commandExit(ctx, 1)
	}
	defer func() { cancel(); _ = host.Wait() }()
	if err := host.Trigger(ctx); err != nil {
		return commandExit(ctx, 1)
	}
	report, ok := <-host.Reports()
	if !ok {
		return commandExit(ctx, 1)
	}
	record, exit := encodeProcessedReport(report, provenance)
	if err := deps.writeRecord(ctx, record); err != nil {
		return commandExit(ctx, 1)
	}
	return commandExit(ctx, exit)
}

func commandExit(ctx context.Context, normal int) int {
	if ctx.Err() != nil {
		return 130
	}
	return normal
}

type deliveryObservation struct {
	Progress         progressRecord `json:"progress"`
	OperatorEvidence *string        `json:"operatorEvidence"`
}

func encodeDeliveryObservation(ctx context.Context, observed deliveryObservation) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(observed)
	return string(encoded), err
}

func bindObservations(identity controlruntime.TargetIdentity, observer plansnapshot.Observer, evidencePath string, deps commandDependencies, provenance *provenanceRecord) (planattempt.PlanTarget[deliveryObservation], error) {
	snapshotObserver := func(ctx context.Context) (plansnapshot.Snapshot, error) {
		acquisition := &acquisitionRecord{StartedAt: deps.now().UTC()}
		snapshot, err := observer(ctx)
		acquisition.CompletedAt = deps.now().UTC()
		provenance.Snapshot = acquisition
		return snapshot, err
	}
	deliveryObserver := func(ctx context.Context) (deliveryObservation, error) {
		acquisition := &acquisitionRecord{StartedAt: deps.now().UTC()}
		defer func() { acquisition.CompletedAt = deps.now().UTC(); provenance.Delivery = acquisition }()
		// Reacquire current external facts AFTER Snapshot; never reuse that Snapshot.
		current, err := observer(ctx)
		if err != nil {
			return deliveryObservation{}, err
		}
		progress, ok := current.Progress()
		if !ok {
			return deliveryObservation{}, errors.New("current progress unavailable")
		}
		observed := deliveryObservation{Progress: encodeProgress(progress)}
		if evidencePath != "" {
			evidence, err := deps.readEvidence(ctx, evidencePath)
			if err != nil {
				return deliveryObservation{}, err
			}
			if !utf8.ValidString(evidence) {
				return deliveryObservation{}, errors.New("operator evidence is not UTF-8")
			}
			observed.OperatorEvidence = &evidence
		}
		return observed, ctx.Err()
	}
	return planattempt.NewPlanTarget(identity, snapshotObserver, deliveryObserver)
}
