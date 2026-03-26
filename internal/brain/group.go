package brain

import (
	"fmt"
	"strings"
)

// GroupTeamData holds team statistics for the group decision prompt.
type GroupTeamData struct {
	SubmissionRate string
	SentimentDist  string
	LatestSummary  string
	Weekday        string
}

// BuildGroupReplyPrompt builds the system prompt for @mention replies in group chat.
func BuildGroupReplyPrompt(mentorName, groupType, teamSummary, userQuestion string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "You are %s, a management mentor active in a team group chat.\n\n", mentorName)

	sb.WriteString("<group_context>\n")
	fmt.Fprintf(&sb, "Group type: %s\n", groupType)
	if teamSummary != "" {
		sb.WriteString("Latest team summary:\n")
		sb.WriteString(teamSummary)
		sb.WriteString("\n")
	}
	sb.WriteString("</group_context>\n\n")

	sb.WriteString("Rules:\n")
	sb.WriteString("- NEVER mention individual employee's private reports, sentiments, or personal memories\n")
	sb.WriteString("- Keep responses concise and relevant to the group context\n")
	sb.WriteString("- Maintain your mentor persona and philosophy\n")
	sb.WriteString("- Answer based on team-level data, not individual data\n")

	fmt.Fprintf(&sb, "\nA team member asks: %q\n", userQuestion)
	sb.WriteString("Respond helpfully as the team's management mentor.")

	return sb.String()
}

// BuildGroupDecisionPrompt builds the system prompt for the daily autonomous posting decision.
func BuildGroupDecisionPrompt(mentorName, groupType string, data GroupTeamData) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "You are %s, managing a %s team group chat.\n\n", mentorName, groupType)

	sb.WriteString("Team data:\n")
	fmt.Fprintf(&sb, "- Submission rate: %s\n", data.SubmissionRate)
	fmt.Fprintf(&sb, "- Sentiment distribution: %s\n", data.SentimentDist)
	if data.LatestSummary != "" {
		fmt.Fprintf(&sb, "- Latest summary: %s\n", data.LatestSummary)
	}
	fmt.Fprintf(&sb, "- Today is: %s\n", data.Weekday)

	sb.WriteString("\nDecide whether to post a message in the group chat.\n")
	sb.WriteString("If not needed, reply with only: SKIP\n")
	sb.WriteString("If needed, output the message content directly (no markers or labels).\n\n")

	sb.WriteString("Rules:\n")
	sb.WriteString("- Don't post every day — about 2-3 times per week is ideal\n")
	sb.WriteString("- Friday is good for a weekly review\n")
	sb.WriteString("- If submission rate is below 60%, encourage the team\n")
	sb.WriteString("- Maintain your mentor style and cultural context\n")
	sb.WriteString("- NEVER mention individual private information\n")
	sb.WriteString("- Keep messages concise (3-5 sentences)\n")
	sb.WriteString("- Use the language appropriate for the team's culture\n")

	return sb.String()
}

// IsSkipDecision checks if the AI decision response indicates no posting.
func IsSkipDecision(response string) bool {
	trimmed := strings.TrimSpace(response)
	if trimmed == "" {
		return true
	}
	return strings.EqualFold(trimmed, "SKIP")
}
