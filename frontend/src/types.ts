export type JobStatus = "new" | "interested" | "applied" | "ignored" | "manual_check" | "expired";

export interface Job {
  id: number;
  company: string;
  title: string;
  city: string;
  direction_tags: string[];
  description: string;
  source_name: string;
  source_url: string;
  apply_url: string;
  discovered_at: string;
  match_score: number;
  recommend_reasons: string[];
  penalty_reasons: string[];
  status: JobStatus;
  notes: string;
}

export interface CandidateProfile {
  id: number;
  target_cities: string[];
  target_directions: string[];
  skills: string[];
  education: string;
  graduation_year: string;
  internship_preference: string;
  preferred_companies: string[];
  blocked_keywords: string[];
  notes: string;
  updated_at: string;
}

export interface JobDecision {
  id: number;
  job_id: number;
  action: string;
  reason: string;
  from_status: string;
  to_status: string;
  notes: string;
  created_at: string;
}

export interface JobFitSummary {
  score: number;
  verdict: string;
  strengths: string[];
  risks: string[];
  profile_signals: string[];
}

export interface JobDetail {
  job: Job;
  fit: JobFitSummary;
  decisions: JobDecision[];
  suggested_action: AgentReportAction;
}

export interface RunSummary {
  sources_total: number;
  sources_success: number;
  sources_failed: number;
  jobs_found: number;
  jobs_created: number;
  jobs_duplicated: number;
  manual_check_count: number;
  landing_pages_ignored: number;
  error_summary: string;
}

export interface JobRun extends RunSummary {
  id: number;
  trigger_type: string;
  started_at: string;
  finished_at?: string;
  status: string;
}

export interface JobRunSource {
  id: number;
  job_run_id: number;
  source_name: string;
  source_url: string;
  status: string;
  jobs_found: number;
  jobs_created: number;
  jobs_duplicated: number;
  jobs_filtered: number;
  manual_check_count: number;
  error_message: string;
}

export interface ImportURLResponse {
  job: Job;
  duplicate: boolean;
  manual_only: boolean;
}

export interface CleanupLandingPagesResponse {
  ignored: number;
}

export interface Source {
  id: number;
  name: string;
  type: string;
  url: string;
  enabled: boolean;
  category: string;
  parser_type: string;
  last_run_at?: string;
  health_status: "unknown" | "healthy" | "warning" | "broken" | string;
  health_reason: string;
  consecutive_failures: number;
  last_success_at?: string;
  last_failure_at?: string;
  last_found_count: number;
}

export interface Company {
  id: number;
  name: string;
  category: string;
  enabled: boolean;
  priority: number;
  notes: string;
  source_count: number;
  healthy_count: number;
  warning_count: number;
  broken_count: number;
  created_at: string;
  updated_at: string;
}

export interface Settings {
  target_cities: string[];
  target_directions: string[];
  excluded_keywords: string[];
  crawl_schedule: string[];
  feishu_webhook_url: string;
  feishu_configured: boolean;
  auto_duty_report_enabled: boolean;
  duty_report_time: string;
  task_sla_hours: number;
  last_duty_report_sent_at?: string;
  updated_at: string;
}

export interface SeedSourcesResult {
  total: number;
  created: number;
  duplicated: number;
}

export interface RecommendedCrawlResponse {
  seeded: number;
  sources: SeedSourcesResult;
  summary: RunSummary;
}

export interface AgentBriefing {
  generated_at: string;
  tone: "steady" | "needs_setup" | "needs_review" | "needs_attention" | string;
  headline: string;
  metrics: AgentMetrics;
  latest_run?: JobRun;
  highlights: string[];
  next_actions: AgentNextAction[];
}

export interface AgentState {
  generated_at: string;
  profile: AgentProfile;
  mode: string;
  focus: string;
  maturity_score: number;
  workload: AgentWorkload;
  automation: AgentAutomationState;
  capabilities: AgentCapability[];
  gaps: AgentCapabilityGap[];
  operating_cycle: AgentOperatingMoment[];
}

export interface AgentProfile {
  name: string;
  role: string;
  mission: string;
  avatar: string;
  presence: string;
}

export interface AgentWorkload {
  open_tasks: number;
  done_tasks: number;
  strong_matches: number;
  manual_decisions: number;
  source_issues: number;
}

