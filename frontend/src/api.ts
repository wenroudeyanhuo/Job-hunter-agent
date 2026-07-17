import type { Job, JobStatus, RunSummary } from "./types";

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
  return request<Job[]>(`/api/jobs${query}`);
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
