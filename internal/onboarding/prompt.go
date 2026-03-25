package onboarding

import (
	"fmt"
	"strings"
)

// BuildConsultantPrompt constructs the system prompt for the onboarding dialogue LLM.
func BuildConsultantPrompt(collected *CollectedData, messageCount int) string {
	var sb strings.Builder

	sb.WriteString(`You are an experienced management consultant conducting an initial assessment for a new client.
Your goal is to understand their company deeply so you can design a complete management system.

RULES:
- Ask ONE question at a time
- Be conversational, not interrogative — follow up on interesting points
- Respond in the boss's language (auto-detect from their messages)
- Do NOT list all questions upfront
`)

	// Inject collected info
	sb.WriteString("\n## Already Collected\n")
	hasAny := false
	if collected.Industry != "" {
		fmt.Fprintf(&sb, "- Industry: %s\n", collected.Industry)
		hasAny = true
	}
	if collected.CompanyStage != "" {
		fmt.Fprintf(&sb, "- Company stage: %s\n", collected.CompanyStage)
		hasAny = true
	}
	if collected.BusinessModel != "" {
		fmt.Fprintf(&sb, "- Business model: %s\n", collected.BusinessModel)
		hasAny = true
	}
	if collected.TeamSize > 0 {
		fmt.Fprintf(&sb, "- Team size: %d\n", collected.TeamSize)
		hasAny = true
	}
	if collected.OrgStructure != "" {
		fmt.Fprintf(&sb, "- Org structure: %s\n", collected.OrgStructure)
		hasAny = true
	}
	if collected.CurrentProjects != "" {
		fmt.Fprintf(&sb, "- Current projects: %s\n", collected.CurrentProjects)
		hasAny = true
	}
	if len(collected.PainPoints) > 0 {
		fmt.Fprintf(&sb, "- Pain points: %s\n", strings.Join(collected.PainPoints, ", "))
		hasAny = true
	}
	if len(collected.CommTools) > 0 {
		fmt.Fprintf(&sb, "- Comm tools: %s\n", strings.Join(collected.CommTools, ", "))
		hasAny = true
	}
	if collected.CulturePrefs != "" {
		fmt.Fprintf(&sb, "- Culture prefs: %s\n", collected.CulturePrefs)
		hasAny = true
	}
	if collected.GoalFramework != "" {
		fmt.Fprintf(&sb, "- Goal framework: %s\n", collected.GoalFramework)
		hasAny = true
	}
	if !hasAny {
		sb.WriteString("- (Nothing collected yet)\n")
	}

	// Inject missing required fields
	sb.WriteString("\n## Still Need (Required)\n")
	missing := missingRequired(collected)
	if len(missing) == 0 {
		sb.WriteString("ALL REQUIRED INFO COLLECTED. Wrap up now.\n")
	} else {
		for _, m := range missing {
			fmt.Fprintf(&sb, "- %s\n", m)
		}
	}

	// Turn awareness
	if messageCount >= 15 {
		sb.WriteString("\nYou are at turn " + fmt.Sprint(messageCount) + "/20. Start wrapping up — summarize what you know and ask about remaining gaps directly.\n")
	}
	if messageCount >= 20 {
		sb.WriteString("\nTURN LIMIT REACHED. Summarize all collected info and tell the boss you'll proceed with what you have. Ask them to fill in any critical gaps.\n")
	}

	return sb.String()
}

// BuildExtractionPrompt constructs the prompt for lightweight info extraction.
func BuildExtractionPrompt(currentData *CollectedData, userMessage string) string {
	return fmt.Sprintf(`Extract structured information from this message. Return ONLY valid JSON matching the CollectedData schema. Only include fields that are NEW or UPDATED — omit unchanged fields. If no new info, return {}.

Current data: %s

User message: %s`, toJSON(currentData), userMessage)
}

func missingRequired(c *CollectedData) []string {
	var missing []string
	if c.Industry == "" {
		missing = append(missing, "Industry")
	}
	if c.CompanyStage == "" {
		missing = append(missing, "Company stage")
	}
	if c.BusinessModel == "" {
		missing = append(missing, "Business model")
	}
	if c.TeamSize == 0 {
		missing = append(missing, "Team size")
	}
	if c.OrgStructure == "" {
		missing = append(missing, "Organizational structure")
	}
	if c.CurrentProjects == "" {
		missing = append(missing, "Current projects")
	}
	if len(c.PainPoints) == 0 {
		missing = append(missing, "Management pain points")
	}
	if len(c.CommTools) == 0 {
		missing = append(missing, "Communication tools")
	}
	return missing
}
