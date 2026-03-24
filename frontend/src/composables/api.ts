const API_BASE = import.meta.env.VITE_API_BASE || "/api/v1";

function getToken(): string | null {
  return localStorage.getItem("token");
}

function setToken(token: string) {
  localStorage.setItem("token", token);
}

function clearToken() {
  localStorage.removeItem("token");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  };
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers });

  if (res.status === 401) {
    clearToken();
    window.location.hash = "#/login";
    throw new Error("Unauthorized");
  }

  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(body.error || `HTTP ${res.status}`);
  }

  return res.json();
}

// Auth
export async function login(email: string, password: string): Promise<string> {
  const res = await request<{ token: string }>("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  setToken(res.token);
  return res.token;
}

export async function register(
  email: string,
  password: string,
  tenantName: string,
): Promise<string> {
  const res = await request<{ token: string }>("/auth/register", {
    method: "POST",
    body: JSON.stringify({ email, password, tenant_name: tenantName }),
  });
  setToken(res.token);
  return res.token;
}

export function logout() {
  clearToken();
  window.location.hash = "#/login";
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

// Dashboard
export async function getDashboard() {
  return request<{ data: DashboardStats }>("/dashboard");
}

// Tenant
export async function getTenant() {
  return request<{ data: Tenant }>("/tenant");
}

export async function updateTenant(name: string, timezone: string) {
  return request<{ data: any }>("/tenant", {
    method: "PUT",
    body: JSON.stringify({ name, timezone }),
  });
}

// Employees
export async function listEmployees() {
  return request<{ data: Employee[] }>("/employees");
}

export async function createEmployee(data: {
  name: string;
  culture_code: string;
  job_title?: string;
  responsibilities?: string;
  country?: string;
  language?: string;
}) {
  return request<{ data: Employee }>("/employees", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateEmployeeProfile(
  id: string,
  data: { job_title?: string; responsibilities?: string; country?: string; language?: string },
) {
  return request<{ data: { job_title: string; responsibilities: string; country: string; language: string } }>(
    `/employees/${id}/profile`,
    { method: "PUT", body: JSON.stringify(data) },
  );
}

// Reports
export async function listReports(date: string) {
  return request<{ data: Report[] }>(`/reports?date=${date}`);
}

export async function getSummary(date: string) {
  return request<{ data: Summary }>(`/reports/summary?date=${date}`);
}

// Mentor
export async function getMentor() {
  return request<{ data: MentorConfig }>("/mentor");
}

export async function updateMentor(mentorId: string) {
  return request<{ data: any }>("/mentor", {
    method: "PUT",
    body: JSON.stringify({ mentor_id: mentorId }),
  });
}

export async function updateBlend(
  primaryId: string,
  secondaryId: string,
  weight: number,
) {
  return request<{ data: any }>("/mentor/blend", {
    method: "PUT",
    body: JSON.stringify({
      primary_id: primaryId,
      secondary_id: secondaryId,
      weight,
    }),
  });
}

// Analytics
export async function getAnalyticsOverview() {
  return request<{ data: AnalyticsOverview }>("/analytics/overview");
}

export async function getEmployeeActivity() {
  return request<{ data: EmployeeActivity[] }>("/analytics/activity");
}

// Organization Wizard
export async function startWizard(mentorId: string) {
  return request<{ data: WizardSession }>("/org/wizard/start", {
    method: "POST",
    body: JSON.stringify({ mentor_id: mentorId }),
  });
}

export async function answerWizard(answer: string) {
  return request<{ data: WizardAnswer }>("/org/wizard/answer", {
    method: "POST",
    body: JSON.stringify({ answer }),
  });
}

export async function getOrgPlan() {
  return request<{ data: OrgPlan }>("/org/plan");
}

export async function adjustOrgPlan(feedback: string) {
  return request<{ data: { plan: ManagementPlan; plan_version: number } }>(
    "/org/plan",
    {
      method: "PUT",
      body: JSON.stringify({ feedback }),
    },
  );
}

export async function activateOrgPlan() {
  return request<{ data: { status: string; roles_activated: number } }>(
    "/org/plan/activate",
    {
      method: "POST",
    },
  );
}

// AI Roles
export async function listAIRoles() {
  return request<{ data: AIRoleInstance[] }>("/org/roles");
}

export async function listSuggestions() {
  return request<{ data: AISuggestion[] }>("/org/suggestions");
}

export async function approveSuggestion(id: string) {
  return request<{ data: { status: string } }>(
    `/org/suggestions/${id}/approve`,
    { method: "POST" },
  );
}

export async function rejectSuggestion(id: string) {
  return request<{ data: { status: string } }>(
    `/org/suggestions/${id}/reject`,
    { method: "POST" },
  );
}

// Admin - Channels
export async function getChannelConfig() {
  return request<{ data: ChannelConfig }>("/admin/channels");
}

export async function updateChannelConfig(data: {
  enabled_channels?: string[];
  slack_bot_token?: string;
  slack_signing_secret?: string;
  lark_app_id?: string;
  lark_app_secret?: string;
  signal_phone?: string;
}) {
  return request<{ data: { updated: boolean } }>("/admin/channels", {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function testChannel(channel: string, userId: string, text?: string) {
  return request<{ data: { sent: boolean; channel: string } }>(
    `/admin/channels/test/${channel}`,
    { method: "POST", body: JSON.stringify({ user_id: userId, text }) },
  );
}

// Admin - Employees
export async function listEmployeesWithChannels() {
  return request<{ data: EmployeeWithChannels[] }>("/admin/employees");
}

export async function updateEmployeeChannels(
  id: string,
  data: { signal_phone?: string; slack_id?: string; lark_id?: string; preferred_channel?: string },
) {
  return request<{ data: { updated: boolean } }>(`/admin/employees/${id}/channels`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function updateEmployeePreferred(id: string, preferredChannel: string) {
  return request<{ data: { preferred_channel: string } }>(
    `/admin/employees/${id}/preferred`,
    { method: "PUT", body: JSON.stringify({ preferred_channel: preferredChannel }) },
  );
}

// Admin - Reports
export async function listAdminReports(params: {
  page?: number;
  limit?: number;
  date_from?: string;
  date_to?: string;
  employee_id?: string;
  channel?: string;
}) {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") qs.set(k, String(v));
  });
  return request<{ data: AdminReport[]; meta: { total: number; page: number; limit: number; has_more: boolean } }>(
    `/admin/reports?${qs}`,
  );
}

export async function getReportStats(dateFrom?: string, dateTo?: string) {
  const qs = new URLSearchParams();
  if (dateFrom) qs.set("date_from", dateFrom);
  if (dateTo) qs.set("date_to", dateTo);
  return request<{ data: { channel: string; count: number }[] }>(
    `/admin/reports/stats?${qs}`,
  );
}

// Admin - Mentors
export async function listAllMentors() {
  return request<{ data: MentorInfo[] }>("/admin/mentors");
}

// Admin - Scheduler
export async function listSchedulerJobs() {
  return request<{ data: SchedulerJob[] }>("/admin/scheduler");
}

export async function updateJobSchedule(job: string, cron: string) {
  return request<{ data: { job: string; cron: string } }>(
    `/admin/scheduler/${job}/schedule`,
    { method: "PUT", body: JSON.stringify({ cron }) },
  );
}

export async function triggerJob(job: string) {
  return request<{ data: { triggered: string } }>(
    `/admin/scheduler/${job}/trigger`,
    { method: "POST" },
  );
}

// Admin - Memories
export async function listAdminMemories(params: {
  page?: number;
  limit?: number;
  type?: string;
  tier?: string;
  employee_id?: string;
}) {
  const qs = new URLSearchParams();
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== null && v !== "") qs.set(k, String(v));
  });
  return request<{ data: MemoryItem[]; meta: { total: number; page: number; limit: number; has_more: boolean } }>(
    `/admin/memories?${qs}`,
  );
}

export async function searchAdminMemories(query: string, limit?: number) {
  return request<{ data: MemoryItem[] }>("/admin/memories/search", {
    method: "POST",
    body: JSON.stringify({ query, limit: limit || 10 }),
  });
}

export async function deleteAdminMemory(id: string) {
  return request<{ data: { deleted: boolean } }>(`/admin/memories/${id}`, {
    method: "DELETE",
  });
}

export async function getMemoryStats() {
  return request<{ data: MemoryStats }>("/admin/memories/stats");
}

// Admin - Group Chats
export async function listGroups() {
  return request<{ data: GroupChat[] }>("/admin/groups");
}

export async function updateGroup(
  id: string,
  data: { name: string; group_type: string; is_active: boolean },
) {
  return request<{ data: GroupChat }>(`/admin/groups/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteGroup(id: string) {
  return request<{ data: { deleted: boolean } }>(`/admin/groups/${id}`, {
    method: "DELETE",
  });
}

// Seats (C-Suite)
export async function listSeats() {
  return request<{ data: Seat[] }>("/seats");
}

export async function createSeat(data: { seat_type: string; persona_id: string; title?: string; scope?: string }) {
  return request<{ data: Seat }>("/seats", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateSeat(id: string, data: { title: string; persona_id: string; scope: string }) {
  return request<{ data: Seat }>(`/seats/${id}`, {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

export async function deleteSeat(id: string) {
  return request<{ data: { deleted: boolean } }>(`/seats/${id}`, {
    method: "DELETE",
  });
}

export async function boardDiscuss(topic: string) {
  return request<{ data: BoardDiscussResult }>("/board/discuss", {
    method: "POST",
    body: JSON.stringify({ topic }),
  });
}

export async function listMentorsWithDomain() {
  return request<{ data: MentorWithDomain[] }>("/mentors");
}

// Types
export interface DashboardStats {
  employee_count: number;
  today_submissions: number;
  current_mentor: string;
  last_summary_date: string;
}

export interface Tenant {
  id: string;
  name: string;
  timezone: string;
  mentor_id: string;
  mentor_blend: BlendConfig | null;
}

export interface Employee {
  id: string;
  name: string;
  culture_code: string;
  role: string;
  is_active: boolean;
  has_telegram: boolean;
  invite_code: string;
  job_title: string;
  responsibilities: string;
  country: string;
  language: string;
}

export interface Report {
  id: string;
  employee_id: string;
  employee_name: string;
  report_date: string;
  answers: any;
  submitted_at: string;
  blockers?: string;
  sentiment?: string;
}

export interface Summary {
  id: string;
  summary_date: string;
  content: string;
  submission_rate: number;
  blockers_count: number;
  key_metrics: any;
}

export interface MentorConfig {
  current_mentor_id: string;
  current_blend: BlendConfig | null;
  available_mentors: MentorInfo[];
}

export interface BlendConfig {
  primary_id: string;
  secondary_id: string;
  weight: number;
}

export interface MentorInfo {
  id: string;
  name: string;
  description: string;
}

export interface AnalyticsOverview {
  today: {
    date: string;
    reports: number;
    employees: number;
    submission_rate: number;
  };
  trend_7d: { date: string; count: number; rate: number }[];
  sentiment: Record<string, number>;
  health_score: number;
}

export interface EmployeeActivity {
  id: string;
  name: string;
  submitted_7d: number;
  missed_7d: number;
  last_sentiment: string;
  culture_code: string;
}

// Organization Wizard types
export interface WizardSession {
  session_id: string;
  mentor_id: string;
  message: string;
  is_complete: boolean;
}

export interface WizardAnswer {
  message: string;
  is_complete: boolean;
  plan?: ManagementPlan;
  profile?: OrgProfile;
}

export interface OrgProfile {
  industry: string;
  size: number;
  stage: string;
  business_model?: string;
  region?: string;
  pain_points?: string[];
}

export interface OrgPlan {
  id: string;
  industry: string;
  size: number;
  stage: string;
  mentor_id: string;
  plan: ManagementPlan;
  plan_version: number;
  status: string;
}

export interface ManagementPlan {
  management_framework: string;
  org_design: OrgDesign;
  culture_principles: string[];
  policies: Record<string, any>;
  kpi_system: KpiItem[];
  daily_questions: Record<string, string[]>;
  meeting_cadence: MeetingItem[];
  alert_rules: AlertRule[];
  reasoning: string;
}

export interface OrgDesign {
  philosophy: string;
  structure_type: string;
  units: OrgUnit[];
  support_roles?: SupportRole[];
}

export interface OrgUnit {
  name: string;
  leader_type: string;
  leader_role: string;
  size?: number;
  kpis?: string[];
}

export interface SupportRole {
  title: string;
  type: string;
  scope: string;
}

export interface KpiItem {
  name: string;
  target: string;
  frequency: string;
  owner: string;
}

export interface MeetingItem {
  name: string;
  frequency: string;
  duration: string;
  attendees: string;
  purpose: string;
}

export interface AlertRule {
  condition: string;
  action: string;
  message: string;
}

// AI Roles types
export interface AIRoleInstance {
  id: string;
  role_id: string;
  title: string;
  mentor_id: string;
  is_active: boolean;
  pending_count: number;
  created_at: string;
}

export interface AISuggestion {
  id: string;
  role_id: string;
  role_title: string;
  capability: string;
  title: string;
  content: string;
  status: string;
  created_at: string;
  reviewed_at?: string;
}

// Admin types
export interface ChannelStatus {
  type: string;
  configured: boolean;
}

export interface ChannelConfig {
  enabled_channels: string[];
  channels: ChannelStatus[];
  registered_channels: string[];
}

export interface EmployeeWithChannels {
  id: string;
  name: string;
  telegram_id: boolean;
  signal_phone: string;
  slack_id: string;
  lark_id: string;
  preferred_channel: string;
  culture_code: string;
  role: string;
}

export interface AdminReport {
  id: string;
  employee_id: string;
  employee_name: string;
  report_date: string;
  answers: Record<string, string>;
  blockers?: string;
  sentiment?: string;
  channel: string;
  submitted_at: string;
}

export interface SchedulerJob {
  name: string;
  cron: string;
  last_run: string;
  next_run: string;
}

export interface MemoryItem {
  id: string;
  tenant_id: string;
  memory_type: string;
  memory_tier: string;
  employee_id: string | null;
  content: string;
  summary: string | null;
  importance: number;
  access_count: number;
  metadata: Record<string, unknown>;
  expires_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface MemoryStats {
  total: number;
}

export interface GroupChat {
  id: string;
  platform: string;
  platform_chat_id: string;
  name: string;
  group_type: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Seat {
  id: string;
  seat_type: string;
  title: string;
  persona_id: string;
  scope: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface BoardResponse {
  seat_type: string;
  title: string;
  persona_id: string;
  content: string;
}

export interface BoardDiscussResult {
  topic: string;
  responses: BoardResponse[];
  synthesis: string;
}

export interface MentorWithDomain {
  id: string;
  name: string;
  name_en: string;
  company: string;
  philosophy: string;
  domain: string;
  tags: string[];
  recommended_seats: string[];
}
