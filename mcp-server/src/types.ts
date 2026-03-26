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

// Brain Layer v2 tools
export interface CompanyState {
  top_risks: Array<{
    signal_type: string;
    score: string;
    reasons: string;
  }>;
  overdue_tasks: Array<{
    id: string;
    title: string;
    priority: string;
    due_at: string;
  }>;
  task_stats: Array<{
    status: string;
    count: number;
  }>;
  event_counts: Array<{
    event_type: string;
    count: number;
  }>;
  blocked_projects: Array<{
    id: string;
    name: string;
    status: string;
  }>;
  working_memory: Record<string, unknown> | null;
}

export interface ExecutionSignal {
  id: string;
  signal_type: string;
  score: string;
  reasons: string;
  subject_type: string;
  subject_id: string;
  time_window: string;
  generated_at: string;
}

export interface CommunicationEvent {
  id: string;
  event_type: string;
  payload: string;
  confidence: string;
  source_type: string;
  platform: string;
  occurred_at: string;
}

export interface MetricWithValue {
  id: string;
  name: string;
  unit: string;
  latest_value: string;
  target_value: string;
  owner_name: string;
}

export interface IncentiveScore {
  id: string;
  rule_id: string;
  person_id: string;
  period: string;
  score: string;
  score_breakdown: string;
  payout_weight: string;
  attribution_confidence: string;
  status: string;
}

// Action tools (write operations)
export interface CheckinResult {
  sent_to: string[];
  skipped: string[];
}

export interface ChaseResult {
  chased: string[];
  skipped: string[];
}

export interface SummaryActionResult {
  summary: string;
  submission_rate: number;
  sent_to: string;
}

export interface MessageResult {
  sent_to: string;
  channel: string;
}
