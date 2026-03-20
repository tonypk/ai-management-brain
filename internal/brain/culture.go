package brain

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CommunicationStyle describes how a culture communicates in the workplace.
type CommunicationStyle struct {
	Directness        string `yaml:"directness"`
	HierarchyRespect  string `yaml:"hierarchy_respect"`
	RelationshipFirst bool   `yaml:"relationship_first"`
	GroupFace         string `yaml:"group_face"`
}

// ChaseRules describes culturally-sensitive rules for chasing team members.
type ChaseRules struct {
	NeverNameInGroup      bool `yaml:"never_name_in_group"`
	PrivateBeforeEscalate bool `yaml:"private_before_escalate"`
	WarmthRequired        bool `yaml:"warmth_required"`
	AcknowledgeEffort     bool `yaml:"acknowledge_effort"`
}

// CulturePack is the top-level structure parsed from a culture YAML file.
type CulturePack struct {
	Market             string             `yaml:"market"`
	Language           string             `yaml:"language"`
	Timezone           string             `yaml:"timezone"`
	Version            int                `yaml:"version"`
	CommunicationStyle CommunicationStyle `yaml:"communication_style"`
	ChaseRules         ChaseRules         `yaml:"chase_rules"`
	ForbiddenPatterns  []string           `yaml:"forbidden_patterns"`
	PreferredPatterns  []string           `yaml:"preferred_patterns"`
}

// defaultCulturePack returns a neutral culture pack used when no specific
// culture file is requested or found.
func defaultCulturePack() *CulturePack {
	return &CulturePack{
		Market:   "Default",
		Language: "English",
		Timezone: "UTC",
		Version:  1,
		CommunicationStyle: CommunicationStyle{
			Directness:        "medium",
			HierarchyRespect:  "medium",
			RelationshipFirst: false,
			GroupFace:         "medium",
		},
		ChaseRules: ChaseRules{
			NeverNameInGroup:      false,
			PrivateBeforeEscalate: false,
			WarmthRequired:        false,
			AcknowledgeEffort:     false,
		},
	}
}

// LoadCulture reads and parses the YAML config for the given culture code.
// It searches for configs/cultures/{code}.yaml starting from the working
// directory and walking up the directory tree until found.
// For "default" code or when the file is not found, a default CulturePack
// with medium directness is returned.
func LoadCulture(code string) (*CulturePack, error) {
	if code == "default" {
		return defaultCulturePack(), nil
	}

	path, err := findCultureFile(code)
	if err != nil {
		// Unknown code with no file: return default rather than error.
		return defaultCulturePack(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read culture file %q: %w", path, err)
	}

	var pack CulturePack
	if err := yaml.Unmarshal(data, &pack); err != nil {
		return nil, fmt.Errorf("parse culture file %q: %w", path, err)
	}

	return &pack, nil
}

// ShouldOverride returns true if the culture's chase rules require overriding
// the given action. Currently, actions "public_reminder" and "public_naming"
// are overridden when NeverNameInGroup is true.
func (c *CulturePack) ShouldOverride(action string) bool {
	if !c.ChaseRules.NeverNameInGroup {
		return false
	}
	return action == "public_reminder" || action == "public_naming"
}

// GetForbiddenPatterns returns the list of phrases forbidden in this culture.
func (c *CulturePack) GetForbiddenPatterns() []string {
	return c.ForbiddenPatterns
}

// GetPreferredPatterns returns the list of preferred phrases for this culture.
func (c *CulturePack) GetPreferredPatterns() []string {
	return c.PreferredPatterns
}

// findCultureFile locates configs/cultures/{code}.yaml by searching from the
// current working directory upward.
func findCultureFile(code string) (string, error) {
	rel := filepath.Join("configs", "cultures", code+".yaml")

	// Start from cwd and walk up.
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, rel)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding the file.
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("configs/cultures/%s.yaml not found (searched from %s)", code, cwd)
}
