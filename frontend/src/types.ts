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
  error_summary: string;
}

export interface ImportURLResponse {
  job: Job;
  duplicate: boolean;
  manual_only: boolean;
}
