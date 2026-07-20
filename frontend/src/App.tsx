import { useEffect, useMemo, useState } from "react";
import {
  createSource,
  getAgentBriefing,
  getSettings,
  importURL,
  listAgentEvents,
  listJobs,
  listRunSources,
  listRuns,
  listSources,
  runCrawl,
  runRecommendedCrawl,
  seedRecommendedSources,
  updateJobStatus,
  updateSettings,
  updateSourceEnabled,
} from "./api";
import type { AgentBriefing, AgentEvent, Job, JobRun, JobRunSource, JobStatus, RunSummary, Settings, Source } from "./types";

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
const defaultSettings: Settings = {
  target_cities: ["Shenzhen"],
  target_directions: ["frontend", "backend", "java", "go", "algorithm", "ai_application"],
  excluded_keywords: ["outsourcing", "training", "bootcamp"],
  crawl_schedule: ["09:00", "12:00", "18:00"],
  feishu_configured: false,
  updated_at: "",
};

export default function App() {
  const [jobs, setJobs] = useState<Job[]>([]);
  const [status, setStatus] = useState<JobStatus | "all">("all");
  const [direction, setDirection] = useState("all");
  const [scoreView, setScoreView] = useState<"all" | "strong">("all");
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [importing, setImporting] = useState(false);
  const [importURLValue, setImportURLValue] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<number | null>(null);
  const [runSources, setRunSources] = useState<JobRunSource[]>([]);
  const [sourceURLValue, setSourceURLValue] = useState("");
  const [addingSource, setAddingSource] = useState(false);
  const [seedingSources, setSeedingSources] = useState(false);
  const [recommendedRunning, setRecommendedRunning] = useState(false);
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [settingsDraft, setSettingsDraft] = useState(settingsToDraft(defaultSettings));
  const [savingSettings, setSavingSettings] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [lastRun, setLastRun] = useState<RunSummary | null>(null);
  const [briefing, setBriefing] = useState<AgentBriefing | null>(null);
  const [agentEvents, setAgentEvents] = useState<AgentEvent[]>([]);

  async function refresh(nextStatus = status) {
    setError("");
    const data = await listJobs(nextStatus);
    setJobs(data);
  }

  async function refreshSources() {
    const data = await listSources();
    setSources(data);
  }

  async function refreshRuns() {
    const data = await listRuns();
    setRuns(data);
    if (selectedRunId === null && data.length > 0) {
      setSelectedRunId(data[0].id);
      setRunSources(await listRunSources(data[0].id));
    }
  }

  async function refreshSettings() {
    const data = await getSettings();
    const nextSettings = normalizeSettings(data);
    setSettings(nextSettings);
    setSettingsDraft(settingsToDraft(nextSettings));
  }

  async function refreshBriefing() {
    const data = await getAgentBriefing();
    setBriefing(data);
  }

  async function refreshAgentEvents() {
    const data = await listAgentEvents();
    setAgentEvents(data);
  }

  useEffect(() => {
    Promise.all([refresh(), refreshSources(), refreshRuns(), refreshSettings(), refreshBriefing(), refreshAgentEvents()])
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const visibleJobs = useMemo(() => {
    return jobs.filter((job) => {
      const directionMatches = direction === "all" || job.direction_tags.includes(direction);
      const scoreMatches = scoreView === "all" || job.match_score >= 70;
      return directionMatches && scoreMatches;
    });
  }, [jobs, direction, scoreView]);

  const strongMatches = jobs.filter((job) => job.match_score >= 70).length;

  async function handleStatusFilter(next: JobStatus | "all") {
    setStatus(next);
    setLoading(true);
    refresh(next)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }

  async function handleAgentAction(action: string) {
    switch (action) {
      case "add_recommended_and_crawl":
        await handleRecommendedCrawl();
        return;
      case "run_crawl":
        await handleRunCrawl();
        return;
      case "review_manual_check":
        setScoreView("all");
        await handleStatusFilter("manual_check");
        setNotice("Showing jobs that need manual review.");
        return;
      case "review_strong_matches":
        setStatus("all");
        setDirection("all");
        setScoreView("strong");
        await refresh("all");
        setNotice("Showing strong matches from the agent briefing.");
        return;
      case "inspect_failed_sources":
        if (runs.length > 0) {
          await selectRun(runs[0].id);
          setNotice("Opened the latest crawl run. Check source errors below.");
        }
        return;
      default:
        setNotice("The agent will keep monitoring your pipeline.");
    }
  }

  async function handleRunCrawl() {
    setRunning(true);
    setError("");
    try {
      const summary = await runCrawl();
      setLastRun(summary);
      await refresh();
      await refreshRuns();
      await refreshBriefing();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Run failed");
    } finally {
      setRunning(false);
    }
  }

  async function handleImportURL(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = importURLValue.trim();
    if (!value) {
      setError("Paste a recruitment URL first.");
      return;
    }
    setImporting(true);
    setError("");
    setNotice("");
    try {
      const result = await importURL(value);
      setImportURLValue("");
      setNotice(
        result.duplicate
          ? "This link was already tracked. Existing job is shown in the list."
          : result.manual_only
            ? "Saved for manual check because the page could not be fully read."
            : "Imported and scored the link.",
      );
      await refresh();
      await refreshBriefing();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Import failed");
    } finally {
      setImporting(false);
    }
  }

  async function handleAddSource(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = sourceURLValue.trim();
    if (!value) {
      setError("Paste a source URL first.");
      return;
    }
    setAddingSource(true);
    setError("");
    setNotice("");
    try {
      await createSource(value);
      setSourceURLValue("");
      setNotice("Source added. It will be used by the next crawl run.");
      await refreshSources();
      await refreshBriefing();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add source");
    } finally {
      setAddingSource(false);
    }
  }

  async function toggleSource(source: Source) {
    await updateSourceEnabled(source.id, !source.enabled);
    setSources((current) => current.map((item) => (item.id === source.id ? { ...item, enabled: !source.enabled } : item)));
    await refreshBriefing();
  }

  async function handleSeedRecommendedSources() {
    setSeedingSources(true);
    setError("");
    setNotice("");
    try {
      const result = await seedRecommendedSources();
      setNotice(
        result.created > 0
          ? `Added ${result.created} recommended sources.`
          : "Recommended sources were already added.",
      );
      await refreshSources();
      await refreshBriefing();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add recommended sources");
    } finally {
      setSeedingSources(false);
    }
  }

  async function handleRecommendedCrawl() {
    setRecommendedRunning(true);
    setError("");
    setNotice("");
    try {
      const result = await runRecommendedCrawl();
      setLastRun(result.summary);
      setNotice(`Recommended crawl finished. Added ${result.sources.created} sources and created ${result.summary.jobs_created} jobs.`);
      await refreshSources();
      await refresh();
      await refreshRuns();
      await refreshBriefing();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Recommended crawl failed");
    } finally {
      setRecommendedRunning(false);
    }
  }

  async function handleSaveSettings(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingSettings(true);
    setError("");
    setNotice("");
    try {
      const saved = await updateSettings({
        target_cities: parseSettingsList(settingsDraft.target_cities),
        target_directions: parseSettingsList(settingsDraft.target_directions),
        excluded_keywords: parseSettingsList(settingsDraft.excluded_keywords),
        crawl_schedule: parseSettingsList(settingsDraft.crawl_schedule),
      });
      const nextSettings = normalizeSettings(saved);
      setSettings(nextSettings);
      setSettingsDraft(settingsToDraft(nextSettings));
      setNotice("Settings saved. Future crawl and scoring steps can use these preferences.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save settings");
    } finally {
      setSavingSettings(false);
    }
  }

  async function selectRun(runId: number) {
    setSelectedRunId(runId);
    setRunSources(await listRunSources(runId));
  }

  async function setJobStatus(id: number, next: JobStatus) {
    await updateJobStatus(id, next);
    setJobs((current) => current.map((job) => (job.id === id ? { ...job, status: next } : job)));
    await refreshBriefing();
    await refreshAgentEvents();
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
        <Metric label="Next runs" value={settings.crawl_schedule.join(" / ")} />
      </section>

      {briefing && <AgentBriefingPanel briefing={briefing} onAction={handleAgentAction} busy={running || recommendedRunning} />}

      <AgentActivityLog events={agentEvents} />

      {lastRun && (
        <section className="run-strip">
          <span>Created {lastRun.jobs_created}</span>
          <span>Duplicated {lastRun.jobs_duplicated}</span>
          <span>Failed sources {lastRun.sources_failed}</span>
          <span>Manual check {lastRun.manual_check_count}</span>
        </section>
      )}

      <form className="import-bar" onSubmit={handleImportURL}>
        <input
          value={importURLValue}
          onChange={(event) => setImportURLValue(event.target.value)}
          placeholder="Paste a recruitment URL"
          aria-label="Recruitment URL"
        />
        <button type="submit" disabled={importing}>
          {importing ? "Importing..." : "Import URL"}
        </button>
      </form>

      {notice && <div className="notice-banner">{notice}</div>}
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
          <label>
            Score
            <select value={scoreView} onChange={(event) => setScoreView(event.target.value as "all" | "strong")}>
              <option value="all">All</option>
              <option value="strong">Strong matches</option>
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

      <section className="sources-panel">
        <div className="panel-header">
          <h2>Sources</h2>
          <span>{sources.filter((source) => source.enabled).length} enabled</span>
        </div>
        <div className="source-actions">
          <button type="button" onClick={handleSeedRecommendedSources} disabled={seedingSources || recommendedRunning}>
            {seedingSources ? "Adding..." : "Add Recommended"}
          </button>
          <button type="button" className="strong-action" onClick={handleRecommendedCrawl} disabled={recommendedRunning || seedingSources}>
            {recommendedRunning ? "Running..." : "Add & Crawl"}
          </button>
        </div>
        <form className="source-form" onSubmit={handleAddSource}>
          <input
            value={sourceURLValue}
            onChange={(event) => setSourceURLValue(event.target.value)}
            placeholder="Add a public recruitment source URL"
            aria-label="Source URL"
          />
          <button type="submit" disabled={addingSource}>
            {addingSource ? "Adding..." : "Add Source"}
          </button>
        </form>
        <div className="source-list">
          {sources.map((source) => (
            <div className="source-row" key={source.id}>
              <div>
                <strong>{source.name}</strong>
                <a href={source.url} target="_blank" rel="noreferrer">
                  {source.url}
                </a>
              </div>
              <button className={source.enabled ? "toggle-on" : "toggle-off"} onClick={() => toggleSource(source)}>
                {source.enabled ? "Enabled" : "Disabled"}
              </button>
            </div>
          ))}
          {sources.length === 0 && <div className="empty-source">No saved sources yet.</div>}
        </div>
      </section>

      <section className="settings-panel">
        <div className="panel-header">
          <h2>Settings</h2>
          <span>{settings.feishu_configured ? "Feishu ready" : "Feishu not configured"}</span>
        </div>
        <form className="settings-grid" onSubmit={handleSaveSettings}>
          <label>
            Target cities
            <textarea
              value={settingsDraft.target_cities}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, target_cities: event.target.value }))}
            />
          </label>
          <label>
            Directions
            <textarea
              value={settingsDraft.target_directions}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, target_directions: event.target.value }))}
            />
          </label>
          <label>
            Excluded keywords
            <textarea
              value={settingsDraft.excluded_keywords}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, excluded_keywords: event.target.value }))}
            />
          </label>
          <label>
            Crawl schedule
            <textarea
              value={settingsDraft.crawl_schedule}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, crawl_schedule: event.target.value }))}
            />
          </label>
          <button type="submit" disabled={savingSettings}>
            {savingSettings ? "Saving..." : "Save Settings"}
          </button>
        </form>
      </section>

      <section className="runs-panel">
        <div className="panel-header">
          <h2>Crawl Runs</h2>
          <span>{runs.length} recorded</span>
        </div>
        <div className="runs-layout">
          <div className="run-list">
            {runs.map((run) => (
              <button
                className={run.id === selectedRunId ? "run-row selected-run" : "run-row"}
                key={run.id}
                onClick={() => selectRun(run.id)}
              >
                <span>
                  <strong>{run.status}</strong>
                  <small>{new Date(run.started_at).toLocaleString()}</small>
                </span>
                <span className="run-counts">
                  +{run.jobs_created} / dup {run.jobs_duplicated} / fail {run.sources_failed}
                </span>
              </button>
            ))}
            {runs.length === 0 && <div className="empty-source">No crawl runs yet.</div>}
          </div>
          <div className="run-detail">
            {runSources.map((source) => (
              <div className="run-source-row" key={source.id}>
                <div>
                  <strong>{source.source_name || "source"}</strong>
                  {source.source_url && (
                    <a href={source.source_url} target="_blank" rel="noreferrer">
                      {source.source_url}
                    </a>
                  )}
                  {source.error_message && <small className="source-error">{source.error_message}</small>}
                </div>
                <div className="run-source-metrics">
                  <span>{source.status}</span>
                  <span>found {source.jobs_found}</span>
                  <span>new {source.jobs_created}</span>
                  <span>dup {source.jobs_duplicated}</span>
                  <span>filtered {source.jobs_filtered}</span>
                  <span>manual {source.manual_check_count}</span>
                </div>
              </div>
            ))}
            {selectedRunId !== null && runSources.length === 0 && <div className="empty-source">No source results for this run.</div>}
          </div>
        </div>
      </section>
    </main>
  );
}

