package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestNewStdioClient(t *testing.T) {
	validExecutable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	nonExecutable := filepath.Join(t.TempDir(), "not-executable")
	if err := os.WriteFile(nonExecutable, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}

	type args struct {
		in0 string
		in1 time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "absolute executable and positive shutdown grace",
			args: args{in0: validExecutable, in1: time.Second},
		},
		{name: "relative executable", args: args{in0: "codex", in1: time.Second}, wantErr: true},
		{name: "missing executable", args: args{in0: filepath.Join(t.TempDir(), "missing"), in1: time.Second}, wantErr: true},
		{name: "directory instead of executable", args: args{in0: t.TempDir(), in1: time.Second}, wantErr: true},
		{name: "file without executable permission", args: args{in0: nonExecutable, in1: time.Second}, wantErr: true},
		{name: "zero shutdown grace", args: args{in0: validExecutable}, wantErr: true},
		{name: "negative shutdown grace", args: args{in0: validExecutable, in1: -time.Nanosecond}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewStdioClient(tt.args.in0, tt.args.in1)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewStdioClient() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Error("NewStdioClient() returned a nil Client")
			}
		})
	}
}

func TestNewStdioClient_doesNotStartProcess(t *testing.T) {
	capturePath := filepath.Join(t.TempDir(), "capture.json")
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
	client, err := NewStdioClient(buildAppServerStub(t), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("NewStdioClient() returned a nil Client")
	}
	if _, err := os.Stat(capturePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("NewStdioClient() started a process: %v", err)
	}
}

