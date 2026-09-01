package codexappserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var errAppServerInteraction = errors.New("codex app-server: interaction failed")

type clientRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

type clientNotification struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type serverMessage struct {
	id        json.RawMessage
	method    string
	params    json.RawMessage
	result    json.RawMessage
	hasID     bool
	hasResult bool
	hasError  bool
}

func decodeServerMessage(encoded []byte) (serverMessage, error) {
	fields, err := decodeMessageObject(encoded)
	if err != nil {
		return serverMessage{}, errAppServerInteraction
	}
	message := serverMessage{
		id:        fields["id"],
		params:    fields["params"],
		result:    fields["result"],
		hasID:     fields["id"] != nil,
		hasResult: fields["result"] != nil,
		hasError:  fields["error"] != nil,
	}
	if rawMethod := fields["method"]; rawMethod != nil {
		if json.Unmarshal(rawMethod, &message.method) != nil || message.method == "" {
			return serverMessage{}, errAppServerInteraction
		}
	}
	if message.method == "" {
		if !message.hasID || fields["params"] != nil || message.hasResult == message.hasError {
			return serverMessage{}, errAppServerInteraction
		}
	} else if message.hasResult || message.hasError {
		return serverMessage{}, errAppServerInteraction
	}
	return message, nil
}

func decodeMessageObject(encoded []byte) (map[string]json.RawMessage, error) {
	allowed := map[string]struct{}{
		"emittedAtMs": {},
		"error":       {},
		"id":          {},
		"method":      {},
		"params":      {},
		"result":      {},
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return nil, errAppServerInteraction
	}
	fields := make(map[string]json.RawMessage, len(allowed))
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, errAppServerInteraction
		}
		name, ok := token.(string)
		if !ok {
			return nil, errAppServerInteraction
		}
		if _, ok := allowed[name]; !ok {
			return nil, errAppServerInteraction
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, errAppServerInteraction
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, errAppServerInteraction
		}
		fields[name] = value
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return nil, errAppServerInteraction
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errAppServerInteraction
	}
	return fields, nil
}

func decodeResponseID(message serverMessage, expected int) error {
	var id int
	if json.Unmarshal(message.id, &id) != nil || id != expected || message.hasError || !message.hasResult {
		return errAppServerInteraction
	}
	return nil
}

func decodeObject(encoded json.RawMessage) (map[string]json.RawMessage, error) {
	if validateNoDuplicateJSONKeys(encoded) != nil {
		return nil, errAppServerInteraction
	}
	var value map[string]json.RawMessage
	if json.Unmarshal(encoded, &value) != nil || value == nil {
		return nil, errAppServerInteraction
	}
	return value, nil
}

func decodeNestedID(encoded json.RawMessage, container string) (string, error) {
	response, err := decodeObject(encoded)
	if err != nil {
		return "", err
	}
	nested, err := decodeObject(response[container])
	if err != nil {
		return "", err
	}
	var id string
	if json.Unmarshal(nested["id"], &id) != nil || id == "" {
		return "", errAppServerInteraction
	}
	return id, nil
}

func compatibleUserAgent(encoded json.RawMessage) bool {
	response, err := decodeObject(encoded)
	if err != nil {
		return false
	}
	var userAgent string
	if json.Unmarshal(response["userAgent"], &userAgent) != nil {
		return false
	}
	product, versionAndMetadata, found := strings.Cut(userAgent, "/")
	if !found || !strings.HasPrefix(product, "Codex ") || !strings.HasPrefix(versionAndMetadata, SupportedCodexVersion) {
		return false
	}
	afterVersion := versionAndMetadata[len(SupportedCodexVersion):]
	return afterVersion == "" || afterVersion[0] == ' ' || afterVersion[0] == '('
}

func safeEffectiveConfiguration(encoded json.RawMessage) bool {
	response, err := decodeObject(encoded)
	if err != nil {
		return false
	}
	config, err := decodeObject(response["config"])
	if err != nil {
		return false
	}
	if raw := config["apps"]; raw != nil && !allNamedConfigurationsDisabled(raw) {
		return false
	}
	if raw := config["mcp_servers"]; raw != nil && !allNamedConfigurationsDisabled(raw) {
		return false
	}
	if raw := config["hooks"]; raw != nil && !emptyEffectiveConfiguration(raw) {
		return false
	}
	if raw := config["web_search"]; raw != nil && string(raw) != "null" {
		var mode string
		if json.Unmarshal(raw, &mode) != nil || mode != "disabled" {
			return false
		}
	}
	if rawTools := config["tools"]; rawTools != nil && string(rawTools) != "null" {
		tools, err := decodeObject(rawTools)
		if err != nil {
			return false
		}
		if raw := tools["web_search"]; raw != nil && string(raw) != "null" {
			return false
		}
	}
	return true
}

