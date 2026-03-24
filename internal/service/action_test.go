package service

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestFuzzyNameMatch(t *testing.T) {
	tests := []struct {
		fullName string
		query    string
		want     bool
	}{
		{"John Santos", "john", true},
		{"John Santos", "John", true},
		{"John Santos", "santos", true},
		{"John Santos", "john santos", true},
		{"John Santos", "JOHN", true},
		{"John Santos", "Jane", false},
		{"John Santos", "", true}, // empty query matches everything
		{"John Santos", "  john  ", true},
		{"Alice Wong", "alice", true},
		{"Alice Wong", "wong", true},
		{"Alice Wong", "bob", false},
	}

	for _, tt := range tests {
		got := fuzzyNameMatch(tt.fullName, tt.query)
		if got != tt.want {
			t.Errorf("fuzzyNameMatch(%q, %q) = %v, want %v", tt.fullName, tt.query, got, tt.want)
		}
	}
}

func TestFormatUUID_Invalid(t *testing.T) {
	u := pgtype.UUID{Valid: false}
	result := formatUUID(u)
	if result != "" {
		t.Errorf("formatUUID(invalid) = %q, want empty", result)
	}
}

func TestFormatUUID_Valid(t *testing.T) {
	u := pgtype.UUID{
		Bytes: [16]byte{0x55, 0x0e, 0x84, 0x00, 0xe2, 0x9b, 0x41, 0xd4, 0xa7, 0x16, 0x44, 0x66, 0x55, 0x44, 0x00, 0x00},
		Valid: true,
	}
	result := formatUUID(u)
	expected := "550e8400-e29b-41d4-a716-446655440000"
	if result != expected {
		t.Errorf("formatUUID = %q, want %q", result, expected)
	}
}
