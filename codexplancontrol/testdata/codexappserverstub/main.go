package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type capture struct {
	Args             []string        `json:"args"`
	WorkingDirectory string          `json:"workingDirectory"`
	ThreadStart      json.RawMessage `json:"threadStart"`
	TurnStart        json.RawMessage `json:"turnStart"`
}

func main() {
	if launchPath := os.Getenv("CODEX_STUB_LAUNCH"); launchPath != "" {
		_ = os.WriteFile(launchPath, []byte("launched"), 0o600)
	}
	requests := bufio.NewScanner(os.Stdin)
	responses := json.NewEncoder(os.Stdout)
	mode := os.Getenv("CODEX_STUB_MODE")
	var threadStart json.RawMessage
	for requests.Scan() {
		var incoming request
		if json.Unmarshal(requests.Bytes(), &incoming) != nil {
			return
		}
		switch incoming.Method {
		case "initialize":
			if mode == "initialize_rpc_error" {
				_ = responses.Encode(map[string]any{"jsonrpc": "2.0", "id": incoming.ID, "error": map[string]any{"code": -32000, "message": "initialize failed"}})
				continue
			}
			initializeResponse := map[string]any{"jsonrpc": "2.0", "id": incoming.ID, "result": map[string]any{}}
			switch mode {
			case "missing_jsonrpc":
				delete(initializeResponse, "jsonrpc")
			case "wrong_jsonrpc":
				initializeResponse["jsonrpc"] = "1.0"
			case "missing_result":
				delete(initializeResponse, "result")
			case "result_and_error":
				initializeResponse["error"] = map[string]any{"code": -32000, "message": "ambiguous"}
			case "duplicate_jsonrpc":
				_, _ = os.Stdout.WriteString(`{"jsonrpc":"2.0","jsonrpc":"1.0","id":1,"result":{}}` + "\n")
				continue
			}
			_ = responses.Encode(initializeResponse)
		case "initialized":
		case "thread/start":
			if mode == "thread_rpc_error" {
				_ = responses.Encode(map[string]any{"jsonrpc": "2.0", "id": incoming.ID, "error": map[string]any{"code": -32001, "message": "thread failed"}})
				continue
			}
			threadStart = append(json.RawMessage(nil), incoming.Params...)
			threadResponse := map[string]any{
				"jsonrpc": "2.0",
				"id":      incoming.ID,
				"result":  map[string]any{"thread": map[string]any{"id": "thread-stub"}},
			}
			if mode == "duplicate_thread_id" {
				encoded, _ := json.Marshal(threadResponse)
				encoded = bytes.Replace(encoded, []byte(`"id":"thread-stub"`), []byte(`"id":"thread-stub","id":"thread-other"`), 1)
				_, _ = os.Stdout.Write(append(encoded, '\n'))
			} else {
				_ = responses.Encode(threadResponse)
			}
			if mode == "block_before_turn" {
				if readyDirectory := os.Getenv("CODEX_STUB_READY_DIRECTORY"); readyDirectory != "" {
					_ = os.WriteFile(filepath.Join(readyDirectory, "blocked.ready"), []byte("ready"), 0o600)
				}
				remainNonCooperativeUntilKilled()
			}
		case "turn/start":
			var turnInput struct {
				Input []struct {
					Text string `json:"text"`
				} `json:"input"`
			}
			_ = json.Unmarshal(incoming.Params, &turnInput)
			var material struct {
				Observations string `json:"observations"`
			}
			if len(turnInput.Input) == 1 {
				_ = json.Unmarshal([]byte(turnInput.Input[0].Text), &material)
			}
			if readyDirectory := os.Getenv("CODEX_STUB_READY_DIRECTORY"); readyDirectory != "" && material.Observations != "" {
				_ = os.WriteFile(filepath.Join(readyDirectory, material.Observations+".ready"), []byte(turnInput.Input[0].Text), 0o600)
			}
			capturePath := os.Getenv("CODEX_STUB_CAPTURE")
			workingDirectory, _ := os.Getwd()
			record, _ := json.Marshal(capture{
				Args:             append([]string(nil), os.Args[1:]...),
				WorkingDirectory: workingDirectory,
				ThreadStart:      threadStart,
				TurnStart:        append(json.RawMessage(nil), incoming.Params...),
			})
			_ = os.WriteFile(capturePath, record, 0o600)
			if mode == "turn_rpc_error" {
				_ = responses.Encode(map[string]any{"jsonrpc": "2.0", "id": incoming.ID, "error": map[string]any{"code": -32002, "message": "turn failed"}})
				continue
			}
			turnResponse := map[string]any{
				"jsonrpc": "2.0",
				"id":      incoming.ID,
				"result":  map[string]any{"turn": map[string]any{"id": "turn-stub"}},
			}
			if mode == "duplicate_turn_id" {
				encoded, _ := json.Marshal(turnResponse)
				encoded = bytes.Replace(encoded, []byte(`"id":"turn-stub"`), []byte(`"id":"turn-stub","id":"turn-other"`), 1)
				_, _ = os.Stdout.Write(append(encoded, '\n'))
			} else {
				_ = responses.Encode(turnResponse)
			}
			if mode == "malformed_protocol" {
				_, _ = os.Stdout.WriteString("{malformed\n")
				continue
			}
			if mode == "server_request" {
				_ = responses.Encode(map[string]any{"jsonrpc": "2.0", "id": 77, "method": "item/commandExecution/requestApproval", "params": map[string]any{}})
				continue
			}
			if mode == "tool_notification" {
				_ = responses.Encode(map[string]any{"jsonrpc": "2.0", "method": "item/commandExecution/outputDelta", "params": map[string]any{"delta": "outside material"}})
			}
			if mode == "hang" || (mode == "mixed" && material.Observations == "hang") {
				remainNonCooperativeUntilKilled()
			}
			output := os.Getenv("CODEX_STUB_OUTPUT")
			if mode == "mixed" && material.Observations == "success" {
				output = `{"outcome":"complete","proposedPlan":null}`
			}
			if mode == "correlated" {
				switch material.Observations {
				case "first":
					output = `{"outcome":"retain","proposedPlan":null}`
				case "second":
					output = `{"outcome":"complete","proposedPlan":null}`
				}
			}
			status := "completed"
			threadID := "thread-stub"
			turnID := "turn-stub"
			phase := "final_answer"
			items := []any{map[string]any{
				"id":    "message-stub",
				"type":  "agentMessage",
				"phase": phase,
				"text":  output,
			}}
			switch mode {
			case "turn_failed":
				status = "failed"
				items = nil
			case "wrong_thread":
				threadID = "thread-other"
			case "wrong_turn":
				turnID = "turn-other"
			case "duplicate_final":
				items = append(items, map[string]any{"id": "message-stub-2", "type": "agentMessage", "phase": "final_answer", "text": output})
			case "missing_final":
				items = nil
			case "unknown_phase":
				items[0].(map[string]any)["phase"] = "unexpected"
			case "tool_item":
				items = append([]any{map[string]any{"id": "tool-stub", "type": "commandExecution"}}, items...)
			}
			completedNotification := map[string]any{
				"jsonrpc": "2.0",
				"method":  "turn/completed",
				"params": map[string]any{
					"threadId": threadID,
					"turn": map[string]any{
						"id":     turnID,
						"status": status,
						"items":  items,
					},
				},
			}
			if mode == "notification_with_id" {
				completedNotification["id"] = 99
			}
			encodedNotification, _ := json.Marshal(completedNotification)
			switch mode {
			case "duplicate_completed_thread_id":
				encodedNotification = bytes.Replace(encodedNotification, []byte(`"threadId":"thread-stub"`), []byte(`"threadId":"thread-stub","threadId":"thread-other"`), 1)
			case "duplicate_completed_turn_id":
				encodedNotification = bytes.Replace(encodedNotification, []byte(`"id":"turn-stub"`), []byte(`"id":"turn-stub","id":"turn-other"`), 1)
			case "duplicate_completed_status":
				encodedNotification = bytes.Replace(encodedNotification, []byte(`"status":"completed"`), []byte(`"status":"completed","status":"failed"`), 1)
			case "duplicate_final_text":
				encodedNotification = bytes.Replace(encodedNotification, []byte(`"text":`), []byte(`"text":"conflicting","text":`), 1)
			}
			_, _ = os.Stdout.Write(append(encodedNotification, '\n'))
		}
	}
}

func remainNonCooperativeUntilKilled() {
	for {
		time.Sleep(time.Hour)
	}
}
