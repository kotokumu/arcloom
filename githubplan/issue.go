package githubplan

import "strings"

func decodeIssueRoot(document responseDocument, number ResourceNumber) rootFactOutcome {
	object, ok := decodeJSONObject(document.bytes)
	if !ok {
		return rootFactOutcome{}
	}
	rootNumber, ok := rawPositiveInt(object["number"])
	if !ok || rootNumber != number.value {
		return rootFactOutcome{}
	}
	if marker, exists := object["pull_request"]; exists && strings.TrimSpace(string(marker)) != "null" {
		return rootFactOutcome{}
	}
	fact := rootFact{number: rootNumber, date: dateFact{state: dateUnavailable}}
	if value, exists := rawString(object["title"]); exists {
		fact.title = textFact{value: value, available: true}
	}
	if value, exists := rawString(object["body"]); exists {
		fact.content = textFact{value: value, available: true}
	}
	return rootFactOutcome{fact: fact, available: true}
}
