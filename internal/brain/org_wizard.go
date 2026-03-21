package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// WizardMessage represents one message in the wizard conversation.
type WizardMessage struct {
	Role    string `json:"role"`    // "mentor" or "user"
	Content string `json:"content"`
}

// WizardResponse is the return value from wizard Start/ProcessAnswer.
type WizardResponse struct {
	MentorMessage string          `json:"mentor_message"`
	IsComplete    bool            `json:"is_complete"`
	Profile       *CompanyProfile `json:"profile,omitempty"`
	Plan          *ManagementPlan `json:"plan,omitempty"`
}

// wizardLLMResponse is the internal JSON structure Claude returns during the wizard.
type wizardLLMResponse struct {
	Status    string          `json:"status"`    // "continue" or "ready"
	Question  string          `json:"question"`  // next question (when status=continue)
	Message   string          `json:"message"`   // display message (when status=continue)
	Extracted json.RawMessage `json:"extracted"` // partial profile data
	Profile   *CompanyProfile `json:"profile"`   // complete profile (when status=ready)
}

// OrgWizard manages multi-turn conversations to collect company information.
type OrgWizard struct {
	llm       *AnthropicClient
	orgEngine *OrgEngine
}

// NewOrgWizard creates a new organization wizard.
func NewOrgWizard(llm *AnthropicClient) *OrgWizard {
	return &OrgWizard{
		llm:       llm,
		orgEngine: NewOrgEngine(llm),
	}
}

// Start initiates a new wizard conversation with the mentor's first question.
func (w *OrgWizard) Start(ctx context.Context, mentor *MentorConfig) (*WizardResponse, error) {
	if mentor == nil {
		return nil, fmt.Errorf("mentor config is required")
	}

	systemPrompt := buildWizardSystemPrompt(mentor)
	userPrompt := "请开始第一个问题。"

	resp, err := w.llm.Chat(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM wizard start: %w", err)
	}

	parsed, err := parseWizardResponse(resp)
	if err != nil {
		// Fallback: treat entire response as the mentor's question
		return &WizardResponse{
			MentorMessage: resp,
			IsComplete:    false,
		}, nil
	}

	msg := parsed.Question
	if msg == "" {
		msg = parsed.Message
	}
	if msg == "" {
		msg = resp
	}

	return &WizardResponse{
		MentorMessage: msg,
		IsComplete:    false,
	}, nil
}

// ProcessAnswer processes a user's answer and returns the next question or the final plan.
func (w *OrgWizard) ProcessAnswer(ctx context.Context, mentor *MentorConfig, history []WizardMessage, answer string) (*WizardResponse, error) {
	if mentor == nil {
		return nil, fmt.Errorf("mentor config is required")
	}

	// Build conversation messages for Claude
	systemPrompt := buildWizardSystemPrompt(mentor)

	var sb strings.Builder
	for _, msg := range history {
		if msg.Role == "mentor" {
			sb.WriteString(fmt.Sprintf("导师: %s\n\n", msg.Content))
		} else {
			sb.WriteString(fmt.Sprintf("用户: %s\n\n", msg.Content))
		}
	}
	sb.WriteString(fmt.Sprintf("用户: %s\n\n", answer))
	sb.WriteString("请分析对话，决定是否需要更多信息。用 JSON 格式回复。")

	resp, err := w.llm.Chat(ctx, systemPrompt, sb.String())
	if err != nil {
		return nil, fmt.Errorf("LLM wizard answer: %w", err)
	}

	parsed, err := parseWizardResponse(resp)
	if err != nil {
		// Fallback: treat as continuing question
		return &WizardResponse{
			MentorMessage: resp,
			IsComplete:    false,
		}, nil
	}

	// If the mentor has enough info, generate the plan
	if parsed.Status == "ready" && parsed.Profile != nil {
		plan, err := w.orgEngine.GeneratePlan(ctx, mentor, *parsed.Profile)
		if err != nil {
			return nil, fmt.Errorf("generate plan: %w", err)
		}

		return &WizardResponse{
			MentorMessage: "信息收集完成，我已经为你设计了管理方案。",
			IsComplete:    true,
			Profile:       parsed.Profile,
			Plan:          plan,
		}, nil
	}

	// Still collecting info
	msg := parsed.Question
	if msg == "" {
		msg = parsed.Message
	}
	if msg == "" {
		msg = resp
	}

	return &WizardResponse{
		MentorMessage: msg,
		IsComplete:    false,
	}, nil
}

// buildWizardSystemPrompt creates the system prompt for the wizard conversation.
func buildWizardSystemPrompt(mentor *MentorConfig) string {
	return fmt.Sprintf(`你是 %s（%s），管理哲学：%s

%s

你正在与一位新客户对话，了解他们的公司，目标是收集足够的信息来设计管理体系。

你需要了解：
1. 行业和业务模式
2. 团队规模
3. 公司阶段（初创/成长/成熟）
4. 痛点和挑战
5. 地区/预算等约束条件

以你的风格自然对话。可以合并问题。不要一次问太多。

当信息足够时，回复：
{"status": "ready", "profile": {"industry": "...", "size": 数字, "stage": "...", "business_model": "...", "region": "...", "pain_points": ["..."]}}

当还需要更多信息时，回复：
{"status": "continue", "question": "你的下一个问题"}`,
		mentor.NameEn, mentor.Company, mentor.Philosophy, mentor.Strategy.SystemPrompt)
}

// parseWizardResponse extracts the wizard LLM response from a raw string.
func parseWizardResponse(resp string) (*wizardLLMResponse, error) {
	resp = strings.TrimSpace(resp)

	// Strip markdown code fences
	if strings.HasPrefix(resp, "```") {
		lines := strings.Split(resp, "\n")
		if len(lines) > 2 {
			resp = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	// Find JSON boundaries
	start := strings.Index(resp, "{")
	end := strings.LastIndex(resp, "}")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("no JSON object found in wizard response")
	}
	resp = resp[start : end+1]

	var result wizardLLMResponse
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("unmarshal wizard response: %w", err)
	}

	return &result, nil
}