export interface AgentCapability {
  key: string;
  label: string;
  status: string;
  level: number;
  evidence: string;
}

export interface AgentCapabilityGap {
  key: string;
  label: string;
  why: string;
  next_step: string;
}

export interface AgentOperatingMoment {
  time: string;
  title: string;
  state: string;
}

export interface AgentAutomationState {
  duty_report_enabled: boolean;
  duty_report_time: string;
  last_report_sent_at?: string;
  next_duty_report_at: string;
  task_sla_hours: number;
  stale_task_count: number;
  stale_tasks: AgentStaleTask[];
}

export interface AgentStaleTask {
  id: number;
  title: string;
  detail: string;
  age_hours: number;
}

export interface AgentCommandResult {
  input: string;
  intent: string;
  summary: string;
  actions: AgentCommandAction[];
  needs: string[];
}

export interface AgentChatStatus {
  mode: string;
  model: string;
  configured: boolean;
  fallback_mode: string;
}

export interface AgentChatMessage {
  id: number;
  role: "user" | "assistant" | string;
  content: string;
  source: string;
  created_at: string;
}

export interface AgentChatReply {
  content: string;
  source: string;
  actions: AgentCommandAction[];
}

export interface AgentChatResponse {
  message: AgentChatMessage;
  reply: AgentChatReply;
}

export interface AgentCommandAction {
  type: string;
  target: string;
  detail: string;
}

export interface AgentDutyReport {
  generated_at: string;
  tone: string;
  headline: string;
  summary: AgentDutySummary;
  todays_work: AgentWorkItem[];
  needs_decision: AgentDecisionItem[];
  source_issues: AgentSourceIssue[];
  tasks: AgentTask[];
  next_best_action: AgentReportAction;
  latest_run?: JobRun;
}

export interface AgentReview {
  generated_at: string;
  health: AgentReviewHealth;
  focus: AgentReviewFocus;
  findings: AgentReviewFinding[];
  decisions: AgentReviewDecision[];
  next_steps: AgentReviewStep[];
}

export interface AgentReviewHealth {
  score: number;
  label: string;
  reason: string;
}

export interface AgentReviewFocus {
  title: string;
  detail: string;
  action: string;
}

export interface AgentReviewFinding {
  kind: string;
  title: string;
  detail: string;
  level: string;
  metric: number;
}

export interface AgentReviewDecision {
  question: string;
  context: string;
  action: string;
}

export interface AgentReviewStep {
  label: string;
  reason: string;
  action: string;
}

export interface AgentDutySummary {
  jobs_to_review: number;
  strong_matches: number;
  manual_check: number;
  source_issues: number;
  new_jobs: number;
  open_tasks: number;
  done_tasks: number;
  stale_tasks: number;
  escalated_tasks: number;
}

export interface AgentTask {
  id: number;
  task_date: string;
  kind: string;
  title: string;
  detail: string;
  status: "open" | "stale" | "escalated" | "snoozed" | "done" | string;
  priority: number;
  count: number;
  subject_id: number;
  job_id: number;
  source_id: number;
  action: string;
  completion_reason: string;
  snoozed_until?: string;
  escalated_at?: string;
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface AgentWorkItem {
  kind: string;
  title: string;
  detail: string;
  priority: number;
  count: number;
}

export interface AgentDecisionItem {
  job_id: number;
  company: string;
  job_title: string;
  city: string;
  reason: string;
  score: number;
}

export interface AgentSourceIssue {
  source_id: number;
  name: string;
  url: string;
  status: string;
  reason: string;
  consecutive_failures: number;
  last_found_count: number;
}

export interface AgentReportAction {
  action: string;
  label: string;
  reason: string;
}

export interface AgentMetrics {
  total_jobs: number;
  strong_matches: number;
  manual_check_jobs: number;
  low_confidence_jobs: number;
  interested_jobs: number;
  applied_jobs: number;
  enabled_sources: number;
  broken_sources: number;
}

export interface AgentNextAction {
  action: string;
  label: string;
  reason: string;
  priority: number;
}

export interface AgentEvent {
  id: number;
  type: string;
  title: string;
  summary: string;
  level: string;
  created_at: string;
}