func Test_stdioClient_CompleteReadOnlyTurn(t *testing.T) {
	stubPath := buildAppServerStub(t)
	workingDirectory := t.TempDir()
	request := newTurnRequest(t, workingDirectory, `{"currentPlan":{"name":"Release"}}`)

	tests := []struct {
		name              string
		mode              string
		want              CompletedTurn
		wantErr           bool
		wantNoThreadStart bool
	}{
		{
			name: "compatible process completes one isolated read-only turn",
			want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`},
		},
		{
			name: "completion arriving before Turn response is retained",
			mode: "early_completion",
			want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`},
		},
		{
			name: "message larger than 64 KiB is decoded",
			mode: "large_message",
			want: CompletedTurn{FinalOutput: strings.Repeat("x", 96*1024)},
		},
		{
			name: "stderr larger than the pipe buffer is drained",
			mode: "stderr_flood",
			want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`},
		},
		{name: "terminal exchange survives immediate process exit", mode: "fast_exit", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "rapid notifications preserve the ordered exchange", mode: "notification_burst", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "compatible Thread settings notification", mode: "thread_settings_notification", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "compatible account rate limits notification", mode: "account_rate_limits_notification", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "Thread settings notification for another Thread", mode: "mismatched_thread_settings_notification", wantErr: true},
		{name: "notification for another Turn", mode: "mismatched_turn_notification", wantErr: true},
		{name: "disabled MCP configuration is safe", mode: "disabled_mcp", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "disabled Apps configuration is safe", mode: "disabled_apps", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "empty Hooks configuration is safe", mode: "empty_hooks", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "incompatible version", mode: "incompatible_version", wantErr: true, wantNoThreadStart: true},
		{name: "another product with the supported number", mode: "other_product", wantErr: true, wantNoThreadStart: true},
		{name: "missing user agent", mode: "missing_user_agent", wantErr: true, wantNoThreadStart: true},
		{name: "effective MCP configuration", mode: "unsafe_mcp", wantErr: true, wantNoThreadStart: true},
		{name: "effective Apps configuration", mode: "unsafe_apps", wantErr: true, wantNoThreadStart: true},
		{name: "effective Hooks configuration", mode: "unsafe_hooks", wantErr: true, wantNoThreadStart: true},
		{name: "effective Web Search mode", mode: "unsafe_web_search", wantErr: true, wantNoThreadStart: true},
		{name: "effective Web Search tool", mode: "unsafe_web_tool", wantErr: true, wantNoThreadStart: true},
		{name: "invalid MCP configuration shape", mode: "invalid_mcp_shape", wantErr: true, wantNoThreadStart: true},
		{name: "server request", mode: "server_request", wantErr: true, wantNoThreadStart: true},
		{name: "error response", mode: "error_response", wantErr: true, wantNoThreadStart: true},
		{name: "unknown notification", mode: "unknown_notification", wantErr: true},
		{name: "malformed message", mode: "malformed", wantErr: true, wantNoThreadStart: true},
		{name: "duplicate message field", mode: "duplicate", wantErr: true, wantNoThreadStart: true},
		{name: "mismatched response ID", mode: "mismatched_response", wantErr: true, wantNoThreadStart: true},
		{name: "mismatched config response ID", mode: "mismatched_config_response", wantErr: true, wantNoThreadStart: true},
		{name: "mismatched Thread response ID", mode: "mismatched_thread_response", wantErr: true},
		{name: "mismatched Turn response ID", mode: "mismatched_turn_response", wantErr: true},
		{name: "failed Turn", mode: "failed_turn", wantErr: true},
		{name: "interrupted Turn", mode: "interrupted_turn", wantErr: true},
		{name: "completed Turn without final output", mode: "missing_final", wantErr: true},
		{name: "multiple final outputs", mode: "multiple_final", wantErr: true},
		{name: "unknown agent message phase", mode: "unknown_phase", wantErr: true},
		{name: "file change item", mode: "file_change_item", wantErr: true},
		{name: "MCP item", mode: "mcp_item", wantErr: true},
		{name: "Web Search item", mode: "web_search_item", wantErr: true},
		{name: "legacy final output without phase", mode: "missing_phase", want: CompletedTurn{FinalOutput: `{"outcome":"complete"}`}},
		{name: "completed item for another Thread", mode: "mismatched_item", wantErr: true},
		{name: "completion for another Thread", mode: "mismatched_turn", wantErr: true},
		{name: "completion for another Turn", mode: "mismatched_completion_turn", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturePath := filepath.Join(t.TempDir(), "capture.json")
			t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
			t.Setenv("ARCLOOM_CODEX_STUB_MODE", tt.mode)
			client := newStdioClient(t, stubPath, 200*time.Millisecond)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			got, err := client.CompleteReadOnlyTurn(ctx, request)
			if (err != nil) != tt.wantErr {
				t.Fatalf("stdioClient.CompleteReadOnlyTurn() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("stdioClient.CompleteReadOnlyTurn() mismatch (-want +got):\n%s", diff)
			}
			captured := readStubCapture(t, capturePath)
			if tt.wantNoThreadStart && len(captured.ThreadStart) != 0 && string(captured.ThreadStart) != "null" {
				t.Errorf("Thread started before compatibility and safety were established: %s", captured.ThreadStart)
			}
			if err := syscall.Kill(captured.PID, 0); !errors.Is(err, syscall.ESRCH) {
				t.Errorf("app-server process %d was not reaped: %v", captured.PID, err)
			}
			if !tt.wantErr {
				assertReadOnlyWire(t, captured, workingDirectory)
			}
		})
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_rejectsInvalidInvocationWithoutStartingProcess(t *testing.T) {
	stubPath := buildAppServerStub(t)
	capturePath := filepath.Join(t.TempDir(), "capture.json")
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
	workingDirectory := t.TempDir()
	validRequest := newTurnRequest(t, workingDirectory, "input")
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	missingDirectoryRequest := newTurnRequest(t, filepath.Join(t.TempDir(), "missing"), "input")

	tests := []struct {
		name    string
		ctx     context.Context
		request ReadOnlyTurnRequest
		wantErr error
	}{
		{name: "nil Context", request: validRequest, wantErr: errInvalidTurnInvocation},
		{name: "already cancelled Context", ctx: cancelledContext, request: validRequest, wantErr: context.Canceled},
		{name: "zero Request", ctx: context.Background(), wantErr: errInvalidTurnInvocation},
		{name: "missing working directory", ctx: context.Background(), request: missingDirectoryRequest, wantErr: errInvalidTurnInvocation},
	}
	client := newStdioClient(t, stubPath, time.Second)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := client.CompleteReadOnlyTurn(tt.ctx, tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("stdioClient.CompleteReadOnlyTurn() error = %v, want %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(CompletedTurn{}, got); diff != "" {
				t.Errorf("stdioClient.CompleteReadOnlyTurn() mismatch (-want +got):\n%s", diff)
			}
			if _, err := os.Stat(capturePath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("invalid invocation started a process: %v", err)
			}
		})
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_cancellationClosesOrKillsAndReaps(t *testing.T) {
	stubPath := buildAppServerStub(t)
	workingDirectory := t.TempDir()
	request := newTurnRequest(t, workingDirectory, "input")
	client := newStdioClient(t, stubPath, 120*time.Millisecond)

	for _, tt := range []struct {
		name            string
		mode            string
		wantShutdownErr bool
	}{
		{name: "cooperative process exits after stdin closes", mode: "cooperative_cancel"},
		{name: "context identity survives a shutdown error", mode: "cancel_exit_error", wantShutdownErr: true},
		{name: "unresponsive process is forcibly terminated", mode: "forced_cancel"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			capturePath := filepath.Join(t.TempDir(), "capture.json")
			t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
			t.Setenv("ARCLOOM_CODEX_STUB_MODE", tt.mode)
			const grace = 120 * time.Millisecond
			ctx, cancel := context.WithCancel(context.Background())
			result := make(chan struct {
				turn CompletedTurn
				err  error
			}, 1)
			go func() {
				turn, err := client.CompleteReadOnlyTurn(ctx, request)
				result <- struct {
					turn CompletedTurn
					err  error
				}{turn: turn, err: err}
			}()
			waitForTurnStart(t, capturePath)
			started := time.Now()
			cancel()
			var got struct {
				turn CompletedTurn
				err  error
			}
			select {
			case got = <-result:
			case <-time.After(time.Second):
				t.Fatal("cancellation did not complete within the test bound")
			}
			elapsed := time.Since(started)
			if !errors.Is(got.err, context.Canceled) {
				t.Fatalf("stdioClient.CompleteReadOnlyTurn() error = %v (context=%v, interaction=%v, capture=%+v), want context.Canceled", got.err, ctx.Err(), errors.Is(got.err, errAppServerInteraction), readStubCapture(t, capturePath))
			}
			if errors.Is(got.err, errAppServerInteraction) != tt.wantShutdownErr {
				t.Errorf("stdioClient.CompleteReadOnlyTurn() shutdown error = %v, want %v", got.err, tt.wantShutdownErr)
			}
			if diff := cmp.Diff(CompletedTurn{}, got.turn); diff != "" {
				t.Errorf("stdioClient.CompleteReadOnlyTurn() mismatch (-want +got):\n%s", diff)
			}
			if elapsed > grace+750*time.Millisecond {
				t.Errorf("cancellation took %s, expected configured grace %s plus scheduling tolerance", elapsed, grace)
			}
			captured := readStubCapture(t, capturePath)
			if err := syscall.Kill(captured.PID, 0); !errors.Is(err, syscall.ESRCH) {
				t.Errorf("app-server process %d was not reaped: %v", captured.PID, err)
			}
			if (tt.mode == "cooperative_cancel" || tt.mode == "cancel_exit_error") && !captured.StdinClosed {
				t.Error("cooperative process did not observe stdin close")
			}
		})
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_preservesDeadlineExceeded(t *testing.T) {
	stubPath := buildAppServerStub(t)
	capturePath := filepath.Join(t.TempDir(), "capture.json")
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
	t.Setenv("ARCLOOM_CODEX_STUB_MODE", "forced_cancel")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	started := time.Now()
	client := newStdioClient(t, stubPath, 120*time.Millisecond)
	got, err := client.CompleteReadOnlyTurn(
		ctx,
		newTurnRequest(t, t.TempDir(), "input"),
	)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("stdioClient.CompleteReadOnlyTurn() error = %v, want context.DeadlineExceeded", err)
	}
	if diff := cmp.Diff(CompletedTurn{}, got); diff != "" {
		t.Errorf("stdioClient.CompleteReadOnlyTurn() mismatch (-want +got):\n%s", diff)
	}
	if elapsed := time.Since(started); elapsed > 900*time.Millisecond {
		t.Errorf("deadline cancellation took %s", elapsed)
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_cancellationUnblocksStdinWrite(t *testing.T) {
	stubPath := buildAppServerStub(t)
	capturePath := filepath.Join(t.TempDir(), "capture.json")
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
	t.Setenv("ARCLOOM_CODEX_STUB_MODE", "stop_before_turn_read")
	ctx, cancel := context.WithCancel(context.Background())
	request := newTurnRequest(t, t.TempDir(), strings.Repeat("large input", 1024*1024))
	client := newStdioClient(t, stubPath, 120*time.Millisecond)
	type result struct {
		turn CompletedTurn
		err  error
	}
	resultChannel := make(chan result, 1)
	go func() {
		turn, err := client.CompleteReadOnlyTurn(ctx, request)
		resultChannel <- result{turn: turn, err: err}
	}()
	waitForCapture(t, capturePath, func(captured stubCapture) bool { return captured.ThreadResponded })
	time.Sleep(25 * time.Millisecond)
	started := time.Now()
	cancel()
	select {
	case got := <-resultChannel:
		if !errors.Is(got.err, context.Canceled) {
			t.Errorf("stdioClient.CompleteReadOnlyTurn() error = %v, want context.Canceled", got.err)
		}
		if diff := cmp.Diff(CompletedTurn{}, got.turn); diff != "" {
			t.Errorf("stdioClient.CompleteReadOnlyTurn() mismatch (-want +got):\n%s", diff)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not unblock the stdin write")
	}
	if elapsed := time.Since(started); elapsed > 850*time.Millisecond {
		t.Errorf("blocked-write cancellation took %s", elapsed)
	}
	captured := readStubCapture(t, capturePath)
	if err := syscall.Kill(captured.PID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("app-server process %d was not reaped: %v", captured.PID, err)
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_establishedSuccessSurvivesLateCancellation(t *testing.T) {
	stubPath := buildAppServerStub(t)
	capturePath := filepath.Join(t.TempDir(), "capture.json")
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
	t.Setenv("ARCLOOM_CODEX_STUB_MODE", "completed_hang")
	ctx, cancel := context.WithCancel(context.Background())
	request := newTurnRequest(t, t.TempDir(), "input")
	client := newStdioClient(t, stubPath, 100*time.Millisecond)
	result := make(chan struct {
		turn CompletedTurn
		err  error
	}, 1)
	go func() {
		turn, err := client.CompleteReadOnlyTurn(
			ctx,
			request,
		)
		result <- struct {
			turn CompletedTurn
			err  error
		}{turn: turn, err: err}
	}()
	waitForStdinClose(t, capturePath)
	started := time.Now()
	cancel()
	var got struct {
		turn CompletedTurn
		err  error
	}
	select {
	case got = <-result:
	case <-time.After(time.Second):
		t.Fatal("established success did not complete bounded shutdown")
	}
	if got.err != nil {
		t.Fatalf("stdioClient.CompleteReadOnlyTurn() error = %v", got.err)
	}
	if diff := cmp.Diff(CompletedTurn{FinalOutput: `{"outcome":"complete"}`}, got.turn); diff != "" {
		t.Errorf("stdioClient.CompleteReadOnlyTurn() mismatch (-want +got):\n%s", diff)
	}
	if elapsed := time.Since(started); elapsed > 850*time.Millisecond {
		t.Errorf("established success shutdown took %s", elapsed)
	}
	captured := readStubCapture(t, capturePath)
	if err := syscall.Kill(captured.PID, 0); !errors.Is(err, syscall.ESRCH) {
		t.Errorf("app-server process %d was not reaped: %v", captured.PID, err)
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_concurrentCallsAreIsolated(t *testing.T) {
	stubPath := buildAppServerStub(t)
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", "")
	t.Setenv("ARCLOOM_CODEX_STUB_MODE", "echo")
	client := newStdioClient(t, stubPath, time.Second)
	wantOutputs := []string{"first material", "second material", "third material", "fourth material"}
	requests := make([]ReadOnlyTurnRequest, len(wantOutputs))
	for index, input := range wantOutputs {
		requests[index] = newTurnRequest(t, t.TempDir(), input)
	}
	gotOutputs := make([]string, len(wantOutputs))
	errs := make([]error, len(wantOutputs))
	var group sync.WaitGroup
	for index, request := range requests {
		group.Add(1)
		go func() {
			defer group.Done()
			gotOutputs[index], errs[index] = func() (string, error) {
				turn, err := client.CompleteReadOnlyTurn(context.Background(), request)
				return turn.FinalOutput, err
			}()
		}()
	}
	done := make(chan struct{})
	go func() {
		group.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("concurrent calls did not finish within the test bound")
	}
	for index, err := range errs {
		if err != nil {
			t.Fatalf("call %d failed: %v", index, err)
		}
	}
	if diff := cmp.Diff(wantOutputs, gotOutputs); diff != "" {
		t.Errorf("concurrent output mismatch (-want +got):\n%s", diff)
	}
}

func Test_stdioClient_CompleteReadOnlyTurn_oneCancellationDoesNotAffectConcurrentCall(t *testing.T) {
	stubPath := buildAppServerStub(t)
	capturePath := filepath.Join(t.TempDir(), "capture.json")
	t.Setenv("ARCLOOM_CODEX_STUB_CAPTURE", capturePath)
	t.Setenv("ARCLOOM_CODEX_STUB_MODE", "mixed")
	client := newStdioClient(t, stubPath, 150*time.Millisecond)
	cancelContext, cancel := context.WithCancel(context.Background())
	cancelRequest := newTurnRequest(t, t.TempDir(), "cancel material")
	successRequest := newTurnRequest(t, t.TempDir(), "successful material")
	type result struct {
		turn CompletedTurn
		err  error
	}
	cancelled := make(chan result, 1)
	succeeded := make(chan result, 1)
	go func() {
		turn, err := client.CompleteReadOnlyTurn(cancelContext, cancelRequest)
		cancelled <- result{turn: turn, err: err}
	}()
	go func() {
		turn, err := client.CompleteReadOnlyTurn(context.Background(), successRequest)
		succeeded <- result{turn: turn, err: err}
	}()
	waitForFile(t, capturePath+".cancel-ready")
	cancel()

	var cancelledResult, succeededResult result
	select {
	case cancelledResult = <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled call did not finish within the test bound")
	}
	select {
	case succeededResult = <-succeeded:
	case <-time.After(2 * time.Second):
		t.Fatal("unaffected call did not finish within the test bound")
	}
	if !errors.Is(cancelledResult.err, context.Canceled) {
		t.Errorf("cancelled call error = %v, want context.Canceled", cancelledResult.err)
	}
	if diff := cmp.Diff(CompletedTurn{}, cancelledResult.turn); diff != "" {
		t.Errorf("cancelled call mismatch (-want +got):\n%s", diff)
	}
	if succeededResult.err != nil {
		t.Fatalf("unaffected call error = %v", succeededResult.err)
	}
	if diff := cmp.Diff(CompletedTurn{FinalOutput: "successful material"}, succeededResult.turn); diff != "" {
		t.Errorf("unaffected call mismatch (-want +got):\n%s", diff)
	}
	captureFiles, err := filepath.Glob(capturePath + ".[0-9]*")
	if err != nil {
		t.Fatal(err)
	}
	if len(captureFiles) != 2 {
		t.Fatalf("concurrent calls used %d observed processes, want 2: %v", len(captureFiles), captureFiles)
	}
	pids := make(map[int]struct{}, 2)
	for _, path := range captureFiles {
		captured := readStubCapture(t, path)
		pids[captured.PID] = struct{}{}
		if err := syscall.Kill(captured.PID, 0); !errors.Is(err, syscall.ESRCH) {
			t.Errorf("app-server process %d was not reaped: %v", captured.PID, err)
		}
	}
	if len(pids) != 2 {
		t.Errorf("concurrent calls did not own distinct processes: %v", pids)
	}
}

type stubCapture struct {
	PID             int             `json:"pid"`
	Arguments       []string        `json:"arguments"`
	Initialize      json.RawMessage `json:"initialize"`
	ConfigRead      json.RawMessage `json:"configRead"`
	ThreadStart     json.RawMessage `json:"threadStart"`
	ThreadResponded bool            `json:"threadResponded"`
	TurnStart       json.RawMessage `json:"turnStart"`
	StdinClosed     bool            `json:"stdinClosed"`
	JSONRPC         bool            `json:"jsonrpc"`
	Messages        []stubMessage   `json:"messages"`
}

type stubMessage struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func buildAppServerStub(t *testing.T) string {
	t.Helper()
	stubPath := filepath.Join(t.TempDir(), "codex")
	build := exec.Command("go", "build", "-o", stubPath, "./testdata/codexappserverstub")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build Codex app-server stub: %v\n%s", err, output)
	}
	return stubPath
}

func newTurnRequest(t *testing.T, workingDirectory, input string) ReadOnlyTurnRequest {
	t.Helper()
	request, err := NewReadOnlyTurnRequest(
		"gpt-5.6-sol",
		"high",
		workingDirectory,
		"Assess only supplied material.",
		input,
		`{"type":"object"}`,
	)
	if err != nil {
		t.Fatal(err)
	}
	return request
}

func newStdioClient(t *testing.T, executable string, shutdownGrace time.Duration) Client {
	t.Helper()
	client, err := NewStdioClient(executable, shutdownGrace)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func readStubCapture(t *testing.T, capturePath string) stubCapture {
	t.Helper()
	encodedCapture, err := os.ReadFile(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	var captured stubCapture
	if err := json.Unmarshal(encodedCapture, &captured); err != nil {
		t.Fatal(err)
	}
	return captured
}

func waitForTurnStart(t *testing.T, capturePath string) {
	t.Helper()
	waitForCapture(t, capturePath, func(captured stubCapture) bool {
		return len(captured.TurnStart) != 0 && string(captured.TurnStart) != "null"
	})
}

func waitForStdinClose(t *testing.T, capturePath string) {
	t.Helper()
	waitForCapture(t, capturePath, func(captured stubCapture) bool { return captured.StdinClosed })
}

func waitForCapture(t *testing.T, capturePath string, ready func(stubCapture) bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		encoded, err := os.ReadFile(capturePath)
		if err == nil {
			var captured stubCapture
			if json.Unmarshal(encoded, &captured) == nil && ready(captured) {
				return
			}
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for app-server stub state")
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func assertReadOnlyWire(t *testing.T, captured stubCapture, workingDirectory string) {
	t.Helper()
	if diff := cmp.Diff([]string{"app-server", "--listen", "stdio://"}, captured.Arguments); diff != "" {
		t.Errorf("process arguments mismatch (-want +got):\n%s", diff)
	}
	if captured.JSONRPC {
		t.Error("wire request included an unsupported jsonrpc member")
	}
	if len(captured.Messages) != 5 {
		t.Fatalf("outbound lifecycle has %d messages: %#v", len(captured.Messages), captured.Messages)
	}
	wantMethods := []string{"initialize", "initialized", "config/read", "thread/start", "turn/start"}
	wantIDs := []string{"1", "null", "2", "3", "4"}
	for index, message := range captured.Messages {
		if message.Method != wantMethods[index] || string(message.ID) != wantIDs[index] {
			t.Errorf("outbound message %d = id %s method %q, want id %s method %q", index, message.ID, message.Method, wantIDs[index], wantMethods[index])
		}
	}
	var initialize, configRead, threadStart, turnStart map[string]any
	for _, target := range []struct {
		name    string
		encoded json.RawMessage
		value   *map[string]any
	}{
		{name: "initialize", encoded: captured.Initialize, value: &initialize},
		{name: "config/read", encoded: captured.ConfigRead, value: &configRead},
		{name: "thread/start", encoded: captured.ThreadStart, value: &threadStart},
		{name: "turn/start", encoded: captured.TurnStart, value: &turnStart},
	} {
		if err := json.Unmarshal(target.encoded, target.value); err != nil {
			t.Fatalf("decode %s: %v", target.name, err)
		}
	}
	if diff := cmp.Diff(map[string]any{
		"capabilities": map[string]any{"experimentalApi": true},
		"clientInfo":   map[string]any{"name": "arcloom", "version": "1"},
	}, initialize); diff != "" {
		t.Errorf("initialize params mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(map[string]any{"cwd": workingDirectory, "includeLayers": false}, configRead); diff != "" {
		t.Errorf("config/read params mismatch (-want +got):\n%s", diff)
	}
	if threadStart["approvalPolicy"] != "never" || threadStart["sandbox"] != "read-only" ||
		threadStart["cwd"] != workingDirectory || threadStart["ephemeral"] != true ||
		threadStart["model"] != "gpt-5.6-sol" || threadStart["developerInstructions"] != "Assess only supplied material." {
		t.Errorf("unsafe or incomplete thread/start params: %#v", threadStart)
	}
	wantThreadConfig := map[string]any{
		"apps": map[string]any{}, "hooks": noHooksConfiguration(), "mcp_servers": map[string]any{}, "web_search": "disabled",
	}
	if diff := cmp.Diff(wantThreadConfig, threadStart["config"]); diff != "" {
		t.Errorf("thread/start config mismatch (-want +got):\n%s", diff)
	}
	if turnStart["approvalPolicy"] != "never" || turnStart["effort"] != "high" || turnStart["threadId"] != "thread-stub" {
		t.Errorf("unsafe or incomplete turn/start params: %#v", turnStart)
	}
	wantInput := []any{map[string]any{"type": "text", "text": `{"currentPlan":{"name":"Release"}}`}}
	if diff := cmp.Diff(wantInput, turnStart["input"]); diff != "" {
		t.Errorf("turn/start input mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(map[string]any{"type": "object"}, turnStart["outputSchema"]); diff != "" {
		t.Errorf("turn/start output schema mismatch (-want +got):\n%s", diff)
	}
	wantSandbox := map[string]any{"type": "readOnly", "networkAccess": false}
	if diff := cmp.Diff(wantSandbox, turnStart["sandboxPolicy"]); diff != "" {
		t.Errorf("turn/start sandbox mismatch (-want +got):\n%s", diff)
	}
}
