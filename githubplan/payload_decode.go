package githubplan

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"
)

type payloadStringOutcome struct {
	value     string
	available bool
}

type payloadConditionsOutcome struct {
	members   []string
	available bool
	complete  bool
}

type payloadTargetState uint8

const (
	payloadTargetUnavailable payloadTargetState = iota + 1
	payloadTargetAbsent
	payloadTargetPresent
)

type payloadTargetOutcome struct {
	state payloadTargetState
	value string
}

type payloadOutcome struct {
	goal       payloadStringOutcome
	conditions payloadConditionsOutcome
	targetDate payloadTargetOutcome
}

func unavailablePayloadOutcome() payloadOutcome {
	return payloadOutcome{
		conditions: payloadConditionsOutcome{},
		targetDate: payloadTargetOutcome{state: payloadTargetUnavailable},
	}
}

func decodePayload(content string, shape payloadShape) payloadOutcome {
	outcome := unavailablePayloadOutcome()
	if !strings.HasPrefix(content, payloadPrefix) {
		return outcome
	}
	remainder := content[len(payloadPrefix):]
	suffix := strings.Index(remainder, "\n-->")
	if suffix < 0 {
		return outcome
	}
	encoded := remainder[:suffix]
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || base64.RawURLEncoding.EncodeToString(decoded) != encoded || !utf8.Valid(decoded) {
		return outcome
	}
	object, ok := parsePayloadObject(decoded)
	if !ok {
		return outcome
	}
	if raw, exists := object["goal"]; exists && raw.kind == payloadJSONString {
		if value, valid := decodePayloadJSONString(raw.raw); valid {
			outcome.goal = payloadStringOutcome{value: value, available: true}
		}
	}
	if raw, exists := object["acceptance_conditions"]; exists && raw.kind == payloadJSONArray {
		outcome.conditions.available = true
		outcome.conditions.complete = true
		seen := make(map[string]struct{}, len(raw.array))
		for _, member := range raw.array {
			if member.kind != payloadJSONString {
				outcome.conditions.complete = false
				continue
			}
			value, valid := decodePayloadJSONString(member.raw)
			if !valid {
				outcome.conditions.complete = false
				continue
			}
			if _, exists := seen[value]; exists {
				outcome.conditions.complete = false
			} else {
				seen[value] = struct{}{}
			}
			outcome.conditions.members = append(outcome.conditions.members, value)
		}
	}
	if shape == payloadWithTargetDate {
		if raw, exists := object["target_date"]; exists {
			switch raw.kind {
			case payloadJSONNull:
				outcome.targetDate.state = payloadTargetAbsent
			case payloadJSONString:
				if value, valid := decodePayloadJSONString(raw.raw); valid {
					outcome.targetDate = payloadTargetOutcome{state: payloadTargetPresent, value: value}
				}
			}
		}
	}
	return outcome
}

type payloadJSONKind uint8

const (
	payloadJSONString payloadJSONKind = iota + 1
	payloadJSONObject
	payloadJSONArray
	payloadJSONNull
	payloadJSONBoolean
	payloadJSONNumber
)

type payloadJSONValue struct {
	kind  payloadJSONKind
	raw   []byte
	array []payloadJSONValue
}

type payloadJSONParser struct {
	input []byte
	index int
}

func parsePayloadObject(input []byte) (map[string]payloadJSONValue, bool) {
	parser := payloadJSONParser{input: input}
	parser.skipWhitespace()
	if parser.index >= len(input) || input[parser.index] != '{' {
		return nil, false
	}
	object, ok := parser.parseTopObject()
	if !ok {
		return nil, false
	}
	parser.skipWhitespace()
	if parser.index != len(input) {
		return nil, false
	}
	return object, true
}

func (p *payloadJSONParser) parseTopObject() (map[string]payloadJSONValue, bool) {
	p.index++
	object := make(map[string]payloadJSONValue)
	p.skipWhitespace()
	if p.index < len(p.input) && p.input[p.index] == '}' {
		p.index++
		return object, true
	}
	for {
		nameRaw, ok := p.parseJSONStringRaw()
		if !ok {
			return nil, false
		}
		name, ok := decodePayloadJSONString(nameRaw)
		if !ok {
			return nil, false
		}
		if _, exists := object[name]; exists {
			return nil, false
		}
		p.skipWhitespace()
		if p.index >= len(p.input) || p.input[p.index] != ':' {
			return nil, false
		}
		p.index++
		value, ok := p.parseValue()
		if !ok {
			return nil, false
		}
		object[name] = value
		p.skipWhitespace()
		if p.index >= len(p.input) {
			return nil, false
		}
		if p.input[p.index] == '}' {
			p.index++
			return object, true
		}
		if p.input[p.index] != ',' {
			return nil, false
		}
		p.index++
		p.skipWhitespace()
	}
}

