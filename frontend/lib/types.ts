// Types mirror the Go models in internal/models/models.go.
// Priority: 0=low 1=medium 2=high 3=urgent
// TodoStatus: 0=pending 1=in_progress 2=done 3=archived

export interface Weather {
  city: string;
  temperature: number;
  feels_like: number;
  humidity: number;
  description: string;
  icon: string;
  wind_speed: number;
  units: string;
  updated_at: string;
}

export interface CalendarEvent {
  id: string;
  title: string;
  description: string;
  location: string;
  start_time: string;
  end_time: string;
  all_day: boolean;
  meeting_url: string;
  attendees: string[];
  source: string;
}

export interface NewsItem {
  title: string;
  description: string;
  url: string;
  source: string;
  published_at: string;
  image_url: string;
}

export interface SlackMessage {
  channel: string;
  user: string;
  text: string;
  timestamp: string;
  thread_ts: string;
  is_urgent: boolean;
  is_dm: boolean;
}

export interface EmailMessage {
  id: string;
  from: string;
  subject: string;
  snippet: string;
  date: string;
  is_unread: boolean;
  is_starred: boolean;
  labels: string[];
  thread_id: string;
}

export interface GitHubNotification {
  id: string;
  title: string;
  repo: string;
  type: string;
  url: string;
  reason: string;
  unread: boolean;
  updated_at: string;
}

export interface JiraTicket {
  key: string;
  summary: string;
  status: string;
  priority: string;
  assignee: string;
  due_date: string;
  url: string;
  type: string;
}

export interface NotionPage {
  id: string;
  title: string;
  status: string;
  priority: string;
  due_date?: string;
  url: string;
  database: string;
  updated_at: string;
}

export interface RecurringRule {
  frequency: 'daily' | 'weekdays' | 'weekly';
  enabled: boolean;
}

export interface TodoItem {
  id: string;
  title: string;
  description: string;
  priority: 0 | 1 | 2 | 3;
  status: 0 | 1 | 2 | 3;
  source: string;
  source_id: string;
  source_url: string;
  due_date?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
  tags: string[];
  notes: string;
  recurring?: RecurringRule;
}

export interface TimeBlock {
  start: string;
  end: string;
  title: string;
  todo_id?: string;
  notes?: string;
  color?: 'red' | 'orange' | 'blue' | 'gray';
}

export interface Briefing {
  date: string;
  weather?: Weather;
  events: CalendarEvent[];
  news: NewsItem[];
  unread_emails: EmailMessage[];
  slack_messages: SlackMessage[];
  github_notifications: GitHubNotification[];
  jira_tickets: JiraTicket[];
  notion_pages: NotionPage[];
  todos: TodoItem[];
  email_count: number;
  slack_unread: number;
  summary?: string;
  generated_at: string;
  integration_statuses: Record<string, boolean>;
  integration_available: Record<string, boolean>;
  integration_errors?: Record<string, string>;
}

export interface DashboardUpdate {
  type: 'full_refresh' | 'todos_updated';
  payload: unknown;
}

// Mirrors Go's config.Config, with sensitive fields shown as "***" when set.
export interface ConfigResponse {
  server: { port: number; host: string; poll_interval_seconds: number };
  weather: { api_key: string; city: string; country: string; units: string; lat: number; lon: number; enabled: boolean };
  news: { api_key: string; sources: string[]; country: string; categories: string[]; max_items: number; enabled: boolean };
  google: { client_id: string; client_secret: string; ical_url: string };
  slack: { bot_token: string; channels: string[]; enabled: boolean };
  email: { imap_server: string; imap_port: number; username: string; password: string; enabled: boolean };
  github: { token: string; repos: string[]; enabled: boolean };
  jira: { base_url: string; email: string; api_token: string; project_key: string; enabled: boolean };
  notion: { token: string; database_id: string; enabled: boolean };
  ai: { api_key: string; model: string; enabled: boolean };
  pomodoro: { work_minutes: number; break_minutes: number; enabled: boolean };
}
