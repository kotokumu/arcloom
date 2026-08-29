package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type request struct {
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	JSONRPC json.RawMessage `json:"jsonrpc"`
}

type capture struct {
	PID             int             `json:"pid"`
	Arguments       []string        `json:"arguments"`
	Initialize      json.RawMessage `json:"initialize"`
	ConfigRead      json.RawMessage `json:"configRead"`
	ThreadStart     json.RawMessage `json:"threadStart"`
	ThreadResponded bool            `json:"threadResponded"`
	TurnStart       json.RawMessage `json:"turnStart"`
	StdinClosed     bool            `json:"stdinClosed"`
	JSONRPC         bool            `json:"jsonrpc"`
	Messages        []message       `json:"messages"`
}

type message struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func main() {
	mode := os.Getenv("ARCLOOM_CODEX_STUB_MODE")
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	record := capture{PID: os.Getpid(), Arguments: append([]string(nil), os.Args[1:]...)}
	writeCapture(record)
	for {
		var incoming request
		if err := decoder.Decode(&incoming); err != nil {
			record.StdinClosed = err == io.EOF
			writeCapture(record)
			return
		}
		record.JSONRPC = record.JSONRPC || len(incoming.JSONRPC) != 0
		record.Messages = append(record.Messages, message{
			ID: append(json.RawMessage(nil), incoming.ID...), Method: incoming.Method, Params: append(json.RawMessage(nil), incoming.Params...),
		})
		switch incoming.Method {
		case "initialize":
			record.Initialize = append(json.RawMessage(nil), incoming.Params...)
			writeCapture(record)
			switch mode {
			case "malformed":
				_, _ = fmt.Fprintln(os.Stdout, `{not-json`)
				continue
			case "duplicate":
				_, _ = fmt.Fprintln(os.Stdout, `{"id":1,"id":1,"result":{"userAgent":"Codex Desktop/0.149.1 (test)"}}`)
				continue
			case "mismatched_response":
				respond(encoder, json.RawMessage("99"), map[string]any{"userAgent": supportedUserAgent()})
				continue
			case "server_request":
				_ = encoder.Encode(map[string]any{
					"id":     99,
					"method": "item/commandExecution/requestApproval",
					"params": map[string]any{},
				})
				continue
			case "error_response":
				_ = encoder.Encode(map[string]any{"id": incoming.ID, "error": map[string]any{"code": -32603, "message": "failed"}})
				continue
			}
			version := "0.149.1"
			if mode == "incompatible_version" {
				version = "0.150.0"
			}
			userAgent := "Codex Desktop/" + version + " (test)"
			if mode == "other_product" {
				userAgent = "Other Product/" + version + " (test)"
			}
			if mode == "missing_user_agent" {
				userAgent = ""
			}
			responseID := incoming.ID
			result := map[string]any{
				"codexHome":      "/tmp/codex-home",
				"platformFamily": "unix",
				"platformOs":     "test",
			}
			if userAgent != "" {
				result["userAgent"] = userAgent
			}
			respond(encoder, responseID, result)
		case "initialized":
		case "config/read":
			record.ConfigRead = append(json.RawMessage(nil), incoming.Params...)
			writeCapture(record)
			config := map[string]any{
				"apps":        map[string]any{},
				"hooks":       []any{},
				"mcp_servers": map[string]any{},
				"tools":       map[string]any{"web_search": nil},
				"web_search":  "disabled",
			}
			switch mode {
			case "unsafe_mcp":
				config["mcp_servers"] = map[string]any{"filesystem": map[string]any{}}
			case "unsafe_apps":
				config["apps"] = map[string]any{"drive": map[string]any{"enabled": true}}
			case "unsafe_hooks":
				config["hooks"] = []any{map[string]any{"event": "turn"}}
			case "unsafe_web_search":
				config["web_search"] = "live"
			case "unsafe_web_tool":
				config["tools"] = map[string]any{"web_search": map[string]any{}}
			case "disabled_mcp":
				config["mcp_servers"] = map[string]any{"filesystem": map[string]any{"enabled": false}}
			case "disabled_apps":
				config["apps"] = map[string]any{"drive": map[string]any{"enabled": false}, "_default": nil}
			case "empty_hooks":
				config["hooks"] = map[string]any{"PreToolUse": []any{}, "managedDir": nil}
			case "invalid_mcp_shape":
				config["mcp_servers"] = "unknown"
			}
			responseID := incoming.ID
			if mode == "mismatched_config_response" {
				responseID = json.RawMessage("99")
			}
			respond(encoder, responseID, map[string]any{
				"config":  config,
				"origins": map[string]any{},
			})
		case "thread/start":
			record.ThreadStart = append(json.RawMessage(nil), incoming.Params...)
			writeCapture(record)
			responseID := incoming.ID
			if mode == "mismatched_thread_response" {
				responseID = json.RawMessage("99")
			}
			respond(encoder, responseID, map[string]any{"thread": map[string]any{"id": "thread-stub"}})
			record.ThreadResponded = true
			writeCapture(record)
			if mode == "stop_before_turn_read" {
				time.Sleep(time.Hour)
			}
		case "turn/start":
			record.TurnStart = append(json.RawMessage(nil), incoming.Params...)
			writeCapture(record)
			if mode == "stderr_flood" {
				_, _ = os.Stderr.WriteString(strings.Repeat("stderr payload\n", 32*1024))
			}
			if mode == "early_completion" {
				complete(encoder, "thread-stub", "turn-stub", "completed", finalOutput(mode, incoming.Params))
			}
			if mode == "notification_burst" {
				for range 256 {
					_ = encoder.Encode(map[string]any{
						"method": "turn/started",
						"params": map[string]any{"threadId": "thread-stub", "turn": map[string]any{"id": "turn-stub"}},
					})
				}
			}
			if mode == "mismatched_turn_notification" {
				_ = encoder.Encode(map[string]any{
					"method": "turn/started",
					"params": map[string]any{"threadId": "thread-stub", "turn": map[string]any{"id": "turn-other"}},
				})
			}
			if mode == "unknown_notification" {
				_ = encoder.Encode(map[string]any{"method": "future/unknown", "params": map[string]any{}})
			}
			responseID := incoming.ID
			if mode == "mismatched_turn_response" {
				responseID = json.RawMessage("99")
			}
			respond(encoder, responseID, map[string]any{"turn": map[string]any{"id": "turn-stub"}})
			if mode == "thread_settings_notification" || mode == "mismatched_thread_settings_notification" {
				threadID := "thread-stub"
				if mode == "mismatched_thread_settings_notification" {
					threadID = "thread-other"
				}
				_ = encoder.Encode(map[string]any{
					"method": "thread/settings/updated",
					"params": map[string]any{
						"threadId":       threadID,
						"threadSettings": map[string]any{"model": "test-model"},
					},
				})
			}
			if mode == "account_rate_limits_notification" {
				_ = encoder.Encode(map[string]any{
					"method": "account/rateLimits/updated",
					"params": map[string]any{"rateLimits": map[string]any{}},
				})
			}
			switch mode {
			case "mixed":
				output := finalOutput(mode, incoming.Params)
				if output == "cancel material" {
					if path := os.Getenv("ARCLOOM_CODEX_STUB_CAPTURE"); path != "" {
						_ = os.WriteFile(path+".cancel-ready", []byte("ready"), 0o600)
					}
					continue
				}
				complete(encoder, "thread-stub", "turn-stub", "completed", output)
			case "cooperative_cancel":
				continue
			case "cancel_exit_error":
				var trailing any
				if err := decoder.Decode(&trailing); err == io.EOF {
					record.StdinClosed = true
					writeCapture(record)
				}
				os.Exit(2)
			case "forced_cancel":
				time.Sleep(time.Hour)
			case "completed_hang":
				complete(encoder, "thread-stub", "turn-stub", "completed", finalOutput(mode, incoming.Params))
				var trailing any
				if err := decoder.Decode(&trailing); err == io.EOF {
					record.StdinClosed = true
					writeCapture(record)
				}
				time.Sleep(time.Hour)
			case "early_completion":
				continue
			case "failed_turn":
				complete(encoder, "thread-stub", "turn-stub", "failed", finalOutput(mode, incoming.Params))
			case "interrupted_turn":
				complete(encoder, "thread-stub", "turn-stub", "interrupted", finalOutput(mode, incoming.Params))
			case "missing_final":
				complete(encoder, "thread-stub", "turn-stub", "completed", "")
			case "mismatched_turn":
				complete(encoder, "thread-other", "turn-stub", "completed", finalOutput(mode, incoming.Params))
			case "mismatched_completion_turn":
				complete(encoder, "thread-stub", "turn-other", "completed", finalOutput(mode, incoming.Params))
			default:
				complete(encoder, "thread-stub", "turn-stub", "completed", finalOutput(mode, incoming.Params))
			}
			if mode == "fast_exit" {
				return
			}
		}
	}
}

