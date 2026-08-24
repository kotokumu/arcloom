package githubplan

import (
	"encoding/base64"
	"strings"

	"github.com/kotokumu/arcloom/plan"
)

const payloadPrefix = "<!-- arcloom-plan:v1\n"
const payloadSuffix = "\n-->\n\n"

type payloadShape uint8

const (
	payloadWithoutTargetDate payloadShape = iota + 1
	payloadWithTargetDate
)

func payloadFor(value plan.Plan, shape payloadShape) string {
	var json strings.Builder
	json.WriteString(`{"goal":`)
	appendJSONString(&json, value.Goal().Text())
	json.WriteString(`,"acceptance_conditions":[`)
	for index, condition := range value.AcceptanceConditions() {
		if index > 0 {
			json.WriteByte(',')
		}
		appendJSONString(&json, condition.Statement())
	}
	json.WriteByte(']')
	if shape == payloadWithTargetDate {
		json.WriteString(`,"target_date":`)
		if date, ok := value.TargetDate(); ok {
			appendJSONString(&json, date.String())
		} else {
			json.WriteString("null")
		}
	}
	json.WriteByte('}')
	return payloadPrefix + base64.RawURLEncoding.EncodeToString([]byte(json.String())) + payloadSuffix
}

func appendJSONString(builder *strings.Builder, value string) {
	builder.WriteByte('"')
	for _, r := range value {
		switch r {
		case '"':
			builder.WriteString(`\"`)
		case '\\':
			builder.WriteString(`\\`)
		case '\b':
			builder.WriteString(`\b`)
		case '\f':
			builder.WriteString(`\f`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if r < 0x20 {
				const hex = "0123456789abcdef"
				builder.WriteString(`\u00`)
				builder.WriteByte(hex[(r>>4)&0xf])
				builder.WriteByte(hex[r&0xf])
				continue
			}
			builder.WriteRune(r)
		}
	}
	builder.WriteByte('"')
}
