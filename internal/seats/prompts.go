package seats

import (
	"fmt"
	"strings"

	"github.com/tonypk/ai-management-brain/internal/brain"
)

// BuildSeatChatPrompt builds the system prompt for a seat chat session.
func BuildSeatChatPrompt(engine *brain.Engine, title, scope, memoryContext string) string {
	var sb strings.Builder

	// Base mentor system prompt
	sb.WriteString(engine.BuildSystemPrompt())

	sb.WriteString(fmt.Sprintf("\n\n## Your Role: %s\n", title))
	sb.WriteString(fmt.Sprintf("Your scope of responsibility: %s\n", scope))
	sb.WriteString("\nFocus your responses on topics within your scope. When asked about topics outside your area, acknowledge the boundary and suggest which role might be better suited.\n")

	if memoryContext != "" {
		sb.WriteString("\n## Company Knowledge\n")
		sb.WriteString(memoryContext)
		sb.WriteString("\n")
	}

	return sb.String()
}

// BuildBoardPrompt builds the system prompt for a seat's contribution to a board discussion.
func BuildBoardPrompt(engine *brain.Engine, title, scope, topic, memoryContext, priorResponses string) string {
	var sb strings.Builder

	sb.WriteString(engine.BuildSystemPrompt())

	sb.WriteString(fmt.Sprintf("\n\n## Board Discussion — Your Role: %s\n", title))
	sb.WriteString(fmt.Sprintf("Your scope: %s\n", scope))
	sb.WriteString("\nYou are participating in a board discussion. Analyze the topic from your professional perspective.\n")
	sb.WriteString("You may agree or disagree with previous speakers. Be concise (2-3 paragraphs max).\n")

	if memoryContext != "" {
		sb.WriteString("\n## Company Knowledge\n")
		sb.WriteString(memoryContext)
		sb.WriteString("\n")
	}

	if priorResponses != "" {
		sb.WriteString("\n## Previous Speakers\n")
		sb.WriteString(priorResponses)
	}

	return sb.String()
}

// BuildSynthesisPrompt builds the prompt for synthesizing all board responses.
func BuildSynthesisPrompt(topic string, responses []BoardResponse) string {
	var sb strings.Builder

	sb.WriteString("You are the board secretary. Your job is to synthesize all executives' opinions into a balanced, actionable decision recommendation.\n\n")
	sb.WriteString(fmt.Sprintf("## Discussion Topic\n%s\n\n", topic))
	sb.WriteString("## Executive Opinions\n\n")

	for _, r := range responses {
		if r.Content == "[unavailable — persona load failed]" || r.Content == "[unavailable — AI response failed]" {
			continue
		}
		sb.WriteString(fmt.Sprintf("### %s (%s)\n%s\n\n", r.Title, r.SeatType, r.Content))
	}

	sb.WriteString("## Your Task\n")
	sb.WriteString("1. Identify points of agreement\n")
	sb.WriteString("2. Highlight key disagreements and trade-offs\n")
	sb.WriteString("3. Provide a balanced recommendation with clear next steps\n")
	sb.WriteString("4. Keep it concise (3-4 paragraphs max)\n")

	return sb.String()
}