func (p *payloadJSONParser) parseValue() (payloadJSONValue, bool) {
	p.skipWhitespace()
	if p.index >= len(p.input) {
		return payloadJSONValue{}, false
	}
	start := p.index
	switch p.input[p.index] {
	case '"':
		raw, ok := p.parseJSONStringRaw()
		return payloadJSONValue{kind: payloadJSONString, raw: raw}, ok
	case '{':
		if !p.parseNestedObject() {
			return payloadJSONValue{}, false
		}
		return payloadJSONValue{kind: payloadJSONObject, raw: p.input[start:p.index]}, true
	case '[':
		members, ok := p.parseArray()
		return payloadJSONValue{kind: payloadJSONArray, raw: p.input[start:p.index], array: members}, ok
	case 'n':
		if p.consumeLiteral("null") {
			return payloadJSONValue{kind: payloadJSONNull, raw: p.input[start:p.index]}, true
		}
	case 't':
		if p.consumeLiteral("true") {
			return payloadJSONValue{kind: payloadJSONBoolean, raw: p.input[start:p.index]}, true
		}
	case 'f':
		if p.consumeLiteral("false") {
			return payloadJSONValue{kind: payloadJSONBoolean, raw: p.input[start:p.index]}, true
		}
	default:
		if p.input[p.index] == '-' || p.input[p.index] >= '0' && p.input[p.index] <= '9' {
			if p.parseNumber() {
				return payloadJSONValue{kind: payloadJSONNumber, raw: p.input[start:p.index]}, true
			}
		}
	}
	return payloadJSONValue{}, false
}

func (p *payloadJSONParser) parseNestedObject() bool {
	p.index++
	p.skipWhitespace()
	if p.index < len(p.input) && p.input[p.index] == '}' {
		p.index++
		return true
	}
	for {
		if _, ok := p.parseJSONStringRaw(); !ok {
			return false
		}
		p.skipWhitespace()
		if p.index >= len(p.input) || p.input[p.index] != ':' {
			return false
		}
		p.index++
		if _, ok := p.parseValue(); !ok {
			return false
		}
		p.skipWhitespace()
		if p.index >= len(p.input) {
			return false
		}
		if p.input[p.index] == '}' {
			p.index++
			return true
		}
		if p.input[p.index] != ',' {
			return false
		}
		p.index++
		p.skipWhitespace()
	}
}

func (p *payloadJSONParser) parseArray() ([]payloadJSONValue, bool) {
	p.index++
	members := make([]payloadJSONValue, 0)
	p.skipWhitespace()
	if p.index < len(p.input) && p.input[p.index] == ']' {
		p.index++
		return members, true
	}
	for {
		member, ok := p.parseValue()
		if !ok {
			return nil, false
		}
		members = append(members, member)
		p.skipWhitespace()
		if p.index >= len(p.input) {
			return nil, false
		}
		if p.input[p.index] == ']' {
			p.index++
			return members, true
		}
		if p.input[p.index] != ',' {
			return nil, false
		}
		p.index++
		p.skipWhitespace()
	}
}

func (p *payloadJSONParser) parseJSONStringRaw() ([]byte, bool) {
	start := p.index
	if p.index >= len(p.input) || p.input[p.index] != '"' {
		return nil, false
	}
	p.index++
	for p.index < len(p.input) {
		value := p.input[p.index]
		switch value {
		case '"':
			p.index++
			return p.input[start:p.index], true
		case '\\':
			p.index++
			if p.index >= len(p.input) {
				return nil, false
			}
			if p.input[p.index] == 'u' {
				if p.index+4 >= len(p.input) {
					return nil, false
				}
				for offset := 1; offset <= 4; offset++ {
					if !isHexDigit(p.input[p.index+offset]) {
						return nil, false
					}
				}
				p.index += 5
				continue
			}
			if !strings.ContainsRune(`"\\/bfnrt`, rune(p.input[p.index])) {
				return nil, false
			}
			p.index++
		default:
			if value < 0x20 {
				return nil, false
			}
			if value < utf8.RuneSelf {
				p.index++
				continue
			}
			_, size := utf8.DecodeRune(p.input[p.index:])
			if size == 0 || p.index+size > len(p.input) {
				return nil, false
			}
			p.index += size
		}
	}
	return nil, false
}

