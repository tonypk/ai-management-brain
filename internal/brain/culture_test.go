package brain_test

import (
	"testing"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

func TestLoadCulture_Philippines(t *testing.T) {
	c, err := brain.LoadCulture("philippines")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.CommunicationStyle.Directness != "low" {
		t.Errorf("directness = %q", c.CommunicationStyle.Directness)
	}
	if !c.ChaseRules.NeverNameInGroup {
		t.Error("PH should never name in group")
	}
	if len(c.ForbiddenPatterns) == 0 {
		t.Error("expected forbidden patterns")
	}
}

func TestLoadCulture_Singapore(t *testing.T) {
	c, err := brain.LoadCulture("singapore")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if c.CommunicationStyle.Directness != "high" {
		t.Errorf("directness = %q, want high", c.CommunicationStyle.Directness)
	}
}

func TestLoadCulture_Default(t *testing.T) {
	c, err := brain.LoadCulture("default")
	if err != nil {
		t.Fatalf("default culture should not error: %v", err)
	}
	if c.CommunicationStyle.Directness != "medium" {
		t.Errorf("default directness should be medium")
	}
}

func TestShouldOverride_PHPublicChase(t *testing.T) {
	c, _ := brain.LoadCulture("philippines")
	// PH culture should override public chase to private
	if !c.ShouldOverride("public_reminder") {
		t.Error("PH should override public_reminder to private")
	}
}

func TestShouldOverride_SGNoOverride(t *testing.T) {
	c, _ := brain.LoadCulture("singapore")
	// SG culture should NOT override public chase
	if c.ShouldOverride("public_reminder") {
		t.Error("SG should not override public_reminder")
	}
}