function AgentBriefingPanel({
  briefing,
  onAction,
  busy,
}: {
  briefing: AgentBriefing;
  onAction: (action: string) => void | Promise<void>;
  busy: boolean;
}) {
  return (
    <section className={`agent-briefing agent-${briefing.tone}`}>
      <div>
        <div className="agent-kicker">Agent Briefing</div>
        <h2>{briefing.headline}</h2>
        <div className="agent-highlights">
          {briefing.highlights.length > 0 ? (
            briefing.highlights.map((highlight) => <span key={highlight}>{highlight}</span>)
          ) : (
            <span>Waiting for the next crawl signal.</span>
          )}
        </div>
      </div>
      <div className="agent-metrics">
        <Metric label="Strong" value={briefing.metrics.strong_matches} />
        <Metric label="Manual" value={briefing.metrics.manual_check_jobs} />
        <Metric label="Sources" value={briefing.metrics.enabled_sources} />
      </div>
      <div className="agent-actions">
        {briefing.next_actions.map((action) => (
          <div className="agent-action" key={action.action}>
            <strong>{action.label}</strong>
            <span>{action.reason}</span>
            <button type="button" onClick={() => onAction(action.action)} disabled={busy}>
              Do it
            </button>
          </div>
        ))}
      </div>
    </section>
  );
}