func (p *payloadJSONParser) parseNumber() bool {
	if p.index < len(p.input) && p.input[p.index] == '-' {
		p.index++
	}
	if p.index >= len(p.input) {
		return false
	}
	if p.input[p.index] == '0' {
		p.index++
	} else {
		if p.input[p.index] < '1' || p.input[p.index] > '9' {
			return false
		}
		for p.index < len(p.input) && p.input[p.index] >= '0' && p.input[p.index] <= '9' {
			p.index++
		}
	}
	if p.index < len(p.input) && p.input[p.index] == '.' {
		p.index++
		start := p.index
		for p.index < len(p.input) && p.input[p.index] >= '0' && p.input[p.index] <= '9' {
			p.index++
		}
		if start == p.index {
			return false
		}
	}
	if p.index < len(p.input) && (p.input[p.index] == 'e' || p.input[p.index] == 'E') {
		p.index++
		if p.index < len(p.input) && (p.input[p.index] == '+' || p.input[p.index] == '-') {
			p.index++
		}
		start := p.index
		for p.index < len(p.input) && p.input[p.index] >= '0' && p.input[p.index] <= '9' {
			p.index++
		}
		if start == p.index {
			return false
		}
	}
	return true
}

func (p *payloadJSONParser) consumeLiteral(value string) bool {
	if !strings.HasPrefix(string(p.input[p.index:]), value) {
		return false
	}
	p.index += len(value)
	return true
}

func (p *payloadJSONParser) skipWhitespace() {
	for p.index < len(p.input) {
		switch p.input[p.index] {
		case ' ', '\t', '\n', '\r':
			p.index++
		default:
			return
		}
	}
}

func decodePayloadJSONString(raw []byte) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}
	var builder strings.Builder
	for index := 1; index < len(raw)-1; {
		value := raw[index]
		if value != '\\' {
			if value < 0x20 {
				return "", false
			}
			runeValue, size := utf8.DecodeRune(raw[index:])
			if runeValue == utf8.RuneError && size == 1 {
				return "", false
			}
			builder.WriteRune(runeValue)
			index += size
			continue
		}
		index++
		if index >= len(raw)-1 {
			return "", false
		}
		switch raw[index] {
		case '"', '\\', '/':
			builder.WriteByte(raw[index])
			index++
		case 'b':
			builder.WriteByte('\b')
			index++
		case 'f':
			builder.WriteByte('\f')
			index++
		case 'n':
			builder.WriteByte('\n')
			index++
		case 'r':
			builder.WriteByte('\r')
			index++
		case 't':
			builder.WriteByte('\t')
			index++
		case 'u':
			if index+4 >= len(raw) {
				return "", false
			}
			code, ok := parseHexCodeUnit(raw[index+1 : index+5])
			if !ok {
				return "", false
			}
			index += 5
			if code >= 0xd800 && code <= 0xdbff {
				if index+6 > len(raw)-1 || raw[index] != '\\' || raw[index+1] != 'u' {
					return "", false
				}
				low, ok := parseHexCodeUnit(raw[index+2 : index+6])
				if !ok || low < 0xdc00 || low > 0xdfff {
					return "", false
				}
				index += 6
				builder.WriteRune(rune(0x10000 + (int(code)-0xd800)*0x400 + int(low) - 0xdc00))
			} else if code >= 0xdc00 && code <= 0xdfff {
				return "", false
			} else {
				builder.WriteRune(rune(code))
			}
		default:
			return "", false
		}
	}
	return builder.String(), true
}

func parseHexCodeUnit(value []byte) (uint16, bool) {
	if len(value) != 4 {
		return 0, false
	}
	var result uint16
	for _, digit := range value {
		if !isHexDigit(digit) {
			return 0, false
		}
		result = result*16 + uint16(hexValue(digit))
	}
	return result, true
}

func isHexDigit(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'a' && value <= 'f' || value >= 'A' && value <= 'F'
}

func hexValue(value byte) byte {
	switch {
	case value >= '0' && value <= '9':
		return value - '0'
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10
	default:
		return value - 'A' + 10
	}
}
