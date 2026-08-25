package githubplan

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

type textFact struct {
	value     string
	available bool
}

type dateFactState uint8

const (
	dateUnavailable dateFactState = iota
	dateAbsent
	datePresent
)

type dateFact struct {
	state dateFactState
	value string
}

type nativeState uint8

const (
	nativeStateUnknown nativeState = iota
	nativeStateOpen
	nativeStateClosed
)

// rootFact contains only GitHub-owned facts. Plan validation and location
// correspondence are deliberately deferred to the selected scheme.
type rootFact struct {
	number  int64
	title   textFact
	content textFact
	date    dateFact
	state   nativeState
}

type rootFactOutcome struct {
	fact      rootFact
	available bool
}

func decodeMilestoneRoot(document responseDocument, number ResourceNumber) rootFactOutcome {
	object, ok := decodeJSONObject(document.bytes)
	if !ok {
		return rootFactOutcome{}
	}
	rootNumber, ok := rawPositiveInt(object["number"])
	if !ok || rootNumber != number.value {
		return rootFactOutcome{}
	}
	fact := rootFact{number: rootNumber, date: dateFact{state: dateAbsent}, state: decodeNativeState(object["state"])}
	if value, exists := rawString(object["title"]); exists {
		fact.title = textFact{value: value, available: true}
	}
	if value, exists := rawString(object["description"]); exists {
		fact.content = textFact{value: value, available: true}
	}
	if raw, exists := object["due_on"]; exists && strings.TrimSpace(string(raw)) != "null" {
		if value, valid := rawString(raw); valid {
			fact.date = dateFact{state: datePresent, value: value}
		} else {
			fact.date = dateFact{state: dateUnavailable}
		}
	}
	return rootFactOutcome{fact: fact, available: true}
}

func decodeNativeState(raw json.RawMessage) nativeState {
	value, ok := rawString(raw)
	if !ok {
		return nativeStateUnknown
	}
	switch value {
	case "open":
		return nativeStateOpen
	case "closed":
		return nativeStateClosed
	default:
		return nativeStateUnknown
	}
}

func decodeJSONObject(body []byte) (map[string]json.RawMessage, bool) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || trimmed[0] != '{' {
		return nil, false
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &object); err != nil || object == nil {
		return nil, false
	}
	return object, true
}

func rawString(raw json.RawMessage) (string, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '"' || trimmed[len(trimmed)-1] != '"' {
		return "", false
	}
	if !utf8.Valid(trimmed) {
		return string(trimmed[1 : len(trimmed)-1]), true
	}
	var value string
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return "", false
	}
	return value, true
}

func rawPositiveInt(raw json.RawMessage) (int64, bool) {
	value := strings.TrimSpace(string(raw))
	if value == "" || strings.ContainsAny(value, ".eE") {
		return 0, false
	}
	parsed := int64(0)
	for index, digit := range value {
		if digit < '0' || digit > '9' || (index == 0 && digit == '0' && len(value) > 1) {
			return 0, false
		}
		if parsed > (1<<63-1-int64(digit-'0'))/10 {
			return 0, false
		}
		parsed = parsed*10 + int64(digit-'0')
	}
	return parsed, parsed > 0
}
