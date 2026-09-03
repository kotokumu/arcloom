package codexappserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// SupportedCodexVersion is the Codex app-server version implemented by this SDK.
const SupportedCodexVersion = "0.149.1"

var (
	errInvalidStdioClient    = errors.New("codex app-server: invalid Stdio Client configuration")
	errInvalidTurnInvocation = errors.New("codex app-server: invalid Turn invocation")
)

type stdioClient struct {
	executable    string
	shutdownGrace time.Duration
}

// NewStdioClient accepts an absolute executable path and a positive shutdown
// bound. It validates them without starting a process. Each
// CompleteReadOnlyTurn call starts its own process and verifies that the
// app-server identifies itself as SupportedCodexVersion before starting a
// Thread.
func NewStdioClient(codexExecutable string, shutdownGrace time.Duration) (Client, error) {
	if !filepath.IsAbs(codexExecutable) || shutdownGrace <= 0 {
		return nil, errInvalidStdioClient
	}
	if _, err := exec.LookPath(codexExecutable); err != nil {
		return nil, errInvalidStdioClient
	}
	return stdioClient{executable: codexExecutable, shutdownGrace: shutdownGrace}, nil
}

func (client stdioClient) CompleteReadOnlyTurn(ctx context.Context, request ReadOnlyTurnRequest) (CompletedTurn, error) {
	if ctx == nil {
		return CompletedTurn{}, errInvalidTurnInvocation
	}
	if err := ctx.Err(); err != nil {
		return CompletedTurn{}, err
	}
	if !request.valid() {
		return CompletedTurn{}, errInvalidTurnInvocation
	}
	workingDirectory, err := os.Stat(request.WorkingDirectory())
	if err != nil || !workingDirectory.IsDir() {
		return CompletedTurn{}, errInvalidTurnInvocation
	}

	process, err := startAppServerProcess(client.executable, request.WorkingDirectory())
	if err != nil {
		return CompletedTurn{}, errAppServerInteraction
	}
	interactionDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = process.closeStdin()
		case <-interactionDone:
		}
	}()
	completed, interactionErr := process.complete(ctx, request)
	close(interactionDone)
	shutdownErr := process.shutdown(client.shutdownGrace)
	if interactionErr != nil {
		return CompletedTurn{}, errors.Join(interactionErr, shutdownErr)
	}
	if shutdownErr != nil {
		return CompletedTurn{}, shutdownErr
	}
	return completed, nil
}

type appServerProcess struct {
	command         *exec.Cmd
	stdin           io.WriteCloser
	encoder         *json.Encoder
	messages        chan serverMessage
	readErrors      chan error
	stdoutDone      chan struct{}
	stderrDone      chan struct{}
	exited          chan struct{}
	stopReading     chan struct{}
	stopOnce        sync.Once
	stdinCloseErr   error
	stdinCloseMu    sync.Mutex
	waitError       error
	waitErrorMu     sync.Mutex
	threadRequested bool
	turnRequested   bool
	threadID        string
	turnID          string
	pendingThread   []serverMessage
	pendingTurn     []serverMessage
}

func startAppServerProcess(executable, workingDirectory string) (*appServerProcess, error) {
	command := exec.Command(executable, "app-server", "--listen", "stdio://")
	command.Dir = workingDirectory
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	process := &appServerProcess{
		command:     command,
		stdin:       stdin,
		encoder:     json.NewEncoder(stdin),
		messages:    make(chan serverMessage),
		readErrors:  make(chan error, 1),
		stdoutDone:  make(chan struct{}),
		stderrDone:  make(chan struct{}),
		exited:      make(chan struct{}),
		stopReading: make(chan struct{}),
	}
	go process.readStdout(json.NewDecoder(stdout))
	go func() {
		defer close(process.stderrDone)
		_, _ = io.Copy(io.Discard, stderr)
	}()
	go func() {
		<-process.stdoutDone
		<-process.stderrDone
		err := command.Wait()
		process.waitErrorMu.Lock()
		process.waitError = err
		process.waitErrorMu.Unlock()
		close(process.exited)
	}()
	return process, nil
}

func (process *appServerProcess) readStdout(decoder *json.Decoder) {
	defer close(process.stdoutDone)
	for {
		var encoded json.RawMessage
		if err := decoder.Decode(&encoded); err != nil {
			select {
			case process.readErrors <- err:
			default:
			}
			return
		}
		message, err := decodeServerMessage(encoded)
		if err != nil {
			select {
			case process.readErrors <- err:
			default:
			}
			return
		}
		select {
		case process.messages <- message:
		case <-process.stopReading:
			return
		}
	}
}

