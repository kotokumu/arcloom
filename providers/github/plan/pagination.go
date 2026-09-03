package githubplan

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

type nextPageState uint8

const (
	nextPageAbsent nextPageState = iota
	nextPageValid
	nextPageRejected
)

type nextPageOutcome struct {
	target *url.URL
	state  nextPageState
}

func (s scheme) nextPage(binding observerBinding, current *url.URL, values []string) nextPageOutcome {
	target, hasNext, valid := parseNextLink(values)
	if !valid {
		return nextPageOutcome{state: nextPageRejected}
	}
	if !hasNext {
		return nextPageOutcome{state: nextPageAbsent}
	}
	if !s.validNextPage(binding, current, target) {
		return nextPageOutcome{state: nextPageRejected}
	}
	return nextPageOutcome{target: target, state: nextPageValid}
}

func parseNextLink(values []string) (*url.URL, bool, bool) {
	var target *url.URL
	hasNext := false
	for _, value := range values {
		parts, valid := splitLinkValue(value, ',')
		if !valid {
			return nil, false, false
		}
		for _, part := range parts {
			candidate, nextCount, valid := parseLinkPart(part)
			if !valid {
				return nil, false, false
			}
			if nextCount == 0 {
				continue
			}
			if nextCount != 1 {
				return nil, false, false
			}
			if hasNext {
				return nil, false, false
			}
			hasNext = true
			target = candidate
		}
	}
	return target, hasNext, true
}

func splitLinkValue(value string, separator byte) ([]string, bool) {
	parts := make([]string, 0, 1)
	start := 0
	inAngle, inQuote, escaped := false, false, false
	for index := 0; index < len(value); index++ {
		current := value[index]
		if escaped {
			escaped = false
			continue
		}
		if inQuote && current == '\\' {
			escaped = true
			continue
		}
		switch current {
		case '<':
			if !inQuote {
				inAngle = true
			}
		case '>':
			if !inQuote {
				inAngle = false
			}
		case '"':
			if !inAngle {
				inQuote = !inQuote
			}
		default:
			if current == separator && !inAngle && !inQuote {
				parts = append(parts, value[start:index])
				start = index + 1
			}
		}
	}
	if inAngle || inQuote || escaped {
		return nil, false
	}
	parts = append(parts, value[start:])
	return parts, true
}

func parseLinkPart(part string) (*url.URL, int, bool) {
	trimmed := strings.TrimSpace(part)
	if trimmed == "" || trimmed[0] != '<' {
		return nil, 0, false
	}
	end := strings.IndexByte(trimmed, '>')
	if end <= 1 {
		return nil, 0, false
	}
	target, err := url.Parse(trimmed[1:end])
	if err != nil {
		return nil, 0, false
	}
	rest := strings.TrimSpace(trimmed[end+1:])
	if rest == "" {
		return target, 0, true
	}
	if rest[0] != ';' {
		return nil, 0, false
	}
	parameters, valid := splitLinkValue(rest[1:], ';')
	if !valid {
		return nil, 0, false
	}
	nextCount := 0
	for _, parameter := range parameters {
		parameter = strings.TrimSpace(parameter)
		if parameter == "" {
			return nil, 0, false
		}
		key, value, found := strings.Cut(parameter, "=")
		if !found || strings.TrimSpace(key) == "" {
			return nil, 0, false
		}
		value, valid = linkParameterValue(strings.TrimSpace(value))
		if !valid {
			return nil, 0, false
		}
		if strings.EqualFold(strings.TrimSpace(key), "rel") {
			for _, relation := range strings.Fields(value) {
				if relation == "next" {
					nextCount++
				}
			}
		}
	}
	return target, nextCount, true
}

func linkParameterValue(value string) (string, bool) {
	if value == "" {
		return "", false
	}
	if value[0] != '"' {
		if strings.ContainsAny(value, "\t\r\n ") {
			return "", false
		}
		return value, true
	}
	if len(value) < 2 || value[len(value)-1] != '"' {
		return "", false
	}
	var builder strings.Builder
	for index := 1; index < len(value)-1; index++ {
		current := value[index]
		if current != '\\' {
			builder.WriteByte(current)
			continue
		}
		index++
		if index >= len(value)-1 {
			return "", false
		}
		switch value[index] {
		case '\\', '"':
			builder.WriteByte(value[index])
		default:
			return "", false
		}
	}
	return builder.String(), true
}

func (s scheme) validNextPage(binding observerBinding, current, target *url.URL) bool {
	if target == nil || target.Scheme != "https" || target.Host != "api.github.com" || target.User != nil || target.Fragment != "" || target.Opaque != "" {
		return false
	}
	expected := s.taskURL(binding)
	if current == nil || current.Path != expected.Path || target.Path != expected.Path {
		return false
	}
	query, err := url.ParseQuery(target.RawQuery)
	if err != nil {
		return false
	}
	allowed := map[string]string{"page": ""}
	if s == milestoneScheme {
		allowed["milestone"] = strconv.FormatInt(binding.number.value, 10)
		allowed["state"] = "all"
	}
	allowed["per_page"] = "100"
	if len(query) != len(allowed) {
		return false
	}
	for key, expectedValue := range allowed {
		values := query[key]
		if len(values) != 1 {
			return false
		}
		if key != "page" && values[0] != expectedValue {
			return false
		}
	}
	_, ok := positivePage(query.Get("page"))
	return ok
}

func positivePage(value string) (int64, bool) {
	return rawPositiveInt(json.RawMessage(value))
}
