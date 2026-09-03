package main

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/kotokumu/arcloom/providers/codex/appserver"
)

func TestCredentialTransportOnlyAuthorizesExactGitHubOrigin(t *testing.T) {
	for _, target := range []string{"https://api.github.com/repos/owner/repo/milestones/7", "https://example.com/private", "http://api.github.com/private", "https://api.github.com:444/private"} {
		t.Run(target, func(t *testing.T) {
			original, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
			if err != nil {
				t.Fatal(err)
			}
			authorized, err := authorizeGitHubRequest(original, "fake-test-token")
			if target != "https://api.github.com/repos/owner/repo/milestones/7" {
				if err == nil || authorized != nil {
					t.Fatal("credentials can leave selected origin")
				}
				return
			}
			if err != nil || authorized.Header.Get("Authorization") != "Bearer fake-test-token" || original.Header.Get("Authorization") != "" {
				t.Fatal("credential injection or request mutation")
			}
			if authorized.Context() != original.Context() {
				t.Fatal("cancellation lost")
			}
		})
	}
}

func TestCommandCancellationBeforeConfiguration(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := runCommand(ctx, nil, commandDependencies{}); got != 130 {
		t.Fatal(got)
	}
	var missing context.Context
	if got := runCommand(missing, nil, commandDependencies{}); got != 1 {
		t.Fatal(got)
	}
}

func TestRuntimeBuildEvidenceLabelsUnavailableFacts(t *testing.T) {
	evidence := runtimeBuildEvidence()
	if evidence.GoVersion == "" || evidence.Revision == "" || evidence.Modified == "" {
		t.Fatal("unlabelled provenance")
	}
}

func TestCommandSDKConfigurationFailureStartsNoObservation(t *testing.T) {
	fixture := &commandFixture{}
	deps := fixture.dependencies()
	calls := 0
	deps.newCodexClient = func(executable string, grace time.Duration) (codexappserver.Client, error) {
		calls++
		if executable != "/exact/codex" || grace != 2*time.Second {
			t.Error("SDK configuration changed")
		}
		return nil, errors.New("SDK rejects unsafe configuration")
	}
	flags := []string{"--owner", "owner", "--repo", "repo", "--milestone", "7", "--codex", "/exact/codex", "--shutdown-grace", "2s", "--model", "model", "--effort", "high", "--workdir", t.TempDir()}
	if got := runCommand(context.Background(), flags, deps); got != 1 {
		t.Fatal(got)
	}
	if calls != 1 || len(fixture.requests) != 0 || len(fixture.turns) != 0 {
		t.Fatal("invalid SDK binding made external calls")
	}
}