function AgentActivityLog({ events }: { events: AgentEvent[] }) {
  return (
    <section className="activity-panel">
      <div className="panel-header">
        <h2>Activity Log</h2>
        <span>{events.length} recent</span>
      </div>
      <div className="activity-list">
        {events.map((event) => (
          <div className={`activity-row activity-${event.level}`} key={event.id}>
            <div>
              <strong>{event.title}</strong>
              <span>{event.summary}</span>
            </div>
            <time>{new Date(event.created_at).toLocaleString()}</time>
          </div>
        ))}
        {events.length === 0 && <div className="empty-source">No agent activity recorded yet.</div>}
      </div>
    </section>
  );
}

function settingsToDraft(settings: Settings) {
  return {
    target_cities: safeSettingsList(settings.target_cities, defaultSettings.target_cities).join("\n"),
    target_directions: safeSettingsList(settings.target_directions, defaultSettings.target_directions).join("\n"),
    excluded_keywords: safeSettingsList(settings.excluded_keywords, defaultSettings.excluded_keywords).join("\n"),
    crawl_schedule: safeSettingsList(settings.crawl_schedule, defaultSettings.crawl_schedule).join("\n"),
  };
}

function normalizeSettings(settings: Partial<Settings>): Settings {
  return {
    target_cities: safeSettingsList(settings.target_cities, defaultSettings.target_cities),
    target_directions: safeSettingsList(settings.target_directions, defaultSettings.target_directions),
    excluded_keywords: safeSettingsList(settings.excluded_keywords, defaultSettings.excluded_keywords),
    crawl_schedule: safeSettingsList(settings.crawl_schedule, defaultSettings.crawl_schedule),
    feishu_configured: Boolean(settings.feishu_configured),
    updated_at: settings.updated_at || "",
  };
}

function safeSettingsList(values: unknown, fallback: string[]) {
  if (!Array.isArray(values)) {
    return fallback;
  }
  const cleaned = values.filter((value): value is string => typeof value === "string" && value.trim() !== "");
  return cleaned.length > 0 ? cleaned : fallback;
}

function parseSettingsList(value: string) {
  const seen = new Set<string>();
  return value
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter((item) => {
      const key = item.toLowerCase();
      if (!item || seen.has(key)) {
        return false;
      }
      seen.add(key);
      return true;
    });
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