func allNamedConfigurationsDisabled(encoded json.RawMessage) bool {
	if string(encoded) == "null" {
		return true
	}
	configurations, err := decodeObject(encoded)
	if err != nil {
		return false
	}
	for _, encodedConfiguration := range configurations {
		if string(encodedConfiguration) == "null" {
			continue
		}
		configuration, err := decodeObject(encodedConfiguration)
		if err != nil {
			return false
		}
		var enabled bool
		if rawEnabled := configuration["enabled"]; rawEnabled == nil || json.Unmarshal(rawEnabled, &enabled) != nil || enabled {
			return false
		}
	}
	return true
}

func emptyEffectiveConfiguration(encoded json.RawMessage) bool {
	var value any
	if json.Unmarshal(encoded, &value) != nil {
		return false
	}
	switch collection := value.(type) {
	case nil:
		return true
	case []any:
		for _, item := range collection {
			encodedItem, err := json.Marshal(item)
			if err != nil || !emptyEffectiveConfiguration(encodedItem) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, item := range collection {
			encodedItem, err := json.Marshal(item)
			if err != nil || !emptyEffectiveConfiguration(encodedItem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func validateCompletedTurn(encoded json.RawMessage, threadID, turnID string) error {
	params, err := decodeObject(encoded)
	if err != nil {
		return err
	}
	var completedThreadID string
	if json.Unmarshal(params["threadId"], &completedThreadID) != nil || completedThreadID != threadID {
		return errAppServerInteraction
	}
	turn, err := decodeObject(params["turn"])
	if err != nil {
		return err
	}
	var completedTurnID, status string
	if json.Unmarshal(turn["id"], &completedTurnID) != nil || completedTurnID != turnID ||
		json.Unmarshal(turn["status"], &status) != nil || status != "completed" {
		return errAppServerInteraction
	}
	return nil
}

func validateNotificationIdentity(encoded json.RawMessage, threadID, turnID string, requireTurn bool) error {
	params, err := decodeObject(encoded)
	if err != nil {
		return err
	}
	if !matchesNotificationID(params, "threadId", "thread", threadID) {
		return errAppServerInteraction
	}
	if requireTurn && !matchesNotificationID(params, "turnId", "turn", turnID) {
		return errAppServerInteraction
	}
	return nil
}

func matchesNotificationID(params map[string]json.RawMessage, directField, nestedField, expected string) bool {
	if rawDirect := params[directField]; rawDirect != nil {
		var actual string
		return json.Unmarshal(rawDirect, &actual) == nil && actual == expected
	}
	nested, err := decodeObject(params[nestedField])
	if err != nil {
		return false
	}
	var actual string
	return json.Unmarshal(nested["id"], &actual) == nil && actual == expected
}

func decodeCompletedItem(encoded json.RawMessage, threadID, turnID string) (string, bool, error) {
	params, err := decodeObject(encoded)
	if err != nil {
		return "", false, err
	}
	var completedThreadID, completedTurnID string
	if json.Unmarshal(params["threadId"], &completedThreadID) != nil || completedThreadID != threadID ||
		json.Unmarshal(params["turnId"], &completedTurnID) != nil || completedTurnID != turnID {
		return "", false, errAppServerInteraction
	}
	item, err := decodeObject(params["item"])
	if err != nil {
		return "", false, err
	}
	var itemType string
	if json.Unmarshal(item["type"], &itemType) != nil || itemType == "" {
		return "", false, errAppServerInteraction
	}
	switch itemType {
	case "commandExecution", "contextCompaction", "imageView", "plan", "reasoning", "sleep", "userMessage":
		return "", false, nil
	case "agentMessage":
	default:
		return "", false, errAppServerInteraction
	}
	var phase string
	if rawPhase := item["phase"]; rawPhase != nil && string(rawPhase) != "null" {
		if json.Unmarshal(rawPhase, &phase) != nil {
			return "", false, errAppServerInteraction
		}
	}
	if phase == "commentary" {
		return "", false, nil
	}
	if phase != "" && phase != "final_answer" {
		return "", false, errAppServerInteraction
	}
	var finalOutput string
	if json.Unmarshal(item["text"], &finalOutput) != nil || strings.TrimSpace(finalOutput) == "" {
		return "", false, errAppServerInteraction
	}
	return finalOutput, true, nil
}

func validateNoDuplicateJSONKeys(encoded []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if err := validateJSONValue(decoder); err != nil {
		return errAppServerInteraction
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return errAppServerInteraction
	}
	return nil
}

func validateJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errAppServerInteraction
			}
			if _, duplicate := seen[key]; duplicate {
				return errAppServerInteraction
			}
			seen[key] = struct{}{}
			if err := validateJSONValue(decoder); err != nil {
				return err
			}
		}
		if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
			return errAppServerInteraction
		}
	case '[':
		for decoder.More() {
			if err := validateJSONValue(decoder); err != nil {
				return err
			}
		}
		if closing, err := decoder.Token(); err != nil || closing != json.Delim(']') {
			return errAppServerInteraction
		}
	default:
		return errAppServerInteraction
	}
	return nil
}
