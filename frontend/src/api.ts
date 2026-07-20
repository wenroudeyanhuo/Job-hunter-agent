import type {
  ImportURLResponse,
  Job,
  JobRun,
  JobRunSource,
  JobStatus,
  RecommendedCrawlResponse,
  RunSummary,
  SeedSourcesResult,
  Settings,
  Source,
} from "./types";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
    ...options,
  });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Request failed with ${response.status}`);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export async function listJobs(status?: JobStatus | "all"): Promise<Job[]> {
  const query = status && status !== "all" ? `?status=${encodeURIComponent(status)}` : "";
  const jobs = await request<Job[] | null>(`/api/jobs${query}`);
  return Array.isArray(jobs) ? jobs : [];
}

export async function updateJobStatus(id: number, status: JobStatus): Promise<void> {
  await request<void>(`/api/jobs/${id}/status`, {
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export async function runCrawl(): Promise<RunSummary> {
  return request<RunSummary>("/api/crawl/run", { method: "POST" });
}

export async function listRuns(): Promise<JobRun[]> {
  const runs = await request<JobRun[] | null>("/api/crawl/runs");
  return Array.isArray(runs) ? runs : [];
}

export async function listRunSources(runId: number): Promise<JobRunSource[]> {
  const results = await request<JobRunSource[] | null>(`/api/crawl/runs/${runId}/sources`);
  return Array.isArray(results) ? results : [];
}

export async function importURL(url: string): Promise<ImportURLResponse> {
  return request<ImportURLResponse>("/api/jobs/import-url", {
    method: "POST",
    body: JSON.stringify({ url }),
  });
}

export async function listSources(): Promise<Source[]> {
  const sources = await request<Source[] | null>("/api/sources");
  return Array.isArray(sources) ? sources : [];
}

export async function createSource(url: string, name = ""): Promise<Source> {
  return request<Source>("/api/sources", {
    method: "POST",
    body: JSON.stringify({ name, url, enabled: true, type: "public_url", parser_type: "generic" }),
  });
}

export async function seedRecommendedSources(): Promise<SeedSourcesResult> {
  return request<SeedSourcesResult>("/api/sources/recommended", { method: "POST" });
}

export async function runRecommendedCrawl(): Promise<RecommendedCrawlResponse> {
  return request<RecommendedCrawlResponse>("/api/crawl/recommended", { method: "POST" });
}

export async function updateSourceEnabled(id: number, enabled: boolean): Promise<void> {
  await request<void>(`/api/sources/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled }),
  });
}

export async function getSettings(): Promise<Settings> {
  return request<Settings>("/api/settings");
}

export async function updateSettings(settings: Pick<Settings, "target_cities" | "target_directions" | "excluded_keywords" | "crawl_schedule">): Promise<Settings> {
  return request<Settings>("/api/settings", {
    method: "PATCH",
    body: JSON.stringify(settings),
  });
}
