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

export interface Settings {
  target_cities: string[];
  target_directions: string[];
  excluded_keywords: string[];
  crawl_schedule: string[];
  feishu_configured: boolean;
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

export interface AgentDutyReport {
  generated_at: string;
  tone: string;
  headline: string;
  summary: AgentDutySummary;
  todays_work: AgentWorkItem[];
  needs_decision: AgentDecisionItem[];
  source_issues: AgentSourceIssue[];
  next_best_action: AgentReportAction;
  latest_run?: JobRun;
}

export interface AgentDutySummary {
  jobs_to_review: number;
  strong_matches: number;
  manual_check: number;
  source_issues: number;
  new_jobs: number;
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
