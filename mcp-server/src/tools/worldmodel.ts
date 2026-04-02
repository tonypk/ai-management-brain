import { CallToolResult } from "@modelcontextprotocol/sdk/types.js";
import { ApiClient, APIError } from "../api-client.js";

// World Model types
interface WorldModelOverview {
  skill_count: number;
  relationship_count: number;
  active_blocker_count: number;
  growth_events_month: number;
  top_categories: Array<{
    category: string;
    count: number;
  }>;
}

interface WorldModelSkill {
  employee_name: string;
  skill_name: string;
  proficiency: string;
  confidence: string;
  mention_count: number;
}

interface WorldModelBlocker {
  employee_name: string;
  category: string;
  description: string;
  status: string;
  recurrence_count: number;
  first_seen_at: string;
}

interface WorldModelInsight {
  dimension: string;
  insight_text: string;
  confidence: string;
  generated_at: string;
}

interface EmployeeWorldModel {
  skills: Array<{
    skill_name: string;
    proficiency: string;
    confidence: string;
    mention_count: number;
  }>;
  growth_events: Array<{
    event_type: string;
    description: string;
    occurred_at: string;
  }>;
  blockers: Array<{
    category: string;
    description: string;
    status: string;
    recurrence_count: number;
    first_seen_at: string;
  }>;
}

interface EmployeeProfileResponse {
  id: string;
  name: string;
  role: string;
}

export async function getWorldModel(
  client: ApiClient,
): Promise<CallToolResult> {
  try {
    const [overview, skills, blockers, insights] = await Promise.all([
      client.get<WorldModelOverview>("/api/v1/world-model/overview"),
      client.get<WorldModelSkill[]>("/api/v1/world-model/skills"),
      client.get<WorldModelBlocker[]>("/api/v1/world-model/blockers"),
      client.get<WorldModelInsight[]>("/api/v1/world-model/insights"),
    ]);

    // Format as human-readable markdown
    let text = "# Team World Model\n\n";

    // Overview section
    text += "## Overview\n";
    text += `- **Skills Tracked**: ${overview.skill_count}\n`;
    text += `- **Relationships**: ${overview.relationship_count}\n`;
    text += `- **Active Blockers**: ${overview.active_blocker_count}\n`;
    text += `- **Growth Events (This Month)**: ${overview.growth_events_month}\n\n`;

    if (overview.top_categories.length > 0) {
      text += "### Top Skill Categories\n";
      overview.top_categories.forEach((cat) => {
        text += `- ${cat.category}: ${cat.count} skills\n`;
      });
      text += "\n";
    }

    // Insights section
    if (insights.length > 0) {
      text += "## AI Insights\n";
      insights.forEach((insight) => {
        const confidencePct = (parseFloat(insight.confidence) * 100).toFixed(0);
        text += `\n**${insight.dimension}** (${confidencePct}% confidence)\n`;
        text += `${insight.insight_text}\n`;
      });
      text += "\n";
    }

    // Top Skills section (limit to first 20)
    if (skills.length > 0) {
      text += "## Top Skills\n";
      const displaySkills = skills.slice(0, 20);
      displaySkills.forEach((skill) => {
        text += `- **${skill.employee_name}**: ${skill.skill_name} (${skill.proficiency}, ${skill.mention_count} mentions)\n`;
      });
      if (skills.length > 20) {
        text += `\n_...and ${skills.length - 20} more skills_\n`;
      }
      text += "\n";
    }

    // Active Blockers section
    if (blockers.length > 0) {
      text += "## Active Blockers\n";
      blockers.forEach((blocker) => {
        text += `\n**${blocker.employee_name}** - ${blocker.category}\n`;
        text += `${blocker.description}\n`;
        text += `- Status: ${blocker.status}\n`;
        text += `- Recurrence: ${blocker.recurrence_count}x\n`;
        text += `- First seen: ${blocker.first_seen_at}\n`;
      });
    }

    return {
      content: [{ type: "text", text }],
    };
  } catch (error) {
    return errorResult(error);
  }
}

export async function getEmployeeWorldModel(
  client: ApiClient,
  name: string,
): Promise<CallToolResult> {
  if (!name.trim()) {
    return {
      content: [{ type: "text", text: "Employee name cannot be empty." }],
      isError: true,
    };
  }

  try {
    // First, resolve employee by name
    const profile = await client.get<EmployeeProfileResponse>(
      `/api/v1/employees/profile/${encodeURIComponent(name)}`,
    );

    if (!profile || !profile.id) {
      return {
        content: [
          {
            type: "text",
            text: `No employee found matching '${name}'.`,
          },
        ],
        isError: true,
      };
    }

    // Then fetch their world model
    const worldModel = await client.get<EmployeeWorldModel>(
      `/api/v1/employees/${profile.id}/world-model`,
    );

    // Format as human-readable markdown
    let text = `# World Model: ${profile.name}\n\n`;

    // Skills section
    if (worldModel.skills.length > 0) {
      text += "## Skills\n";
      worldModel.skills.forEach((skill) => {
        text += `- **${skill.skill_name}**: ${skill.proficiency} (${skill.mention_count} mentions)\n`;
      });
      text += "\n";
    } else {
      text += "## Skills\n_No skills tracked yet._\n\n";
    }

    // Growth Events section
    if (worldModel.growth_events.length > 0) {
      text += "## Growth Events\n";
      worldModel.growth_events.forEach((event) => {
        text += `\n**${event.event_type}** (${event.occurred_at})\n`;
        text += `${event.description}\n`;
      });
      text += "\n";
    } else {
      text += "## Growth Events\n_No growth events recorded yet._\n\n";
    }

    // Blockers section
    if (worldModel.blockers.length > 0) {
      text += "## Blockers\n";
      worldModel.blockers.forEach((blocker) => {
        text += `\n**${blocker.category}** (${blocker.status})\n`;
        text += `${blocker.description}\n`;
        text += `- Recurrence: ${blocker.recurrence_count}x\n`;
        text += `- First seen: ${blocker.first_seen_at}\n`;
      });
    } else {
      text += "## Blockers\n_No active blockers._\n";
    }

    return {
      content: [{ type: "text", text }],
    };
  } catch (error) {
    if (error instanceof APIError && error.statusCode === 404) {
      return {
        content: [
          {
            type: "text",
            text: `No employee found matching '${name}'.`,
          },
        ],
        isError: true,
      };
    }
    return errorResult(error);
  }
}

function errorResult(error: unknown): CallToolResult {
  const message =
    error instanceof APIError
      ? error.message
      : "An unexpected error occurred.";
  return { content: [{ type: "text", text: message }], isError: true };
}
