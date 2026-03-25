package onboarding

import (
	"encoding/json"
	"strings"
)

// toJSON marshals a value to a JSON string, returning "{}" on error.
func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

// cleanJSON strips markdown code fences from LLM output.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
