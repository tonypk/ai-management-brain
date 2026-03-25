package onboarding

import "testing"

func TestCleanJSON(t *testing.T) {
	tests := []struct{ in, want string }{
		{`{"a":1}`, `{"a":1}`},
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"```\n{\"a\":1}\n```", `{"a":1}`},
		{"  {\"a\":1}  ", `{"a":1}`},
	}
	for _, tc := range tests {
		if got := cleanJSON(tc.in); got != tc.want {
			t.Errorf("cleanJSON(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestToJSON(t *testing.T) {
	got := toJSON(map[string]int{"x": 1})
	if got != `{"x":1}` {
		t.Errorf("toJSON = %q", got)
	}
}
