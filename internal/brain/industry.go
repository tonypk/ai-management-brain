package brain

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// IndustryTemplate holds industry-specific reference data for LLM context injection.
type IndustryTemplate struct {
	ID               string     `yaml:"id"`
	Name             string     `yaml:"name"`
	NameEn           string     `yaml:"name_en"`
	Version          int        `yaml:"version"`
	Keywords         []string   `yaml:"keywords"`
	KPIReference     []KPIRef   `yaml:"kpi_reference"`
	RoleReference    []RoleRef  `yaml:"role_reference"`
	QuestionRef      []string   `yaml:"question_reference"`
	AlertReference   []AlertRef `yaml:"alert_reference"`
	CommonPainPoints []string   `yaml:"common_pain_points"`
}

// KPIRef is a reference KPI for an industry.
type KPIRef struct {
	Name        string `yaml:"name"`
	Frequency   string `yaml:"frequency"`
	Description string `yaml:"description"`
}

// RoleRef is a reference role for an industry.
type RoleRef struct {
	Title string `yaml:"title"`
	Type  string `yaml:"type"` // human, ai, human_or_ai
}

// AlertRef is a reference alert pattern for an industry.
type AlertRef struct {
	Condition string `yaml:"condition"`
	Action    string `yaml:"action"`
}

var (
	industries   map[string]*IndustryTemplate
	industryOnce sync.Once
)

// LoadIndustries loads all industry YAML configs from configs/industries/.
func LoadIndustries() error {
	var loadErr error
	industryOnce.Do(func() {
		industries = make(map[string]*IndustryTemplate)

		dir, err := findConfigDir("industries")
		if err != nil {
			slog.Warn("industry configs not found, skipping", "error", err)
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			loadErr = fmt.Errorf("read industry dir: %w", err)
			return
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				slog.Warn("skip industry file", "file", entry.Name(), "error", err)
				continue
			}

			var tmpl IndustryTemplate
			if err := yaml.Unmarshal(data, &tmpl); err != nil {
				slog.Warn("skip industry file", "file", entry.Name(), "error", err)
				continue
			}

			if tmpl.ID == "" {
				slog.Warn("skip industry file without id", "file", entry.Name())
				continue
			}

			industries[tmpl.ID] = &tmpl
		}

		slog.Info("loaded industry templates", "count", len(industries))
	})
	return loadErr
}

// MatchIndustry finds the best matching industry template for the given text.
// Returns nil if no match found.
func MatchIndustry(text string) *IndustryTemplate {
	if text == "" {
		return nil
	}
	lower := strings.ToLower(text)

	var best *IndustryTemplate
	bestScore := 0

	for _, tmpl := range industries {
		score := 0
		for _, kw := range tmpl.Keywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(lower, kwLower) {
				// Longer keyword matches are more specific
				score += len(kw)
			}
		}
		if score > bestScore {
			bestScore = score
			best = tmpl
		}
	}

	if best == nil {
		slog.Debug("no industry template matched", "text", text)
	}

	return best
}

// GetIndustry returns an industry template by ID, or nil if not found.
func GetIndustry(id string) *IndustryTemplate {
	return industries[id]
}

// ListIndustries returns all loaded industry template IDs.
func ListIndustries() []string {
	ids := make([]string, 0, len(industries))
	for id := range industries {
		ids = append(ids, id)
	}
	return ids
}

// BuildIndustryContext formats an IndustryTemplate as LLM context text.
func BuildIndustryContext(tmpl *IndustryTemplate) string {
	if tmpl == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n\n--- Industry Context: %s (%s) ---\n", tmpl.Name, tmpl.NameEn))

	if len(tmpl.KPIReference) > 0 {
		b.WriteString("\n参考 KPI:\n")
		for _, kpi := range tmpl.KPIReference {
			b.WriteString(fmt.Sprintf("- %s (%s): %s\n", kpi.Name, kpi.Frequency, kpi.Description))
		}
	}

	if len(tmpl.RoleReference) > 0 {
		b.WriteString("\n典型岗位:\n")
		for _, role := range tmpl.RoleReference {
			b.WriteString(fmt.Sprintf("- %s (type: %s)\n", role.Title, role.Type))
		}
	}

	if len(tmpl.QuestionRef) > 0 {
		b.WriteString("\n建议的 check-in 问题:\n")
		for _, q := range tmpl.QuestionRef {
			b.WriteString(fmt.Sprintf("- %s\n", q))
		}
	}

	if len(tmpl.AlertReference) > 0 {
		b.WriteString("\n常见预警场景:\n")
		for _, a := range tmpl.AlertReference {
			b.WriteString(fmt.Sprintf("- 条件: %s → 动作: %s\n", a.Condition, a.Action))
		}
	}

	b.WriteString("\n注意: 以上仅供参考，请根据公司实际情况调整和取舍。不要照搬，要结合你的管理哲学来选择。\n")

	return b.String()
}

// BuildPainPointsContext formats pain points as wizard context.
func BuildPainPointsContext(tmpl *IndustryTemplate) string {
	if tmpl == nil || len(tmpl.CommonPainPoints) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n\n--- %s 行业常见痛点 ---\n", tmpl.Name))
	for _, pp := range tmpl.CommonPainPoints {
		b.WriteString(fmt.Sprintf("- %s\n", pp))
	}
	b.WriteString("你可以根据这些痛点主动询问，帮助快速了解公司情况。\n")

	return b.String()
}

// findConfigDir locates the configs/<subdir> directory by walking up from cwd.
func findConfigDir(subdir string) (string, error) {
	rel := filepath.Join("configs", subdir)

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	dir := cwd
	for {
		candidate := filepath.Join(dir, rel)
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("directory %q not found from %q", rel, cwd)
}
