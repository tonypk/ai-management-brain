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

export async function createEmployee(name: string, cultureCode: string) {
  return request<{ data: Employee }>("/employees", {
    method: "POST",
    body: JSON.stringify({ name, culture_code: cultureCode }),
  });
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
