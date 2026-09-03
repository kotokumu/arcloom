//go:build linux || darwin

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kotokumu/arcloom/providers/codex/appserver"
)

func main() { os.Exit(runInterrupted(os.Args[1:], nativeDependencies())) }

func runInterrupted(arguments []string, deps commandDependencies) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runApplication(ctx, arguments, deps)
}

func runApplication(ctx context.Context, arguments []string, deps commandDependencies) int {
	if ctx == nil {
		return 1
	}
	if ctx.Err() != nil {
		return 130
	}
	// Parse the same immutable command arguments to establish the output boundary
	// before composing any Provider work. The cycle also validates its own inputs.
	options, err := parseOptions(arguments)
	if err != nil {
		return 1
	}
	output, err := openFIFOOutput(ctx, options.outputPath)
	if err != nil {
		return commandExit(ctx, 1)
	}
	deps.writeRecord = output.WriteRecord
	code := runCommand(ctx, arguments, deps)
	if err := output.Close(); err != nil {
		return commandExit(ctx, 1)
	}
	return code
}

func nativeDependencies() commandDependencies {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	return commandDependencies{
		httpClient:     &http.Client{Transport: githubCredentialTransport{token: os.Getenv("GITHUB_TOKEN"), transport: transport}},
		newCodexClient: codexappserver.NewStdioClient,
		readEvidence:   readOperatorEvidence,
		now:            time.Now,
		build:          runtimeBuildEvidence(),
	}
}

type githubCredentialTransport struct {
	token     string
	transport *http.Transport
}

func (t githubCredentialTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	authorized, err := authorizeGitHubRequest(request, t.token)
	if err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(authorized)
}