func (process *appServerProcess) complete(ctx context.Context, request ReadOnlyTurnRequest) (CompletedTurn, error) {
	initialize, err := process.call(ctx, 1, "initialize", map[string]any{
		"capabilities": map[string]any{"experimentalApi": true},
		"clientInfo":   map[string]string{"name": "arcloom", "version": "1"},
	})
	if err != nil {
		return CompletedTurn{}, err
	}
	if !compatibleUserAgent(initialize) {
		return CompletedTurn{}, errAppServerInteraction
	}
	if err := process.encoder.Encode(clientNotification{Method: "initialized"}); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return CompletedTurn{}, contextErr
		}
		return CompletedTurn{}, errAppServerInteraction
	}
	configuration, err := process.call(ctx, 2, "config/read", map[string]any{
		"cwd":           request.WorkingDirectory(),
		"includeLayers": false,
	})
	if err != nil {
		return CompletedTurn{}, err
	}
	threadConfiguration, err := readOnlyThreadConfiguration(configuration)
	if err != nil {
		return CompletedTurn{}, err
	}
	process.threadRequested = true
	thread, err := process.call(ctx, 3, "thread/start", map[string]any{
		"approvalPolicy":        "never",
		"config":                threadConfiguration,
		"cwd":                   request.WorkingDirectory(),
		"developerInstructions": request.DeveloperInstructions(),
		"ephemeral":             true,
		"model":                 request.Model(),
		"sandbox":               "read-only",
	})
	if err != nil {
		return CompletedTurn{}, err
	}
	threadID, err := decodeNestedID(thread, "thread")
	if err != nil {
		return CompletedTurn{}, err
	}
	process.threadID = threadID
	for _, message := range process.pendingThread {
		if err := process.validateNotificationCorrelation(message); err != nil {
			return CompletedTurn{}, err
		}
	}
	process.pendingThread = nil
	process.turnRequested = true
	turn, err := process.call(ctx, 4, "turn/start", map[string]any{
		"approvalPolicy": "never",
		"effort":         request.ReasoningEffort(),
		"input": []any{map[string]any{
			"type": "text",
			"text": request.Input(),
		}},
		"outputSchema": json.RawMessage(request.OutputSchema()),
		"sandboxPolicy": map[string]any{
			"type":          "readOnly",
			"networkAccess": false,
		},
		"threadId": threadID,
	})
	if err != nil {
		return CompletedTurn{}, err
	}
	turnID, err := decodeNestedID(turn, "turn")
	if err != nil {
		return CompletedTurn{}, err
	}
	process.turnID = turnID
	finalOutputs := make([]string, 0, 1)
	pending := process.pendingTurn
	process.pendingTurn = nil
	for {
		var message serverMessage
		if len(pending) != 0 {
			message = pending[0]
			pending = pending[1:]
		} else {
			message, err = process.receive(ctx)
			if err != nil {
				return CompletedTurn{}, err
			}
		}
		switch message.method {
		case "item/completed":
			finalOutput, isFinal, err := decodeCompletedItem(message.params, threadID, turnID)
			if err != nil {
				return CompletedTurn{}, err
			}
			if isFinal {
				finalOutputs = append(finalOutputs, finalOutput)
			}
		case "turn/completed":
			if err := ctx.Err(); err != nil {
				return CompletedTurn{}, err
			}
			if err := validateCompletedTurn(message.params, threadID, turnID); err != nil || len(finalOutputs) != 1 {
				return CompletedTurn{}, errAppServerInteraction
			}
			return CompletedTurn{FinalOutput: finalOutputs[0]}, nil
		default:
			if err := process.validateNotificationCorrelation(message); err != nil {
				return CompletedTurn{}, err
			}
			if err := process.acceptNotification(message); err != nil {
				return CompletedTurn{}, err
			}
		}
	}
}

func noHooksConfiguration() map[string]any {
	return map[string]any{
		"PermissionRequest": []any{},
		"PostCompact":       []any{},
		"PostToolUse":       []any{},
		"PreCompact":        []any{},
		"PreToolUse":        []any{},
		"SessionEnd":        []any{},
		"SessionStart":      []any{},
		"Stop":              []any{},
		"SubagentStart":     []any{},
		"SubagentStop":      []any{},
		"UserPromptSubmit":  []any{},
	}
}

func (process *appServerProcess) call(ctx context.Context, id int, method string, params any) (json.RawMessage, error) {
	if err := process.encoder.Encode(clientRequest{ID: id, Method: method, Params: params}); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return nil, contextErr
		}
		return nil, errAppServerInteraction
	}
	for {
		message, err := process.receive(ctx)
		if err != nil {
			return nil, err
		}
		if message.method != "" {
			if notificationNeedsTurnIdentity(message.method) {
				if !process.turnRequested || message.hasID {
					return nil, errAppServerInteraction
				}
				if process.turnID == "" {
					process.pendingTurn = append(process.pendingTurn, message)
					continue
				}
			} else if notificationNeedsThreadIdentity(message.method) {
				if !process.threadRequested || message.hasID {
					return nil, errAppServerInteraction
				}
				if process.threadID == "" {
					process.pendingThread = append(process.pendingThread, message)
					continue
				}
			}
			if err := process.validateNotificationCorrelation(message); err != nil {
				return nil, err
			}
			if err := process.acceptNotification(message); err != nil {
				return nil, err
			}
			continue
		}
		if err := decodeResponseID(message, id); err != nil {
			return nil, err
		}
		return message.result, nil
	}
}

