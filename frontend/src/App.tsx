import { useEffect, useMemo, useState } from "react";
import { listJobs, runCrawl, updateJobStatus } from "./api";
import type { Job, JobStatus, RunSummary } from "./types";

const statusLabels: Record<JobStatus | "all", string> = {
  all: "All",
  new: "New",
  interested: "Interested",
  applied: "Applied",
  ignored: "Ignored",
  manual_check: "Manual check",
  expired: "Expired",
};

const directionOptions = ["all", "frontend", "backend", "java", "go", "algorithm", "ai_application"];

export default function App() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [status, setStatus] = useState<JobStatus | "all">("all");
  const [direction, setDirection] = useState("all");
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState("");
  const [lastRun, setLastRun] = useState<RunSummary | null>(null);

  async function refresh(nextStatus = status) {
    setError("");
    const data = await listJobs(nextStatus);
    setJobs(data);
  }

  useEffect(() => {
    refresh()
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const visibleJobs = useMemo(() => {
    return jobs.filter((job) => direction === "all" || job.direction_tags.includes(direction));
  }, [jobs, direction]);

  const strongMatches = jobs.filter((job) => job.match_score >= 70).length;

  async function handleStatusFilter(next: JobStatus | "all") {
    setStatus(next);
    setLoading(true);
    refresh(next)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }

  async function handleRunCrawl() {
    setRunning(true);
    setError("");
    try {
      const summary = await runCrawl();
      setLastRun(summary);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Run failed");
    } finally {
      setRunning(false);
    }
  }

  async function setJobStatus(id: number, next: JobStatus) {
    await updateJobStatus(id, next);
    setJobs((current) => current.map((job) => (job.id === id ? { ...job, status: next } : job)));
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <h1>Job Hunter Agent</h1>
          <p>Local autumn recruitment radar for Shenzhen-focused tech roles.</p>
        </div>
        <button className="primary-button" onClick={handleRunCrawl} disabled={running}>
          {running ? "Running..." : "Run Crawl"}
        </button>
      </header>

      <section className="summary-grid">
        <Metric label="Tracked jobs" value={jobs.length} />
        <Metric label="Strong matches" value={strongMatches} />
        <Metric label="Visible now" value={visibleJobs.length} />
        <Metric label="Next runs" value="09:00 / 12:00 / 18:00" />
      </section>

      {lastRun && (
        <section className="run-strip">
          <span>Created {lastRun.jobs_created}</span>
          <span>Duplicated {lastRun.jobs_duplicated}</span>
          <span>Failed sources {lastRun.sources_failed}</span>
          <span>Manual check {lastRun.manual_check_count}</span>
        </section>
      )}

      {error && <div className="error-banner">{error}</div>}

      <section className="workspace">
        <aside className="filters">
          <h2>Filters</h2>
          <label>
            Status
            <select value={status} onChange={(event) => handleStatusFilter(event.target.value as JobStatus | "all")}>
              {Object.entries(statusLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
          <label>
            Direction
            <select value={direction} onChange={(event) => setDirection(event.target.value)}>
              {directionOptions.map((value) => (
                <option key={value} value={value}>
                  {value === "all" ? "All" : value.replace("_", " ")}
                </option>
              ))}
            </select>
          </label>
        </aside>

        <section className="job-panel">
          <div className="panel-header">
            <h2>Opportunities</h2>
            {loading && <span>Loading...</span>}
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Score</th>
                  <th>Company</th>
                  <th>Role</th>
                  <th>City</th>
                  <th>Tags</th>
                  <th>Status</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {visibleJobs.map((job) => (
                  <tr key={job.id}>
                    <td>
                      <span className={`score ${job.match_score >= 70 ? "score-strong" : ""}`}>{job.match_score}</span>
                    </td>
                    <td>{job.company}</td>
                    <td>
                      <div className="role-cell">
                        <a href={job.apply_url || job.source_url} target="_blank" rel="noreferrer">
                          {job.title}
                        </a>
                        <small>{job.recommend_reasons.slice(0, 2).join(" · ") || "No reasons yet"}</small>
                      </div>
                    </td>
                    <td>{job.city || "Unknown"}</td>
                    <td>
                      <div className="tags">
                        {job.direction_tags.map((tag) => (
                          <span key={tag}>{tag.replace("_", " ")}</span>
                        ))}
                      </div>
                    </td>
                    <td>{statusLabels[job.status]}</td>
                    <td>
                      <div className="row-actions">
                        <button onClick={() => setJobStatus(job.id, "interested")}>Interested</button>
                        <button onClick={() => setJobStatus(job.id, "applied")}>Applied</button>
                        <button onClick={() => setJobStatus(job.id, "ignored")}>Ignore</button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!loading && visibleJobs.length === 0 && (
                  <tr>
                    <td colSpan={7} className="empty-state">
                      No jobs yet. Run a crawl to create the first collection record.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </section>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
