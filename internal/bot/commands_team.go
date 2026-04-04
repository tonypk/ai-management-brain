package bot

import (
	"context"
	"fmt"
	"strings"
)

// HandleStatus shows the boss a summary of the team and their activity.
func (h *CommandHandler) HandleStatus(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	employees, err := h.db.ListEmployeesByTenant(context.Background(), tenant.ID)
	if err != nil {
		return fmt.Errorf("list employees: %w", err)
	}

	if len(employees) == 0 {
		return c.Send("No employees yet. Use /addemployee <name> <culture> to add.")
	}

	var sb strings.Builder
	sb.WriteString("Team Status:\n\n")
	for _, emp := range employees {
		status := "not linked"
		if emp.TelegramID > 0 {
			status = "active"
		}
		sb.WriteString(fmt.Sprintf("- %s (%s)\n", emp.Name, status))
	}
	return c.Send(sb.String())
}

// HandleAddEmployee adds a new employee to the boss's team and generates an invite code.
func (h *CommandHandler) HandleAddEmployee(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	raw := strings.TrimSpace(c.Text())
	idx := strings.Index(raw, " ")
	if idx < 0 {
		return c.Send("Usage: /addemployee name | job_title | responsibilities | country | language\nOnly name is required.\nExample: /addemployee Alice | Frontend Developer | Handles UI/UX | Philippines | Chinese")
	}
	body := strings.TrimSpace(raw[idx+1:])

	parts := strings.SplitN(body, "|", 5)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	name := parts[0]
	if name == "" {
		return c.Send("Name is required.\nUsage: /addemployee name | job_title | responsibilities | country | language")
	}

	var jobTitle, responsibilities, country, language string
	if len(parts) > 1 {
		jobTitle = parts[1]
	}
	if len(parts) > 2 {
		responsibilities = parts[2]
	}
	if len(parts) > 3 {
		country = parts[3]
	}
	if len(parts) > 4 {
		language = parts[4]
	}

	inviteCode := generateInviteCode()

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	emp, err := h.db.CreateEmployee(context.Background(), CreateEmployeeParams{
		TenantID:         tenant.ID,
		Name:             name,
		CultureCode:      "default",
		InviteCode:       inviteCode,
		JobTitle:         jobTitle,
		Responsibilities: responsibilities,
		Country:          country,
		Language:         language,
	})
	if err != nil {
		return fmt.Errorf("create employee: %w", err)
	}

	msg := fmt.Sprintf("Employee added!\nName: %s", emp.Name)
	if jobTitle != "" {
		msg += fmt.Sprintf("\nJob: %s", jobTitle)
	}
	if country != "" {
		msg += fmt.Sprintf("\nCountry: %s", country)
	}
	if language != "" {
		msg += fmt.Sprintf("\nLanguage: %s", language)
	}
	msg += fmt.Sprintf("\nInvite code: %s\n\nShare this code with %s to join via /join %s", inviteCode, name, inviteCode)

	return c.Send(msg)
}

// HandleProfile shows an employee's submission profile to the boss.
func (h *CommandHandler) HandleProfile(c BotContext) error {
	if c.SenderID() != h.bossChatID {
		return c.Send("Permission denied")
	}

	parts := strings.Fields(c.Text())
	if len(parts) < 2 {
		return c.Send("Usage: /profile <employee_name>\nExample: /profile Alice")
	}

	empName := parts[1]

	tenant, err := h.db.GetTenantByBossChatID(context.Background(), c.SenderID())
	if err != nil {
		return c.Send("No team found. Use /start first.")
	}

	employees, err := h.db.ListEmployeesByTenant(context.Background(), tenant.ID)
	if err != nil {
		return fmt.Errorf("list employees: %w", err)
	}

	var found *Employee
	for i, emp := range employees {
		if strings.EqualFold(emp.Name, empName) {
			found = &employees[i]
			break
		}
	}
	if found == nil {
		return c.Send(fmt.Sprintf("Employee '%s' not found.", empName))
	}

	profile, err := h.db.GetEmployeeProfile(context.Background(), found.ID)
	if err != nil {
		return c.Send(fmt.Sprintf("Could not load profile for %s.", found.Name))
	}

	status := "not linked"
	if found.TelegramID > 0 {
		status = "active"
	}

	msg := fmt.Sprintf("Employee Profile: %s\n\nStatus: %s", found.Name, status)
	if found.JobTitle != "" {
		msg += fmt.Sprintf("\nJob Title: %s", found.JobTitle)
	}
	if found.Responsibilities != "" {
		msg += fmt.Sprintf("\nResponsibilities: %s", found.Responsibilities)
	}
	if found.Country != "" {
		msg += fmt.Sprintf("\nCountry: %s", found.Country)
	}
	if found.Language != "" {
		msg += fmt.Sprintf("\nLanguage: %s", found.Language)
	}
	msg += fmt.Sprintf("\nCulture: %s", found.CultureCode)
	msg += fmt.Sprintf("\n\nLast 7 days: %d/7 submitted\nLast 30 days: %d submitted\nCurrent streak: %d days\nSentiment trend: %s",
		profile.SubmittedLast7, profile.SubmittedLast30,
		profile.CurrentStreak, profile.SentimentTrend)
	return c.Send(msg)
}