func notificationNeedsThreadIdentity(method string) bool {
	switch method {
	case "thread/settings/updated", "thread/started", "thread/status/changed":
		return true
	default:
		return notificationNeedsTurnIdentity(method)
	}
}

func notificationNeedsTurnIdentity(method string) bool {
	switch method {
	case "item/agentMessage/delta",
		"item/commandExecution/outputDelta",
		"item/commandExecution/terminalInteraction",
		"item/completed",
		"item/plan/delta",
		"item/reasoning/summaryPartAdded",
		"item/reasoning/summaryTextDelta",
		"item/reasoning/textDelta",
		"item/started",
		"thread/tokenUsage/updated",
		"turn/completed",
		"turn/moderationMetadata",
		"turn/plan/updated",
		"turn/started":
		return true
	default:
		return false
	}
}

func (process *appServerProcess) validateNotificationCorrelation(message serverMessage) error {
	if notificationNeedsTurnIdentity(message.method) {
		return validateNotificationIdentity(message.params, process.threadID, process.turnID, true)
	}
	if notificationNeedsThreadIdentity(message.method) {
		return validateNotificationIdentity(message.params, process.threadID, "", false)
	}
	return nil
}

func (process *appServerProcess) acceptNotification(message serverMessage) error {
	if message.hasID {
		return errAppServerInteraction
	}
	switch message.method {
	case "configWarning",
		"deprecationNotice",
		"item/agentMessage/delta",
		"item/commandExecution/outputDelta",
		"item/commandExecution/terminalInteraction",
		"item/plan/delta",
		"item/reasoning/summaryPartAdded",
		"item/reasoning/summaryTextDelta",
		"item/reasoning/textDelta",
		"item/started",
		"process/exited",
		"process/outputDelta",
		"remoteControl/status/changed",
		"account/rateLimits/updated",
		"thread/settings/updated",
		"thread/started",
		"thread/status/changed",
		"thread/tokenUsage/updated",
		"turn/moderationMetadata",
		"turn/plan/updated",
		"turn/started",
		"warning":
		return nil
	default:
		return errAppServerInteraction
	}
}

func (process *appServerProcess) receive(ctx context.Context) (serverMessage, error) {
	if err := ctx.Err(); err != nil {
		return serverMessage{}, err
	}
	select {
	case <-ctx.Done():
		return serverMessage{}, ctx.Err()
	case message := <-process.messages:
		if err := ctx.Err(); err != nil {
			return serverMessage{}, err
		}
		return message, nil
	case <-process.readErrors:
		if err := ctx.Err(); err != nil {
			return serverMessage{}, err
		}
		return serverMessage{}, errAppServerInteraction
	case <-process.exited:
		if err := ctx.Err(); err != nil {
			return serverMessage{}, err
		}
		return serverMessage{}, errAppServerInteraction
	}
}

func (process *appServerProcess) shutdown(grace time.Duration) error {
	closeErr := process.closeStdin()
	closeFailed := closeErr != nil && !errors.Is(closeErr, os.ErrClosed)
	close(process.stopReading)
	cooperativeTimer := time.NewTimer(grace / 2)
	defer cooperativeTimer.Stop()
	killed := false
	select {
	case <-process.exited:
	case <-cooperativeTimer.C:
		killed = true
		if err := process.command.Process.Kill(); err != nil {
			select {
			case <-process.exited:
			default:
				return errAppServerInteraction
			}
		}
		reapTimer := time.NewTimer(grace - grace/2)
		defer reapTimer.Stop()
		select {
		case <-process.exited:
		case <-reapTimer.C:
			return errAppServerInteraction
		}
	}
	process.waitErrorMu.Lock()
	waitErr := process.waitError
	process.waitErrorMu.Unlock()
	if closeFailed || waitErr != nil && !killed {
		return errAppServerInteraction
	}
	return nil
}

func (process *appServerProcess) closeStdin() error {
	process.stopOnce.Do(func() {
		err := process.stdin.Close()
		process.stdinCloseMu.Lock()
		process.stdinCloseErr = err
		process.stdinCloseMu.Unlock()
	})
	process.stdinCloseMu.Lock()
	defer process.stdinCloseMu.Unlock()
	return process.stdinCloseErr
}
