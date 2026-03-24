// Team tools
export interface TeamStatus {
  date: string;
  total_employees: number;
  submitted: number;
  pending: Array<{
    id: string;
    name: string;
    chase_count: number;
  }>;
  chase_count: number;
  mentor: string;
  mentor_name: string;
}

export interface TeamReport {
  period: string;
  date_range: { start: string; end: string };
  submission_rate: string;
  ranking: Array<{
    id: string;
    name: string;
    days: number;
    medal?: string;
  }>;
  one_on_one_suggestions: Array<{
    id: string;
    name: string;
    days: number;
  }>;
}

export interface Alerts {
  alerts: Array<{
    employee_id: string;
    employee_name: string;
    missed_days: number;
    severity: string;
  }>;
  total: number;
}

// Mentor tools
export interface SwitchMentorSuccess {
  result: string;
  mentor_id: string;
  name: string;
}

export interface SwitchMentorError {
  error: string;
  available_mentors: string[];
}

export interface Mentor {
  id: string;
  name: string;
  name_en: string;
  company: string;
  philosophy: string;
  domain: string;
  tags: string[];
  recommended_seats: string[];
}

// C-Suite tools
export interface BoardDiscussResponse {
  topic: string;
  responses: Array<{
    seat_type: string;
    title: string;
    persona_id: string;
    content: string;
  }>;
  synthesis: string;
}

export interface SeatChatResponse {
  seat_type: string;
  title: string;
  persona_id: string;
  response: string;
}

export interface SeatChatInactiveResponse {
  message: string;
}

// Employee tools
export interface CommandResult {
  result: string;
  employees: Array<{
    id: string;
    name: string;
    role: string;
  }>;
}

export interface EmployeeProfile {
  employee: {
    id: string;
    name: string;
    role: string;
    job_title: string;
    country: string;
  };
  submission_rate: string;
  recent_reports: Array<{
    date: string;
    sentiment: string;
    blockers: string;
  }>;
  sentiment_trend: string;
  consecutive_missed: number;
}
