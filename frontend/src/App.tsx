import { useEffect, useMemo, useState } from "react";
import {
  cleanupLandingPages,
  createSource,
  getAgentBriefing,
  getAgentDutyReport,
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
import type { AgentBriefing, AgentDutyReport, AgentEvent, Job, JobRun, JobRunSource, JobStatus, RunSummary, Settings, Source } from "./types";

const statusLabels: Record<JobStatus | "all", string> = {
  all: "All",
  new: "New",
  interested: "Interested",
  applied: "Applied",
  ignored: "Ignored",
  manual_check: "Manual check",
  expired: "Expired",
};

const sourceHealthLabels: Record<string, string> = {
  healthy: "Healthy",
  warning: "Warning",
  broken: "Broken",
  unknown: "Unknown",
};

type AppView = "dashboard" | "opportunities" | "companies" | "runs" | "settings";

const appViews: Array<{ id: AppView; label: string }> = [
  { id: "dashboard", label: "Dashboard" },
  { id: "opportunities", label: "Opportunities" },
  { id: "companies", label: "Companies" },
  { id: "runs", label: "Runs" },
  { id: "settings", label: "Settings" },
];

const categoryLabels: Record<string, string> = {
  all: "All categories",
  internet: "Internet",
  ai: "AI",
  hardware: "Hardware",
  fintech: "Fintech",
  game: "Games",
  new_energy: "New energy",
  software: "Software",
  security: "Security",
  logistics: "Logistics",
  medical: "Medical",
  manufacturing: "Manufacturing",
  custom: "Custom",
  general: "General",
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
  const [activeView, setActiveView] = useState<AppView>("dashboard");
  const [jobs, setJobs] = useState<Job[]>([]);
  const [status, setStatus] = useState<JobStatus | "all">("all");
  const [direction, setDirection] = useState("all");
  const [scoreView, setScoreView] = useState<"all" | "strong" | "low_confidence">("all");
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [importing, setImporting] = useState(false);
  const [cleaningLandingPages, setCleaningLandingPages] = useState(false);
  const [importURLValue, setImportURLValue] = useState("");
  const [sources, setSources] = useState<Source[]>([]);
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<number | null>(null);
  const [runSources, setRunSources] = useState<JobRunSource[]>([]);
  const [sourceURLValue, setSourceURLValue] = useState("");
  const [companyCategoryFilter, setCompanyCategoryFilter] = useState("all");
  const [companyQuery, setCompanyQuery] = useState("");
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
  const [dutyReport, setDutyReport] = useState<AgentDutyReport | null>(null);
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

  async function refreshDutyReport() {
    const data = await getAgentDutyReport();
    setDutyReport(data);
  }

  async function refreshAgentEvents() {
    const data = await listAgentEvents();
    setAgentEvents(data);
  }

  useEffect(() => {
    Promise.all([refresh(), refreshSources(), refreshRuns(), refreshSettings(), refreshBriefing(), refreshDutyReport(), refreshAgentEvents()])
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const visibleJobs = useMemo(() => {
    return jobs.filter((job) => {
      const directionMatches = direction === "all" || job.direction_tags.includes(direction);
      const lowConfidence = job.penalty_reasons.includes("Low confidence job posting");
      const scoreMatches =
        scoreView === "all" ||
        (scoreView === "strong" && job.match_score >= 70) ||
        (scoreView === "low_confidence" && lowConfidence);
      return directionMatches && scoreMatches;
    });
  }, [jobs, direction, scoreView]);

  const strongMatches = jobs.filter((job) => job.match_score >= 70).length;
  const enabledSources = sources.filter((source) => source.enabled).length;
  const companyCategories = useMemo(() => {
    const categories = new Set<string>();
    sources.forEach((source) => categories.add(source.category || "general"));
    return ["all", ...Array.from(categories).sort()];
  }, [sources]);
  const visibleSources = useMemo(() => {
    const query = companyQuery.trim().toLowerCase();
    return sources.filter((source) => {
      const category = source.category || "general";
      const categoryMatches = companyCategoryFilter === "all" || category === companyCategoryFilter;
      const queryMatches =
        query === "" ||
        source.name.toLowerCase().includes(query) ||
        source.url.toLowerCase().includes(query) ||
        category.toLowerCase().includes(query);
      return categoryMatches && queryMatches;
    });
  }, [sources, companyCategoryFilter, companyQuery]);

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
        setActiveView("opportunities");
        setScoreView("all");
        await handleStatusFilter("manual_check");
        setNotice("Showing jobs that need manual review.");
        return;
      case "review_low_confidence":
        setActiveView("opportunities");
        setStatus("manual_check");
        setDirection("all");
        setScoreView("low_confidence");
        await refresh("manual_check");
        setNotice("Showing low-confidence pages that need a human decision.");
        return;
      case "cleanup_landing_pages":
        await handleCleanupLandingPages();
        return;
      case "review_strong_matches":
        setActiveView("opportunities");
        setStatus("all");
        setDirection("all");
        setScoreView("strong");
        await refresh("all");
        setNotice("Showing strong matches from the agent briefing.");
        return;
      case "inspect_failed_sources":
        setActiveView("runs");
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
      setNotice(`Crawl finished. Created ${summary.jobs_created} jobs and cleaned ${summary.landing_pages_ignored} landing pages.`);
      await refresh();
      await refreshRuns();
      await refreshBriefing();
      await refreshDutyReport();
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
      await refreshDutyReport();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Import failed");
    } finally {
      setImporting(false);
    }
  }

  async function handleCleanupLandingPages() {
    setCleaningLandingPages(true);
    setError("");
    setNotice("");
    try {
      const result = await cleanupLandingPages();
      setNotice(
        result.ignored > 0
          ? `Moved ${result.ignored} recruitment landing pages to ignored.`
          : "No recruitment landing pages needed cleanup.",
      );
      await refresh();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Cleanup failed");
    } finally {
      setCleaningLandingPages(false);
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
      await refreshDutyReport();
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
    await refreshDutyReport();
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
      await refreshDutyReport();
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
      setNotice(
        `Recommended crawl finished. Added ${result.sources.created} sources, created ${result.summary.jobs_created} jobs, and cleaned ${result.summary.landing_pages_ignored} landing pages.`,
      );
      await refreshSources();
      await refresh();
      await refreshRuns();
      await refreshBriefing();
      await refreshDutyReport();
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
      await refreshDutyReport();
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
    await refreshDutyReport();
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

      <nav className="view-nav" aria-label="Primary views">
        {appViews.map((view) => (
          <button key={view.id} className={activeView === view.id ? "active-view" : ""} onClick={() => setActiveView(view.id)}>
            {view.label}
          </button>
        ))}
      </nav>

      {notice && <div className="notice-banner">{notice}</div>}
      {error && <div className="error-banner">{error}</div>}

      {activeView === "dashboard" && (
        <>
          <section className="summary-grid">
            <Metric label="Tracked jobs" value={jobs.length} />
            <Metric label="Strong matches" value={strongMatches} />
            <Metric label="Enabled companies" value={enabledSources} />
            <Metric label="Next runs" value={settings.crawl_schedule.join(" / ")} />
          </section>

          {briefing && <AgentBriefingPanel briefing={briefing} onAction={handleAgentAction} busy={running || recommendedRunning} />}

          {dutyReport && <AgentDutyReportPanel report={dutyReport} onAction={handleAgentAction} busy={running || recommendedRunning} />}

          <AgentActivityLog events={agentEvents} />

          {lastRun && (
            <section className="run-strip">
              <span>Created {lastRun.jobs_created}</span>
              <span>Duplicated {lastRun.jobs_duplicated}</span>
              <span>Failed sources {lastRun.sources_failed}</span>
              <span>Manual check {lastRun.manual_check_count}</span>
              <span>Cleaned {lastRun.landing_pages_ignored}</span>
            </section>
          )}
        </>
      )}

      {activeView === "opportunities" && (
        <>
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
            <button type="button" className="secondary-action" onClick={handleCleanupLandingPages} disabled={cleaningLandingPages}>
              {cleaningLandingPages ? "Cleaning..." : "Clean landing pages"}
            </button>
          </form>

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
            <select value={scoreView} onChange={(event) => setScoreView(event.target.value as "all" | "strong" | "low_confidence")}>
              <option value="all">All</option>
              <option value="strong">Strong matches</option>
              <option value="low_confidence">Low confidence</option>
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
                        {job.penalty_reasons.length > 0 && <small className="penalty-line">{job.penalty_reasons.slice(0, 2).join(" | ")}</small>}
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
        </>
      )}

      {activeView === "companies" && (
      <section className="sources-panel">
        <div className="panel-header">
          <h2>Companies</h2>
          <span>{enabledSources} enabled / {sources.length} total</span>
        </div>
        <div className="company-toolbar">
          <input
            value={companyQuery}
            onChange={(event) => setCompanyQuery(event.target.value)}
            placeholder="Search company or source URL"
            aria-label="Search companies"
          />
          <select value={companyCategoryFilter} onChange={(event) => setCompanyCategoryFilter(event.target.value)}>
            {companyCategories.map((category) => (
              <option key={category} value={category}>
                {categoryLabels[category] || category}
              </option>
            ))}
          </select>
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
          {visibleSources.map((source) => (
            <div className="source-row" key={source.id}>
              <div>
                <strong>{source.name}</strong>
                <div className="source-meta">
                  <span>{categoryLabels[source.category] || source.category || "General"}</span>
                  <span>{source.parser_type || "generic"}</span>
                </div>
                <a href={source.url} target="_blank" rel="noreferrer">
                  {source.url}
                </a>
                <div className="source-health">
                  <span className={`health-badge health-${source.health_status || "unknown"}`}>
                    {sourceHealthLabels[source.health_status] || source.health_status || "Unknown"}
                  </span>
                  <span>{source.health_reason || "Waiting for first crawl"}</span>
                  <span>found {source.last_found_count ?? 0}</span>
                  {source.consecutive_failures > 0 && <span>failures {source.consecutive_failures}</span>}
                </div>
              </div>
              <button className={source.enabled ? "toggle-on" : "toggle-off"} onClick={() => toggleSource(source)}>
                {source.enabled ? "Enabled" : "Disabled"}
              </button>
            </div>
          ))}
          {visibleSources.length === 0 && <div className="empty-source">No companies match the current filters.</div>}
        </div>
      </section>
      )}

      {activeView === "settings" && (
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
      )}

      {activeView === "runs" && (
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
      )}
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
        <Metric label="Low conf" value={briefing.metrics.low_confidence_jobs} />
        <Metric label="Sources" value={briefing.metrics.enabled_sources} />
        <Metric label="Broken" value={briefing.metrics.broken_sources} />
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

function AgentDutyReportPanel({
  report,
  onAction,
  busy,
}: {
  report: AgentDutyReport;
  onAction: (action: string) => void | Promise<void>;
  busy: boolean;
}) {
  const topDecision = report.needs_decision.slice(0, 3);
  const sourceIssues = report.source_issues.slice(0, 3);
  return (
    <section className={`duty-report duty-${report.tone}`}>
      <div className="panel-header">
        <div>
          <h2>Today's Work</h2>
          <span>{report.headline}</span>
        </div>
        <button type="button" onClick={() => onAction(report.next_best_action.action)} disabled={busy}>
          {report.next_best_action.label}
        </button>
      </div>
      <div className="duty-grid">
        <div className="duty-column">
          <h3>Queue</h3>
          {report.todays_work.map((item) => (
            <div className="duty-item" key={item.kind}>
              <div>
                <strong>{item.title}</strong>
                <span>{item.detail}</span>
              </div>
              <b>{item.count}</b>
            </div>
          ))}
          {report.todays_work.length === 0 && <div className="empty-source">No active work queued.</div>}
        </div>
        <div className="duty-column">
          <h3>Needs Your Decision</h3>
          {topDecision.map((item) => (
            <div className="decision-item" key={`${item.job_id}-${item.job_title}`}>
              <strong>{item.company} · {item.job_title}</strong>
              <span>{item.city} · score {item.score}</span>
              <small>{item.reason}</small>
            </div>
          ))}
          {topDecision.length === 0 && <div className="empty-source">No manual decisions waiting.</div>}
        </div>
        <div className="duty-column">
          <h3>Source Issues</h3>
          {sourceIssues.map((issue) => (
            <div className={`source-issue issue-${issue.status}`} key={issue.source_id || issue.url}>
              <strong>{issue.name}</strong>
              <span>{sourceHealthLabels[issue.status] || issue.status} · {issue.reason}</span>
              <small>found {issue.last_found_count} · failures {issue.consecutive_failures}</small>
            </div>
          ))}
          {sourceIssues.length === 0 && <div className="empty-source">Sources look stable.</div>}
        </div>
      </div>
      <div className="duty-summary">
        <span>{report.summary.new_jobs} new</span>
        <span>{report.summary.strong_matches} strong</span>
        <span>{report.summary.manual_check} manual</span>
        <span>{report.summary.source_issues} source issues</span>
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