func supportedUserAgent() string {
	return "Codex Desktop/0.149.1 (test)"
}

func respond(encoder *json.Encoder, id json.RawMessage, result any) {
	_ = encoder.Encode(map[string]any{"id": id, "result": result})
}

func complete(encoder *json.Encoder, threadID, turnID, status, output string) {
	mode := os.Getenv("ARCLOOM_CODEX_STUB_MODE")
	items := []any{map[string]any{
		"id":   "reasoning-stub",
		"type": "reasoning",
	}}
	if mode == "file_change_item" {
		items = append(items, map[string]any{"id": "change-stub", "type": "fileChange"})
	}
	if mode == "mcp_item" {
		items = append(items, map[string]any{"id": "mcp-stub", "type": "mcpToolCall"})
	}
	if mode == "web_search_item" {
		items = append(items, map[string]any{"id": "web-stub", "type": "webSearch"})
	}
	if output != "" {
		finalMessage := map[string]any{
			"id":    "message-stub",
			"type":  "agentMessage",
			"phase": "final_answer",
			"text":  output,
		}
		if mode == "missing_phase" {
			delete(finalMessage, "phase")
		}
		if mode == "unknown_phase" {
			finalMessage["phase"] = "analysis"
		}
		items = append(items,
			map[string]any{
				"id":    "commentary-stub",
				"type":  "agentMessage",
				"phase": "commentary",
				"text":  "working",
			},
			finalMessage,
		)
		if mode == "multiple_final" {
			items = append(items, map[string]any{
				"id": "message-stub-2", "type": "agentMessage", "phase": "final_answer", "text": "another final",
			})
		}
	}
	for _, item := range items {
		itemThreadID := threadID
		if mode == "mismatched_item" {
			itemThreadID = "thread-other"
		}
		_ = encoder.Encode(map[string]any{
			"method": "item/completed",
			"params": map[string]any{
				"completedAtMs": 1,
				"item":          item,
				"threadId":      itemThreadID,
				"turnId":        turnID,
			},
		})
	}
	_ = encoder.Encode(map[string]any{
		"method": "turn/completed",
		"params": map[string]any{
			"threadId": threadID,
			"turn": map[string]any{
				"id":     turnID,
				"status": status,
				"items":  items,
			},
		},
	})
}

func finalOutput(mode string, params json.RawMessage) string {
	switch mode {
	case "large_message":
		return strings.Repeat("x", 96*1024)
	case "echo", "mixed":
		var turn struct {
			Input []struct {
				Text string `json:"text"`
			} `json:"input"`
		}
		if json.Unmarshal(params, &turn) == nil && len(turn.Input) == 1 {
			return turn.Input[0].Text
		}
	}
	return `{"outcome":"complete"}`
}

func writeCapture(record capture) {
	if path := os.Getenv("ARCLOOM_CODEX_STUB_CAPTURE"); path != "" {
		encoded, _ := json.Marshal(record)
		_ = os.WriteFile(path, encoded, 0o600)
		if os.Getenv("ARCLOOM_CODEX_STUB_MODE") == "mixed" {
			_ = os.WriteFile(fmt.Sprintf("%s.%d", path, record.PID), encoded, 0o600)
		}
	}
}
