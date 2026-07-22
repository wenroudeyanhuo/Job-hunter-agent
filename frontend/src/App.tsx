import { useEffect, useMemo, useState } from "react";
import {
  acceptSourceCandidate,
  cleanupLandingPages,
  createSource,
  getAutomationStatus,
  getAgentChatStatus,
  getAgentBriefing,
  getAgentDutyReport,
  getAgentReview,
  getAgentReviewHistory,
  getAgentState,
  getCandidateProfile,
  getJobDetail,
  getSettings,
  importURL,
  listAgentEvents,
  listAgentChatMessages,
  listApplicationPlans,
  listAgentTasks,
  listCompanies,
  listJobs,
  listRunSources,
  listRuns,
  listSourceCandidates,
  listSources,
  refreshAgentTasks,
  runAutomationDutyReport,
  runAgentChat,
  runCrawl,
  runAgentCommand,
  runRecommendedCrawl,
  runSourceDiscovery,
  saveAgentReviewSnapshot,
  sendFeishuReport,
  sendFeishuTest,
  seedRecommendedSources,
  syncApplicationPlans,
  updateAgentTaskStatus,
  updateApplicationPlan,
  updateCandidateProfile,
  updateJobNotes,
  updateJobStatus,
  updateCompanyEnabled,
  updateSettings,
  updateSourceEnabled,
  validateSourceCandidate,
  rejectSourceCandidate,
} from "./api";
import { DigitalEmployee3D } from "./DigitalEmployee3D";
import type { AgentAutomationDiagnostics, AgentBriefing, AgentChatMessage, AgentChatStatus, AgentCommandResult, AgentDutyReport, AgentEvent, AgentReview, AgentReviewHistory, AgentState, AgentTask, ApplicationPlan, CandidateProfile, Company, Job, JobDetail, JobRun, JobRunSource, JobStatus, RunSummary, Settings, Source, SourceCandidate } from "./types";

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

type AppView = "dashboard" | "opportunities" | "applications" | "profile" | "companies" | "runs" | "settings";

const appViews: Array<{ id: AppView; label: string }> = [
  { id: "dashboard", label: "Dashboard" },
  { id: "opportunities", label: "Opportunities" },
  { id: "applications", label: "Applications" },
  { id: "profile", label: "Profile" },
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

type CandidateProfileDraft = {
  target_cities: string;
  target_directions: string;
  skills: string;
  education: string;
  graduation_year: string;
  internship_preference: string;
  preferred_companies: string;
  blocked_keywords: string;
  notes: string;
};

const defaultSettings: Settings = {
  target_cities: ["Shenzhen"],
  target_directions: ["frontend", "backend", "java", "go", "algorithm", "ai_application"],
  excluded_keywords: ["outsourcing", "training", "bootcamp"],
  crawl_schedule: ["09:00", "12:00", "18:00"],
  feishu_webhook_url: "",
  feishu_configured: false,
  time_zone: "Asia/Shanghai",
  auto_duty_report_enabled: false,
  auto_source_discovery_enabled: true,
  source_discovery_interval_hours: 24,
  duty_report_time: "18:00",
  task_sla_hours: 24,
  updated_at: "",
};

const defaultProfile: CandidateProfile = {
  id: 1,
  target_cities: ["Shenzhen"],
  target_directions: ["frontend", "backend", "java", "go", "algorithm", "ai_application"],
  skills: ["Go", "Java", "React", "TypeScript", "Algorithm", "LLM"],
  education: "",
  graduation_year: "",
  internship_preference: "accept_conversion_clear",
  preferred_companies: [],
  blocked_keywords: ["outsourcing", "training", "bootcamp", "外包", "培训"],
  notes: "",
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
  const [sourceCandidates, setSourceCandidates] = useState<SourceCandidate[]>([]);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [runs, setRuns] = useState<JobRun[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<number | null>(null);
  const [runSources, setRunSources] = useState<JobRunSource[]>([]);
  const [sourceURLValue, setSourceURLValue] = useState("");
  const [companyCategoryFilter, setCompanyCategoryFilter] = useState("all");
  const [companyQuery, setCompanyQuery] = useState("");
  const [addingSource, setAddingSource] = useState(false);
  const [seedingSources, setSeedingSources] = useState(false);
  const [discoveringSources, setDiscoveringSources] = useState(false);
  const [validatingSourceCandidateId, setValidatingSourceCandidateId] = useState<number | null>(null);
  const [recommendedRunning, setRecommendedRunning] = useState(false);
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [settingsDraft, setSettingsDraft] = useState(settingsToDraft(defaultSettings));
  const [profile, setProfile] = useState<CandidateProfile>(defaultProfile);
  const [profileDraft, setProfileDraft] = useState(profileToDraft(defaultProfile));
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingSettings, setSavingSettings] = useState(false);
  const [testingFeishu, setTestingFeishu] = useState(false);
  const [sendingFeishuReport, setSendingFeishuReport] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [lastRun, setLastRun] = useState<RunSummary | null>(null);
  const [briefing, setBriefing] = useState<AgentBriefing | null>(null);
  const [agentState, setAgentState] = useState<AgentState | null>(null);
  const [dutyReport, setDutyReport] = useState<AgentDutyReport | null>(null);
  const [agentReview, setAgentReview] = useState<AgentReview | null>(null);
  const [agentReviewHistory, setAgentReviewHistory] = useState<AgentReviewHistory | null>(null);
  const [agentEvents, setAgentEvents] = useState<AgentEvent[]>([]);
  const [agentTasks, setAgentTasks] = useState<AgentTask[]>([]);
  const [applicationPlans, setApplicationPlans] = useState<ApplicationPlan[]>([]);
  const [automationStatus, setAutomationStatus] = useState<AgentAutomationDiagnostics | null>(null);
  const [chatStatus, setChatStatus] = useState<AgentChatStatus | null>(null);
  const [chatMessages, setChatMessages] = useState<AgentChatMessage[]>([]);
  const [chatActions, setChatActions] = useState<AgentCommandResult["actions"]>([]);
  const [chatText, setChatText] = useState("");
  const [chatOpen, setChatOpen] = useState(true);
  const [chatSending, setChatSending] = useState(false);
  const [selectedJobDetail, setSelectedJobDetail] = useState<JobDetail | null>(null);
  const [loadingJobDetail, setLoadingJobDetail] = useState(false);
  const [refreshingTasks, setRefreshingTasks] = useState(false);
  const [syncingApplications, setSyncingApplications] = useState(false);
  const [commandText, setCommandText] = useState("");
  const [commandResult, setCommandResult] = useState<AgentCommandResult | null>(null);
  const [runningCommand, setRunningCommand] = useState(false);
  const [savingReviewSnapshot, setSavingReviewSnapshot] = useState(false);

  async function refresh(nextStatus = status) {
    setError("");
    const data = await listJobs(nextStatus);
    setJobs(data);
  }

  async function refreshSources() {
    const data = await listSources();
    setSources(data);
  }

  async function refreshSourceCandidates() {
    const data = await listSourceCandidates();
    setSourceCandidates(data);
  }

  async function refreshCompanies() {
    const data = await listCompanies();
    setCompanies(data);
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

  async function refreshProfile() {
    const data = await getCandidateProfile();
    const nextProfile = normalizeProfile(data);
    setProfile(nextProfile);
    setProfileDraft(profileToDraft(nextProfile));
  }

  async function refreshBriefing() {
    const data = await getAgentBriefing();
    setBriefing(data);
  }

  async function refreshAgentState() {
    const data = await getAgentState();
    setAgentState(data);
  }

  async function refreshDutyReport() {
    const data = await getAgentDutyReport();
    setDutyReport(data);
  }

  async function refreshAgentReview() {
    const data = await getAgentReview();
    setAgentReview(data);
  }

  async function refreshAgentReviewHistory() {
    const data = await getAgentReviewHistory();
    setAgentReviewHistory(data);
  }

  async function refreshAgentEvents() {
    const data = await listAgentEvents();
    setAgentEvents(data);
  }

  async function refreshChat() {
    const [status, messages] = await Promise.all([getAgentChatStatus(), listAgentChatMessages()]);
    setChatStatus(status);
    setChatMessages(messages);
  }

  async function refreshTasks() {
    const data = await listAgentTasks();
    setAgentTasks(data);
  }

  async function refreshApplicationPlans() {
    const data = await listApplicationPlans();
    setApplicationPlans(data);
  }

  async function refreshAutomationStatus() {
    const data = await getAutomationStatus();
    setAutomationStatus(data);
  }

  useEffect(() => {
    Promise.all([refresh(), refreshSources(), refreshSourceCandidates(), refreshCompanies(), refreshRuns(), refreshSettings(), refreshProfile(), refreshBriefing(), refreshAgentState(), refreshDutyReport(), refreshAgentReview(), refreshAgentReviewHistory(), refreshAgentEvents(), refreshTasks(), refreshApplicationPlans(), refreshAutomationStatus(), refreshChat()])
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
  const enabledCompanies = companies.filter((company) => company.enabled).length;
  const companyCategories = useMemo(() => {
    const categories = new Set<string>();
    companies.forEach((company) => categories.add(company.category || "general"));
    sources.forEach((source) => categories.add(source.category || "general"));
    return ["all", ...Array.from(categories).sort()];
  }, [companies, sources]);
  const visibleCompanies = useMemo(() => {
    const query = companyQuery.trim().toLowerCase();
    return companies.filter((company) => {
      const category = company.category || "general";
      const categoryMatches = companyCategoryFilter === "all" || category === companyCategoryFilter;
      const queryMatches = query === "" || company.name.toLowerCase().includes(query) || category.toLowerCase().includes(query);
      return categoryMatches && queryMatches;
    });
  }, [companies, companyCategoryFilter, companyQuery]);
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
  const readinessItems = [
    {
      id: "company_scope",
      label: "Company scope",
      detail: companies.length > 0 ? `${enabledCompanies} enabled companies` : "No company pool yet",
      done: companies.length > 0 && enabledCompanies > 0,
      actionLabel: companies.length > 0 ? "Manage" : "Add companies",
      action: () => setActiveView("companies"),
    },
    {
      id: "preferences",
      label: "Preferences",
      detail: `${settings.target_cities.join(", ")} / ${settings.target_directions.length} directions`,
      done: settings.target_cities.length > 0 && settings.target_directions.length > 0,
      actionLabel: "Edit",
      action: () => setActiveView("settings"),
    },
    {
      id: "candidate_profile",
      label: "Candidate profile",
      detail: `${profile.skills.length} skills / ${profile.preferred_companies.length} preferred companies`,
      done: profile.skills.length > 0 && profile.target_directions.length > 0,
      actionLabel: "Profile",
      action: () => setActiveView("profile"),
    },
    {
      id: "crawl_history",
      label: "Crawl history",
      detail: runs.length > 0 ? `${runs.length} recorded runs` : "No crawl run yet",
      done: runs.length > 0,
      actionLabel: runs.length > 0 ? "View runs" : "Run crawl",
      action: runs.length > 0 ? () => setActiveView("runs") : handleRunCrawl,
    },
    {
      id: "feishu",
      label: "Feishu",
      detail: settings.feishu_configured ? "Webhook configured" : "Webhook not configured",
      done: settings.feishu_configured,
      actionLabel: "Settings",
      action: () => setActiveView("settings"),
    },
  ];

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
      case "refresh_tasks":
        await handleRefreshAgentTasks();
        return;
      case "sync_application_plans":
      case "prepare_application":
        setActiveView("applications");
        await handleSyncApplicationPlans();
        return;
      case "follow_up_application":
        setActiveView("applications");
        setNotice("Opened application follow-up workspace.");
        return;
      case "discover_sources":
        setActiveView("companies");
        await handleRunSourceDiscovery();
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
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
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
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
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
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
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
      await refreshCompanies();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
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
    await refreshAgentReview();
      await refreshAgentReviewHistory();
    await refreshTasks();
    await refreshAgentState();
  }

  async function toggleCompany(company: Company) {
    await updateCompanyEnabled(company.id, !company.enabled);
    setCompanies((current) => current.map((item) => (item.id === company.id ? { ...item, enabled: !company.enabled } : item)));
    await refreshSources();
    await refreshBriefing();
    await refreshDutyReport();
    await refreshAgentReview();
      await refreshAgentReviewHistory();
    await refreshAgentEvents();
    await refreshTasks();
    await refreshApplicationPlans();
    await refreshAgentState();
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
      await refreshCompanies();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not add recommended sources");
    } finally {
      setSeedingSources(false);
    }
  }

  async function handleRunSourceDiscovery() {
    setDiscoveringSources(true);
    setError("");
    setNotice("");
    try {
      const result = await runSourceDiscovery(settings.target_cities, settings.target_directions);
      setNotice(`Source discovery finished. Proposed ${result.created} new candidates and skipped ${result.duplicated} duplicates.`);
      await refreshSourceCandidates();
      await refreshAgentEvents();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not discover source candidates");
    } finally {
      setDiscoveringSources(false);
    }
  }

  async function handleAcceptSourceCandidate(candidate: SourceCandidate) {
    setError("");
    setNotice("");
    try {
      await acceptSourceCandidate(candidate.id);
      setNotice(`${candidate.name} accepted into active sources.`);
      await refreshSourceCandidates();
      await refreshSources();
      await refreshCompanies();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not accept source candidate");
    }
  }

  async function handleRejectSourceCandidate(candidate: SourceCandidate) {
    setError("");
    setNotice("");
    try {
      await rejectSourceCandidate(candidate.id);
      setNotice(`${candidate.name} rejected.`);
      await refreshSourceCandidates();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not reject source candidate");
    }
  }

  async function handleValidateSourceCandidate(candidate: SourceCandidate) {
    setValidatingSourceCandidateId(candidate.id);
    setError("");
    setNotice("");
    try {
      const validated = await validateSourceCandidate(candidate.id);
      setNotice(`${candidate.name} checked: ${validated.validation_status}.`);
      await refreshSourceCandidates();
      await refreshAgentEvents();
      await refreshAgentState();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not validate source candidate");
    } finally {
      setValidatingSourceCandidateId(null);
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
      await refreshCompanies();
      await refresh();
      await refreshRuns();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
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
        feishu_webhook_url: settingsDraft.feishu_webhook_url.trim(),
        time_zone: settingsDraft.time_zone.trim() || defaultSettings.time_zone,
        auto_duty_report_enabled: settingsDraft.auto_duty_report_enabled,
        auto_source_discovery_enabled: settingsDraft.auto_source_discovery_enabled,
        source_discovery_interval_hours: Number(settingsDraft.source_discovery_interval_hours) || defaultSettings.source_discovery_interval_hours,
        duty_report_time: settingsDraft.duty_report_time.trim(),
        task_sla_hours: Number(settingsDraft.task_sla_hours) || defaultSettings.task_sla_hours,
      });
      const nextSettings = normalizeSettings(saved);
      setSettings(nextSettings);
      setSettingsDraft(settingsToDraft(nextSettings));
      setNotice("Settings saved. Future crawl and scoring steps can use these preferences.");
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshTasks();
      await refreshAgentState();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save settings");
    } finally {
      setSavingSettings(false);
    }
  }

  async function handleSaveProfile(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingProfile(true);
    setError("");
    setNotice("");
    try {
      const saved = await updateCandidateProfile({
        target_cities: parseSettingsList(profileDraft.target_cities),
        target_directions: parseSettingsList(profileDraft.target_directions),
        skills: parseSettingsList(profileDraft.skills),
        education: profileDraft.education.trim(),
        graduation_year: profileDraft.graduation_year.trim(),
        internship_preference: profileDraft.internship_preference.trim(),
        preferred_companies: parseSettingsList(profileDraft.preferred_companies),
        blocked_keywords: parseSettingsList(profileDraft.blocked_keywords),
        notes: profileDraft.notes.trim(),
      });
      const nextProfile = normalizeProfile(saved);
      setProfile(nextProfile);
      setProfileDraft(profileToDraft(nextProfile));
      setNotice("Candidate profile saved. Job detail fit signals will use it immediately.");
      if (selectedJobDetail) {
        await handleOpenJobDetail(selectedJobDetail.job.id);
      }
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save candidate profile");
    } finally {
      setSavingProfile(false);
    }
  }

  async function handleSendFeishuTest() {
    setTestingFeishu(true);
    setError("");
    setNotice("");
    try {
      await sendFeishuTest();
      setNotice("Feishu test notification sent.");
      await refreshSettings();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not send Feishu test notification");
    } finally {
      setTestingFeishu(false);
    }
  }

  async function handleSendFeishuReport() {
    setSendingFeishuReport(true);
    setError("");
    setNotice("");
    try {
      await sendFeishuReport();
      setNotice("Feishu duty report sent.");
      await refreshAgentEvents();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not send Feishu duty report");
    } finally {
      setSendingFeishuReport(false);
    }
  }

  async function handleRunAutomationDutyReport() {
    setSendingFeishuReport(true);
    setError("");
    setNotice("");
    try {
      await runAutomationDutyReport();
      setNotice("Automatic duty report sent.");
      await refreshSettings();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshAgentState();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not run automatic duty report");
    } finally {
      setSendingFeishuReport(false);
    }
  }

  async function handleRefreshAgentTasks() {
    setRefreshingTasks(true);
    setError("");
    setNotice("");
    try {
      const tasks = await refreshAgentTasks();
      setAgentTasks(tasks);
      setNotice("Daily tasks refreshed from the current recruiting pipeline.");
      await refreshApplicationPlans();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshAgentState();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not refresh daily tasks");
    } finally {
      setRefreshingTasks(false);
    }
  }

  async function handleSyncApplicationPlans() {
    setSyncingApplications(true);
    setError("");
    setNotice("");
    try {
      const plans = await syncApplicationPlans();
      setApplicationPlans(plans);
      await refreshTasks();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshAgentState();
      setNotice(`Application workspace synced with ${plans.length} plans.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not sync application plans");
    } finally {
      setSyncingApplications(false);
    }
  }

  async function handleApplicationPlanStatus(plan: ApplicationPlan, status: ApplicationPlan["status"]) {
    setError("");
    setNotice("");
    try {
      await updateApplicationPlan(plan.id, { ...plan, status });
      await refreshApplicationPlans();
      await refreshTasks();
      await refreshDutyReport();
      await refreshAgentEvents();
      await refreshAgentState();
      setNotice(`Application plan updated to ${status}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update application plan");
    }
  }

  async function handleTaskDone(task: AgentTask) {
    await updateAgentTaskStatus(task.id, "done", { completion_reason: "Completed from dashboard" });
    setNotice("Task completed.");
    await refreshAfterTaskMutation();
  }

  async function handleTaskSnooze(task: AgentTask) {
    const snoozedUntil = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    await updateAgentTaskStatus(task.id, "snoozed", { snoozed_until: snoozedUntil });
    setNotice("Task snoozed for 24 hours.");
    await refreshAfterTaskMutation();
  }

  async function handleTaskIgnore(task: AgentTask) {
    await updateAgentTaskStatus(task.id, "done", { completion_reason: "Ignored from dashboard" });
    setNotice("Task ignored.");
    await refreshAfterTaskMutation();
  }

  async function refreshAfterTaskMutation() {
    await refreshTasks();
    await refreshDutyReport();
    await refreshAgentReview();
    await refreshAgentReviewHistory();
    await refreshAgentEvents();
    await refreshAgentState();
    await refreshAutomationStatus();
  }

  async function handleRunAgentCommand(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = commandText.trim();
    if (!value) {
      setError("Type a command for the agent first.");
      return;
    }
    setRunningCommand(true);
    setError("");
    setNotice("");
    try {
      const result = await runAgentCommand(value);
      setCommandResult(result);
      setCommandText("");
      setNotice(result.summary || "Command processed.");
      await refreshSettings();
      await refresh();
      await refreshSources();
      await refreshCompanies();
      await refreshRuns();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not run command");
    } finally {
      setRunningCommand(false);
    }
  }

  async function handleSendChat(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = chatText.trim();
    if (!value) {
      return;
    }
    const optimistic: AgentChatMessage = {
      id: Date.now(),
      role: "user",
      content: value,
      source: "user",
      created_at: new Date().toISOString(),
    };
    setChatMessages((current) => [...current, optimistic]);
    setChatText("");
    setChatActions([]);
    setChatSending(true);
    setError("");
    try {
      const response = await runAgentChat(value, activeView);
      setChatActions(response.reply.actions || []);
      await refreshChat();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not talk to the agent");
    } finally {
      setChatSending(false);
    }
  }

  async function selectRun(runId: number) {
    setSelectedRunId(runId);
    setRunSources(await listRunSources(runId));
  }

  async function handleOpenJobDetail(id: number) {
    setLoadingJobDetail(true);
    setError("");
    try {
      const detail = await getJobDetail(id);
      setSelectedJobDetail(detail);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not load job detail");
    } finally {
      setLoadingJobDetail(false);
    }
  }

  async function handleSaveJobNotes(job: Job, notes: string) {
    await updateJobNotes(job.id, notes);
    await refresh();
    await handleOpenJobDetail(job.id);
    await refreshAgentEvents();
    setNotice("Job notes saved.");
  }

  async function handleSaveReviewSnapshot() {
    setSavingReviewSnapshot(true);
    setError("");
    setNotice("");
    try {
      await saveAgentReviewSnapshot("manual");
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      setNotice("Review snapshot saved for trend comparison.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save review snapshot");
    } finally {
      setSavingReviewSnapshot(false);
    }
  }

  async function setJobStatus(id: number, next: JobStatus) {
    await updateJobStatus(id, next);
    setJobs((current) => current.map((job) => (job.id === id ? { ...job, status: next } : job)));
    if (selectedJobDetail?.job.id === id) {
      await handleOpenJobDetail(id);
    }
    await refreshBriefing();
    await refreshDutyReport();
    await refreshAgentReview();
    await refreshAgentReviewHistory();
    await refreshAgentEvents();
    await refreshTasks();
      await refreshAgentState();
      await refreshAutomationStatus();
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
        <section className="dashboard-workbench">
          <div className="dashboard-main">
            <section className="summary-grid">
              <Metric label="Tracked jobs" value={jobs.length} />
              <Metric label="Strong matches" value={strongMatches} />
              <Metric label="Enabled companies" value={enabledCompanies} />
              <Metric label="Next runs" value={settings.crawl_schedule.join(" / ")} />
            </section>

            <ProductReadinessPanel items={readinessItems} busy={running || seedingSources || recommendedRunning} />

            {agentReview && (
              <AgentReviewPanel
                review={agentReview}
                history={agentReviewHistory}
                onAction={handleAgentAction}
                onSaveSnapshot={handleSaveReviewSnapshot}
                busy={running || recommendedRunning}
                savingSnapshot={savingReviewSnapshot}
              />
            )}

            <AgentTasksPanel
              tasks={agentTasks}
              onAction={handleAgentAction}
              onComplete={handleTaskDone}
              onSnooze={handleTaskSnooze}
              onIgnore={handleTaskIgnore}
              onRefresh={handleRefreshAgentTasks}
              refreshing={refreshingTasks}
              busy={running || recommendedRunning}
            />

            {briefing && <AgentBriefingPanel briefing={briefing} onAction={handleAgentAction} busy={running || recommendedRunning} />}

            {dutyReport && (
              <AgentDutyReportPanel
                report={dutyReport}
                onAction={handleAgentAction}
                onSendFeishu={handleSendFeishuReport}
                busy={running || recommendedRunning}
                sendingFeishu={sendingFeishuReport}
                feishuReady={settings.feishu_configured}
              />
            )}

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
          </div>
          {agentState && (
            <AgentEmployeeSidebar
              state={agentState}
              onRefreshTasks={handleRefreshAgentTasks}
              onSendFeishu={handleSendFeishuReport}
              refreshingTasks={refreshingTasks}
              sendingFeishu={sendingFeishuReport}
              feishuReady={settings.feishu_configured}
              onRunAutomationDutyReport={handleRunAutomationDutyReport}
              commandText={commandText}
              commandResult={commandResult}
              runningCommand={runningCommand}
              onCommandTextChange={setCommandText}
              onRunCommand={handleRunAgentCommand}
            />
          )}
        </section>
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
                        <small>{job.recommend_reasons.slice(0, 2).join(" / ") || "No reasons yet"}</small>
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
                        <button onClick={() => handleOpenJobDetail(job.id)} disabled={loadingJobDetail}>
                          Details
                        </button>
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
          {selectedJobDetail && (
            <JobDetailPanel
              detail={selectedJobDetail}
              busy={loadingJobDetail}
              onClose={() => setSelectedJobDetail(null)}
              onStatus={setJobStatus}
              onSaveNotes={handleSaveJobNotes}
            />
          )}
        </>
      )}

      {activeView === "applications" && (
        <ApplicationWorkspace
          plans={applicationPlans}
          jobs={jobs}
          syncing={syncingApplications}
          onSync={handleSyncApplicationPlans}
          onStatus={handleApplicationPlanStatus}
          onOpenJob={handleOpenJobDetail}
        />
      )}

      {activeView === "profile" && (
        <ProfilePanel
          profile={profile}
          draft={profileDraft}
          saving={savingProfile}
          onDraftChange={setProfileDraft}
          onSubmit={handleSaveProfile}
        />
      )}

      {activeView === "companies" && (
      <section className="sources-panel">
        <div className="panel-header">
          <h2>Companies</h2>
          <span>{enabledCompanies} enabled / {companies.length} total</span>
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
        <div className="company-grid">
          {visibleCompanies.map((company) => (
            <div className="company-card" key={company.id}>
              <div>
                <strong>{company.name}</strong>
                <div className="source-meta">
                  <span>{categoryLabels[company.category] || company.category || "General"}</span>
                  <span>{company.source_count} sources</span>
                  {company.broken_count > 0 && <span>{company.broken_count} broken</span>}
                  {company.warning_count > 0 && <span>{company.warning_count} warning</span>}
                </div>
              </div>
              <button className={company.enabled ? "toggle-on" : "toggle-off"} onClick={() => toggleCompany(company)}>
                {company.enabled ? "Enabled" : "Disabled"}
              </button>
            </div>
          ))}
          {visibleCompanies.length === 0 && <div className="empty-source">No companies match the current filters.</div>}
        </div>
        <div className="source-actions">
          <button type="button" onClick={handleSeedRecommendedSources} disabled={seedingSources || recommendedRunning}>
            {seedingSources ? "Adding..." : "Add Recommended"}
          </button>
          <button type="button" onClick={handleRunSourceDiscovery} disabled={discoveringSources || recommendedRunning}>
            {discoveringSources ? "Discovering..." : "Discover Sources"}
          </button>
          <button type="button" className="strong-action" onClick={handleRecommendedCrawl} disabled={recommendedRunning || seedingSources}>
            {recommendedRunning ? "Running..." : "Add & Crawl"}
          </button>
        </div>
        <SourceCandidatesPanel
          candidates={sourceCandidates}
          onAccept={handleAcceptSourceCandidate}
          onReject={handleRejectSourceCandidate}
          onValidate={handleValidateSourceCandidate}
          busy={discoveringSources || recommendedRunning}
          validatingId={validatingSourceCandidateId}
        />
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
          {visibleSources.length === 0 && <div className="empty-source">No source entries match the current filters.</div>}
        </div>
      </section>
      )}

      {activeView === "settings" && (
      <section className="settings-panel">
        <div className="panel-header">
          <h2>Settings</h2>
          <span>{settings.feishu_configured ? "Feishu ready" : "Feishu not configured"}</span>
        </div>
        {automationStatus && (
          <div className={automationStatus.ready_for_automatic_report ? "automation-diagnostic ready" : "automation-diagnostic"}>
            <div>
              <strong>{automationStatus.ready_for_automatic_report ? "Automatic report ready" : "Automatic report needs setup"}</strong>
              <span>{automationStatus.reason}</span>
            </div>
            <div className="automation-diagnostic-grid">
              <span>Scheduler {automationStatus.scheduler_expected ? "expected" : "not expected"}</span>
              <span>Webhook {automationStatus.webhook_configured ? "configured" : "missing"}</span>
              <span>Duty report {automationStatus.duty_report_enabled ? "enabled" : "disabled"}</span>
              <span>{automationStatus.time_zone} / {automationStatus.duty_report_time}</span>
              <span>Next {formatDateTime(automationStatus.next_duty_report_at)}</span>
              <span>{automationStatus.last_duty_report_sent_at ? `Last ${formatDateTime(automationStatus.last_duty_report_sent_at)}` : "No automatic report sent yet"}</span>
            </div>
          </div>
        )}
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
          <label>
            Time zone
            <input
              value={settingsDraft.time_zone}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, time_zone: event.target.value }))}
              placeholder="Asia/Shanghai"
            />
          </label>
          <label className="settings-wide">
            Feishu bot webhook
            <input
              value={settingsDraft.feishu_webhook_url}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, feishu_webhook_url: event.target.value }))}
              placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
            />
          </label>
          <label className="settings-toggle">
            <input
              type="checkbox"
              checked={settingsDraft.auto_duty_report_enabled}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, auto_duty_report_enabled: event.target.checked }))}
            />
            Automatic duty report
          </label>
          <label className="settings-toggle">
            <input
              type="checkbox"
              checked={settingsDraft.auto_source_discovery_enabled}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, auto_source_discovery_enabled: event.target.checked }))}
            />
            Automatic source discovery
          </label>
          <label>
            Duty report time
            <input
              value={settingsDraft.duty_report_time}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, duty_report_time: event.target.value }))}
              placeholder="18:00"
            />
          </label>
          <label>
            Discovery interval hours
            <input
              type="number"
              min="1"
              value={settingsDraft.source_discovery_interval_hours}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, source_discovery_interval_hours: event.target.value }))}
            />
          </label>
          <label>
            Task SLA hours
            <input
              type="number"
              min="1"
              value={settingsDraft.task_sla_hours}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, task_sla_hours: event.target.value }))}
            />
          </label>
          <button type="submit" disabled={savingSettings}>
            {savingSettings ? "Saving..." : "Save Settings"}
          </button>
          <button type="button" className="secondary-settings-action" onClick={handleSendFeishuTest} disabled={testingFeishu || !settings.feishu_configured}>
            {testingFeishu ? "Sending..." : "Send Feishu Test"}
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

      <GlobalEmployeeChat
        state={agentState}
        status={chatStatus}
        messages={chatMessages}
        text={chatText}
        open={chatOpen}
        sending={chatSending}
        activeView={activeView}
        onToggle={() => setChatOpen((current) => !current)}
        onTextChange={setChatText}
        onSubmit={handleSendChat}
        actions={chatActions}
        onAction={handleAgentAction}
      />
    </main>
  );
}

function ProfilePanel({
  profile,
  draft,
  saving,
  onDraftChange,
  onSubmit,
}: {
  profile: CandidateProfile;
  draft: CandidateProfileDraft;
  saving: boolean;
  onDraftChange: React.Dispatch<React.SetStateAction<CandidateProfileDraft>>;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
}) {
  return (
    <section className="profile-panel">
      <div className="panel-header">
        <div>
          <h2>Candidate Profile</h2>
          <span>Updated {formatDateTime(profile.updated_at)}</span>
        </div>
      </div>
      <form className="profile-grid" onSubmit={onSubmit}>
        <label>
          Target cities
          <textarea value={draft.target_cities} onChange={(event) => onDraftChange((current) => ({ ...current, target_cities: event.target.value }))} />
        </label>
        <label>
          Target directions
          <textarea value={draft.target_directions} onChange={(event) => onDraftChange((current) => ({ ...current, target_directions: event.target.value }))} />
        </label>
        <label>
          Skills
          <textarea value={draft.skills} onChange={(event) => onDraftChange((current) => ({ ...current, skills: event.target.value }))} />
        </label>
        <label>
          Preferred companies
          <textarea value={draft.preferred_companies} onChange={(event) => onDraftChange((current) => ({ ...current, preferred_companies: event.target.value }))} />
        </label>
        <label>
          Education
          <input value={draft.education} onChange={(event) => onDraftChange((current) => ({ ...current, education: event.target.value }))} placeholder="Bachelor / Master / Other" />
        </label>
        <label>
          Graduation year
          <input value={draft.graduation_year} onChange={(event) => onDraftChange((current) => ({ ...current, graduation_year: event.target.value }))} placeholder="2027" />
        </label>
        <label>
          Internship preference
          <input value={draft.internship_preference} onChange={(event) => onDraftChange((current) => ({ ...current, internship_preference: event.target.value }))} />
        </label>
        <label>
          Blocked keywords
          <textarea value={draft.blocked_keywords} onChange={(event) => onDraftChange((current) => ({ ...current, blocked_keywords: event.target.value }))} />
        </label>
        <label className="profile-wide">
          Notes
          <textarea value={draft.notes} onChange={(event) => onDraftChange((current) => ({ ...current, notes: event.target.value }))} placeholder="Preferred roles, teams, and deal breakers." />
        </label>
        <button type="submit" disabled={saving}>
          {saving ? "Saving..." : "Save Profile"}
        </button>
      </form>
    </section>
  );
}

function ApplicationWorkspace({
  plans,
  jobs,
  syncing,
  onSync,
  onStatus,
  onOpenJob,
}: {
  plans: ApplicationPlan[];
  jobs: Job[];
  syncing: boolean;
  onSync: () => void | Promise<void>;
  onStatus: (plan: ApplicationPlan, status: ApplicationPlan["status"]) => void | Promise<void>;
  onOpenJob: (id: number) => void | Promise<void>;
}) {
  const jobsByID = new Map(jobs.map((job) => [job.id, job]));
  const activePlans = plans.filter((plan) => plan.status !== "applied" && plan.status !== "paused");
  return (
    <section className="applications-panel">
      <div className="panel-header">
        <div>
          <h2>Application Workspace</h2>
          <span>{activePlans.length} active / {plans.length} total</span>
        </div>
        <button type="button" onClick={onSync} disabled={syncing}>
          {syncing ? "Syncing..." : "Sync Plans"}
        </button>
      </div>
      <div className="application-list">
        {plans.map((plan) => {
          const job = jobsByID.get(plan.job_id);
          return (
            <div className={`application-row application-${plan.status}`} key={plan.id}>
              <div>
                <div className="candidate-title">
                  <strong>{job ? `${job.company} / ${job.title}` : `Job #${plan.job_id}`}</strong>
                  <b>{plan.priority}</b>
                </div>
                <div className="source-meta">
                  <span>{plan.status}</span>
                  <span>{plan.target_apply_date || "No target date"}</span>
                  <span>follow {plan.follow_up_date || "not set"}</span>
                  <span>resume {plan.resume_version || "default"}</span>
                  {job?.city && <span>{job.city}</span>}
                </div>
                <p>{plan.next_action || "No next action yet."}</p>
                {plan.draft_notes && <p className="application-draft">{plan.draft_notes}</p>}
                <div className="application-checklist">
                  {plan.checklist.slice(0, 5).map((item) => (
                    <span key={item}>{item}</span>
                  ))}
                </div>
                {plan.blocker_notes && <small>{plan.blocker_notes}</small>}
              </div>
              <div className="candidate-actions">
                <button type="button" onClick={() => onOpenJob(plan.job_id)}>
                  Details
                </button>
                <button type="button" onClick={() => onStatus(plan, "ready")}>
                  Ready
                </button>
                <button type="button" onClick={() => onStatus(plan, "applied")}>
                  Applied
                </button>
                <button type="button" onClick={() => onStatus(plan, "paused")}>
                  Pause
                </button>
              </div>
            </div>
          );
        })}
        {plans.length === 0 && <div className="empty-source">No application plans yet. Mark strong jobs as Interested, then sync plans.</div>}
      </div>
    </section>
  );
}

function JobDetailPanel({
  detail,
  busy,
  onClose,
  onStatus,
  onSaveNotes,
}: {
  detail: JobDetail;
  busy: boolean;
  onClose: () => void;
  onStatus: (id: number, status: JobStatus) => void | Promise<void>;
  onSaveNotes: (job: Job, notes: string) => void | Promise<void>;
}) {
  const [notes, setNotes] = useState(detail.job.notes || "");

  useEffect(() => {
    setNotes(detail.job.notes || "");
  }, [detail.job.id, detail.job.notes]);

  return (
    <section className="job-detail-panel">
      <div className="panel-header">
        <div>
          <h2>{detail.job.company} / {detail.job.title}</h2>
          <span>{detail.job.city || "Unknown city"} / {formatFitVerdict(detail.fit.verdict)}</span>
        </div>
        <button type="button" className="secondary-detail-action" onClick={onClose}>
          Close
        </button>
      </div>
      <div className="job-detail-grid">
        <div className="fit-card">
          <span>Profile fit</span>
          <strong>{detail.fit.score}</strong>
          <small>{detail.suggested_action.reason}</small>
          <div className="job-detail-actions">
            <button type="button" onClick={() => onStatus(detail.job.id, "interested")} disabled={busy}>
              Interested
            </button>
            <button type="button" onClick={() => onStatus(detail.job.id, "applied")} disabled={busy}>
              Applied
            </button>
            <button type="button" onClick={() => onStatus(detail.job.id, "ignored")} disabled={busy}>
              Ignore
            </button>
          </div>
        </div>
        <div className="detail-column">
          <h3>Why It Fits</h3>
          {[...detail.fit.profile_signals, ...detail.fit.strengths].slice(0, 8).map((item) => (
            <span className="detail-pill" key={item}>{item}</span>
          ))}
          {detail.fit.profile_signals.length === 0 && detail.fit.strengths.length === 0 && <div className="empty-source">No fit signals yet.</div>}
        </div>
        <div className="detail-column">
          <h3>Risks</h3>
          {detail.fit.risks.slice(0, 8).map((item) => (
            <span className="risk-pill" key={item}>{item}</span>
          ))}
          {detail.fit.risks.length === 0 && <div className="empty-source">No obvious risk signals.</div>}
        </div>
        <div className="detail-column">
          <h3>Application Plan</h3>
          {detail.application_plan ? (
            <>
              <span className="detail-pill">{detail.application_plan.status}</span>
              <span className="detail-pill">{detail.application_plan.target_apply_date || "No target date"}</span>
              <small>{detail.application_plan.next_action}</small>
              {detail.application_plan.checklist.slice(0, 4).map((item) => (
                <span className="detail-pill" key={item}>{item}</span>
              ))}
            </>
          ) : (
            <div className="empty-source">Mark this job as Interested and refresh tasks to create a plan.</div>
          )}
        </div>
      </div>
      <div className="job-detail-bottom">
        <form className="notes-box" onSubmit={(event) => { event.preventDefault(); onSaveNotes(detail.job, notes); }}>
          <label>
            Decision notes
            <textarea value={notes} onChange={(event) => setNotes(event.target.value)} />
          </label>
          <button type="submit" disabled={busy}>
            Save Notes
          </button>
        </form>
        <div className="decision-timeline">
          <h3>Decision History</h3>
          {detail.decisions.map((decision) => (
            <div className="decision-row" key={decision.id}>
              <strong>{formatDecisionAction(decision)}</strong>
              <span>{decision.notes || decision.reason || `${decision.from_status || "unknown"} -> ${decision.to_status || "unknown"}`}</span>
              <time>{formatDateTime(decision.created_at)}</time>
            </div>
          ))}
          {detail.decisions.length === 0 && <div className="empty-source">No decision history yet.</div>}
        </div>
      </div>
      <div className="detail-links">
        {detail.job.apply_url && <a href={detail.job.apply_url} target="_blank" rel="noreferrer">Apply URL</a>}
        {detail.job.source_url && <a href={detail.job.source_url} target="_blank" rel="noreferrer">Source URL</a>}
      </div>
    </section>
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
  onSendFeishu,
  busy,
  sendingFeishu,
  feishuReady,
}: {
  report: AgentDutyReport;
  onAction: (action: string) => void | Promise<void>;
  onSendFeishu: () => void | Promise<void>;
  busy: boolean;
  sendingFeishu: boolean;
  feishuReady: boolean;
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
        <div className="duty-actions">
          <button type="button" onClick={() => onAction(report.next_best_action.action)} disabled={busy}>
            {report.next_best_action.label}
          </button>
          <button type="button" className="secondary-duty-action" onClick={onSendFeishu} disabled={sendingFeishu || !feishuReady}>
            {sendingFeishu ? "Sending..." : "Send to Feishu"}
          </button>
        </div>
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
              <strong>{item.company} / {item.job_title}</strong>
              <span>{item.city} / score {item.score}</span>
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
              <span>{sourceHealthLabels[issue.status] || issue.status} / {issue.reason}</span>
              <small>found {issue.last_found_count} / failures {issue.consecutive_failures}</small>
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
        <span>{report.summary.open_tasks} open tasks</span>
        <span>{report.summary.stale_tasks} stale</span>
        <span>{report.summary.escalated_tasks} escalated</span>
        <span>{report.summary.done_tasks} done</span>
      </div>
      {report.trend_summary && <div className="duty-trend">{report.trend_summary}</div>}
    </section>
  );
}

function AgentReviewPanel({
  review,
  history,
  onAction,
  onSaveSnapshot,
  busy,
  savingSnapshot,
}: {
  review: AgentReview;
  history: AgentReviewHistory | null;
  onAction: (action: string) => void | Promise<void>;
  onSaveSnapshot: () => void | Promise<void>;
  busy: boolean;
  savingSnapshot: boolean;
}) {
  const topFindings = review.findings.slice(0, 4);
  const decisions = review.decisions.slice(0, 2);
  const recentSnapshots = history?.snapshots.slice(0, 3) || [];
  return (
    <section className={`agent-review review-${review.health.label.toLowerCase().replace(/\s+/g, "-")}`}>
      <div className="review-lead">
        <div className="review-score">
          <strong>{review.health.score}</strong>
          <span>{review.health.label}</span>
        </div>
        <div>
          <h2>{review.focus.title}</h2>
          <p>{review.focus.detail}</p>
        </div>
        <button type="button" onClick={() => onAction(review.focus.action)} disabled={busy}>
          Take Action
        </button>
        <button className="secondary-review-action" type="button" onClick={onSaveSnapshot} disabled={busy || savingSnapshot}>
          {savingSnapshot ? "Saving..." : "Save Snapshot"}
        </button>
      </div>
      <div className="review-trend">
        <div>
          <strong>Trend Review</strong>
          <span>{history?.summary || "No trend memory yet. Save a snapshot after meaningful changes."}</span>
        </div>
        <div className="trend-metrics">
          <TrendMetric label="Jobs" value={history?.delta.tracked_jobs || 0} />
          <TrendMetric label="Strong" value={history?.delta.strong_matches || 0} />
          <TrendMetric label="Manual" value={history?.delta.manual_decisions || 0} />
          <TrendMetric label="Sources" value={history?.delta.source_issues || 0} inverse />
          <TrendMetric label="Tasks" value={history?.delta.open_tasks || 0} inverse />
          <TrendMetric label="Applied" value={history?.delta.applied_jobs || 0} />
        </div>
      </div>
      <div className="review-body">
        <div className="review-column">
          <h3>Findings</h3>
          {topFindings.map((finding) => (
            <div className={`review-finding finding-${finding.level}`} key={`${finding.kind}-${finding.title}`}>
              <div>
                <strong>{finding.title}</strong>
                <span>{finding.detail}</span>
              </div>
              <b>{finding.metric}</b>
            </div>
          ))}
        </div>
        <div className="review-column">
          <h3>Needs Decision</h3>
          {decisions.map((decision) => (
            <div className="review-decision" key={decision.question}>
              <strong>{decision.question}</strong>
              <span>{decision.context}</span>
              <button type="button" onClick={() => onAction(decision.action)} disabled={busy}>
                Decide
              </button>
            </div>
          ))}
          {decisions.length === 0 && <div className="empty-source">No decision is blocking me right now.</div>}
        </div>
        <div className="review-column">
          <h3>Recent Memory</h3>
          {recentSnapshots.map((snapshot) => (
            <div className="review-memory" key={snapshot.id}>
              <strong>{snapshot.health_label} / {snapshot.focus_title}</strong>
              <span>{snapshot.trigger_type} / {formatDateTime(snapshot.captured_at)}</span>
              <small>{snapshot.stats.strong_matches} strong / {snapshot.stats.source_issues} source issues / {snapshot.stats.open_tasks} tasks</small>
            </div>
          ))}
          {recentSnapshots.length === 0 && <div className="empty-source">No saved snapshots yet.</div>}
          <h3 className="review-next-heading">Next Steps</h3>
          {review.next_steps.map((step) => (
            <button className="review-step" type="button" key={`${step.action}-${step.label}`} onClick={() => onAction(step.action)} disabled={busy}>
              <strong>{step.label}</strong>
              <span>{step.reason}</span>
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}

function SourceCandidatesPanel({
  candidates,
  onAccept,
  onReject,
  onValidate,
  busy,
  validatingId,
}: {
  candidates: SourceCandidate[];
  onAccept: (candidate: SourceCandidate) => void | Promise<void>;
  onReject: (candidate: SourceCandidate) => void | Promise<void>;
  onValidate: (candidate: SourceCandidate) => void | Promise<void>;
  busy: boolean;
  validatingId: number | null;
}) {
  const pending = candidates.filter((candidate) => candidate.status === "pending");
  const recent = candidates.filter((candidate) => candidate.status !== "pending").slice(0, 4);
  return (
    <section className="candidate-panel">
      <div className="panel-header">
        <h2>Source Discovery</h2>
        <span>{pending.length} pending / {candidates.length} total</span>
      </div>
      <div className="candidate-list">
        {pending.slice(0, 8).map((candidate) => (
          <div className="candidate-row" key={candidate.id}>
            <div>
              <div className="candidate-title">
                <strong>{candidate.name}</strong>
                <b>{candidate.confidence}</b>
              </div>
              <div className="source-meta">
                <span>{categoryLabels[candidate.category] || candidate.category}</span>
                <span>{candidate.parser_type || "generic"}</span>
                <span>{candidate.validation_status}</span>
              </div>
              <a href={candidate.url} target="_blank" rel="noreferrer">
                {candidate.url}
              </a>
              <small>{candidate.reason}</small>
              {candidate.validation_reason && <small>{candidate.validation_reason}</small>}
            </div>
            <div className="candidate-actions">
              <button type="button" onClick={() => onValidate(candidate)} disabled={busy || validatingId === candidate.id}>
                {validatingId === candidate.id ? "Checking..." : "Validate"}
              </button>
              <button type="button" onClick={() => onAccept(candidate)} disabled={busy}>
                Accept
              </button>
              <button type="button" onClick={() => onReject(candidate)} disabled={busy}>
                Reject
              </button>
            </div>
          </div>
        ))}
        {pending.length === 0 && <div className="empty-source">No pending candidates. Run discovery to let the agent propose new entrances.</div>}
      </div>
      {recent.length > 0 && (
        <div className="candidate-history">
          {recent.map((candidate) => (
            <span key={candidate.id}>{candidate.status}: {candidate.name}</span>
          ))}
        </div>
      )}
    </section>
  );
}

function TrendMetric({ label, value, inverse = false }: { label: string; value: number; inverse?: boolean }) {
  const status = value === 0 ? "flat" : inverse ? (value < 0 ? "good" : "bad") : value > 0 ? "good" : "bad";
  return (
    <span className={`trend-metric trend-${status}`}>
      <b>{signedDisplay(value)}</b>
      {label}
    </span>
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

function AgentTasksPanel({
  tasks,
  onAction,
  onComplete,
  onSnooze,
  onIgnore,
  onRefresh,
  refreshing,
  busy,
}: {
  tasks: AgentTask[];
  onAction: (action: string) => void | Promise<void>;
  onComplete: (task: AgentTask) => void | Promise<void>;
  onSnooze: (task: AgentTask) => void | Promise<void>;
  onIgnore: (task: AgentTask) => void | Promise<void>;
  onRefresh: () => void | Promise<void>;
  refreshing: boolean;
  busy: boolean;
}) {
  const openTasks = tasks.filter((task) => task.status !== "done");
  const doneTasks = tasks.length - openTasks.length;
  const staleTasks = tasks.filter((task) => task.status === "stale").length;
  const escalatedTasks = tasks.filter((task) => task.status === "escalated").length;
  return (
    <section className="tasks-panel">
      <div className="panel-header">
        <h2>Daily Tasks</h2>
        <span>{openTasks.length} open / {staleTasks} stale / {escalatedTasks} escalated / {doneTasks} done</span>
      </div>
      <div className="tasks-toolbar">
        <span>{tasks.length > 0 ? `Work date ${tasks[0].task_date}` : "No task queue generated yet"}</span>
        <button type="button" onClick={onRefresh} disabled={refreshing || busy}>
          {refreshing ? "Refreshing..." : "Refresh Tasks"}
        </button>
      </div>
      <div className="task-list">
        {tasks.map((task) => {
          const isDone = task.status === "done";
          const isSnoozed = task.status === "snoozed";
          return (
            <div className={`task-row task-${task.status}`} key={task.id}>
              <div>
                <div className="task-title-line">
                  <strong>{task.title}</strong>
                  <b className={`task-status status-${task.status}`}>{formatTaskStatus(task.status)}</b>
                </div>
                <span>{task.detail}</span>
                {task.snoozed_until && <small>Snoozed until {formatDateTime(task.snoozed_until)}</small>}
                {task.escalated_at && <small>Escalated at {formatDateTime(task.escalated_at)}</small>}
                {task.completion_reason && <small>{task.completion_reason}</small>}
              </div>
              <div className="task-actions">
                {task.action && (
                  <button type="button" onClick={() => onAction(task.action)} disabled={busy}>
                    Open
                  </button>
                )}
                <button type="button" onClick={() => onSnooze(task)} disabled={busy || isDone || isSnoozed}>
                  {isSnoozed ? "Snoozed" : "Snooze"}
                </button>
                <button type="button" onClick={() => onComplete(task)} disabled={busy || isDone}>
                  {isDone ? "Done" : "Complete"}
                </button>
                <button type="button" onClick={() => onIgnore(task)} disabled={busy || isDone}>
                  Ignore
                </button>
              </div>
            </div>
          );
        })}
        {tasks.length === 0 && <div className="empty-source">Refresh tasks after setting companies and running a crawl.</div>}
      </div>
    </section>
  );
}

function AgentEmployeeSidebar({
  state,
  onRefreshTasks,
  onSendFeishu,
  onRunAutomationDutyReport,
  refreshingTasks,
  sendingFeishu,
  feishuReady,
  commandText,
  commandResult,
  runningCommand,
  onCommandTextChange,
  onRunCommand,
}: {
  state: AgentState;
  onRefreshTasks: () => void | Promise<void>;
  onSendFeishu: () => void | Promise<void>;
  onRunAutomationDutyReport: () => void | Promise<void>;
  refreshingTasks: boolean;
  sendingFeishu: boolean;
  feishuReady: boolean;
  commandText: string;
  commandResult: AgentCommandResult | null;
  runningCommand: boolean;
  onCommandTextChange: (value: string) => void;
  onRunCommand: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
}) {
  const topGaps = state.gaps.slice(0, 3);
  return (
    <aside className={`employee-sidebar employee-${state.mode}`}>
      <div className="employee-portrait">
        <img src={state.profile.avatar} alt={state.profile.name} />
        <div className="employee-presence">
          <span />
          {state.profile.presence}
        </div>
      </div>

      <div className="employee-identity">
        <h2>{state.profile.name}</h2>
        <strong>{state.profile.role}</strong>
        <p>{state.profile.mission}</p>
      </div>

      <form className="command-center" onSubmit={onRunCommand}>
        <label>
          Command Center
          <textarea
            value={commandText}
            onChange={(event) => onCommandTextChange(event.target.value)}
            placeholder="Only Shenzhen Go backend, refresh tasks"
          />
        </label>
        <button type="submit" disabled={runningCommand}>
          {runningCommand ? "Working..." : "Run Command"}
        </button>
        {commandResult && (
          <div className="command-result">
            <strong>{commandResult.intent}</strong>
            <span>{commandResult.summary}</span>
            {commandResult.actions.length > 0 && (
              <small>{commandResult.actions.map((action) => action.type).join(" / ")}</small>
            )}
            {commandResult.needs.length > 0 && <small>{commandResult.needs.join(" ")}</small>}
          </div>
        )}
      </form>

      <div className="employee-focus">
        <span>Current Focus</span>
        <strong>{state.focus}</strong>
      </div>

      <div className="employee-score">
        <div>
          <span>Digital employee maturity</span>
          <strong>{state.maturity_score}</strong>
        </div>
        <div className="score-track" aria-label="Digital employee maturity">
          <span style={{ width: `${state.maturity_score}%` }} />
        </div>
      </div>

      <div className="employee-workload">
        <Metric label="Open tasks" value={state.workload.open_tasks} />
        <Metric label="Strong" value={state.workload.strong_matches} />
        <Metric label="Decisions" value={state.workload.manual_decisions} />
        <Metric label="Source issues" value={state.workload.source_issues} />
      </div>

      <div className="employee-actions">
        <button type="button" onClick={onRefreshTasks} disabled={refreshingTasks}>
          {refreshingTasks ? "Refreshing..." : "Refresh Work Queue"}
        </button>
        <button type="button" onClick={onSendFeishu} disabled={sendingFeishu || !feishuReady}>
          {sendingFeishu ? "Sending..." : "Send Duty Report"}
        </button>
        <button type="button" onClick={onRunAutomationDutyReport} disabled={sendingFeishu || !feishuReady || !state.automation.duty_report_enabled}>
          Run Auto Report
        </button>
      </div>

      <section className="employee-section">
        <h3>Automation</h3>
        <div className="automation-panel">
          <div>
            <strong>{state.automation.duty_report_enabled ? "Duty report armed" : "Duty report paused"}</strong>
            <span>Next {formatDateTime(state.automation.next_duty_report_at)} / SLA {state.automation.task_sla_hours}h</span>
          </div>
          <div>
            <strong>{state.automation.source_discovery_enabled ? "Source discovery armed" : "Source discovery paused"}</strong>
            <span>Next {formatDateTime(state.automation.next_source_discovery_due_at)} / every {state.automation.source_discovery_interval_hours}h</span>
          </div>
          <div>
            <strong>{state.automation.stale_task_count} stale tasks</strong>
            <span>{state.automation.last_report_sent_at ? `Last sent ${formatDateTime(state.automation.last_report_sent_at)}` : "No automatic report sent yet"}</span>
          </div>
        </div>
        {state.automation.stale_tasks.length > 0 && (
          <div className="stale-task-list">
            {state.automation.stale_tasks.slice(0, 3).map((task) => (
              <div className="stale-task" key={task.id}>
                <strong>{task.title}</strong>
                <span>{task.age_hours}h pending / {task.detail}</span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="employee-section">
        <h3>Capabilities</h3>
        <div className="capability-list">
          {state.capabilities.map((item) => (
            <div className="capability-row" key={item.key}>
              <div>
                <strong>{item.label}</strong>
                <span>{item.evidence}</span>
              </div>
              <b>{item.level}</b>
              <div className="capability-track">
                <span style={{ width: `${item.level}%` }} />
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="employee-section">
        <h3>Mainstream Gaps</h3>
        <div className="gap-list">
          {topGaps.map((gap) => (
            <div className="gap-item" key={gap.key}>
              <strong>{gap.label}</strong>
              <span>{gap.next_step}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="employee-section">
        <h3>Operating Cycle</h3>
        <div className="cycle-list">
          {state.operating_cycle.map((moment) => (
            <div className="cycle-row" key={`${moment.time}-${moment.title}`}>
              <strong>{moment.time}</strong>
              <span>{moment.title}</span>
            </div>
          ))}
        </div>
      </section>
    </aside>
  );
}

function GlobalEmployeeChat({
  state,
  status,
  messages,
  text,
  open,
  sending,
  activeView,
  onToggle,
  onTextChange,
  onSubmit,
  actions,
  onAction,
}: {
  state: AgentState | null;
  status: AgentChatStatus | null;
  messages: AgentChatMessage[];
  text: string;
  open: boolean;
  sending: boolean;
  activeView: string;
  onToggle: () => void;
  onTextChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
  actions: AgentCommandResult["actions"];
  onAction: (action: string) => void | Promise<void>;
}) {
  const modeLabel = status?.configured ? `Model: ${status.model}` : "Local rules";
  return (
    <aside className={open ? "global-employee open" : "global-employee"} aria-label="Digital employee chat">
      <button type="button" className="employee-fab" onClick={onToggle} aria-label="Toggle digital employee chat">
        <DigitalEmployee3D active={open} thinking={sending} />
        <span className="employee-fab-status">{sending ? "Analyzing" : status?.configured ? "Model online" : "Local online"}</span>
        <strong>Qiu Zhao</strong>
      </button>
      {open && (
        <section className="employee-chat-card">
          <div className="employee-chat-header">
            <div>
              <strong>{state?.profile.name || "Job Hunter Agent"}</strong>
              <span>{modeLabel} / {activeView}</span>
            </div>
            <button type="button" onClick={onToggle} aria-label="Close chat">
              Close
            </button>
          </div>
          <div className="employee-chat-messages">
            {messages.map((message) => (
              <div className={`chat-message chat-${message.role}`} key={message.id}>
                <span>{message.role === "assistant" ? state?.profile.name || "Agent" : "You"}</span>
                <p>{message.content}</p>
                <small>{message.source} / {formatDateTime(message.created_at)}</small>
              </div>
            ))}
            {messages.length === 0 && (
              <div className="chat-empty">
                <strong>I am here.</strong>
                <span>Ask me what to apply for today, why a role fits, or tell me to refresh tasks.</span>
              </div>
            )}
            {actions.length > 0 && (
              <div className="chat-actions">
                {actions.map((action) => (
                  <button type="button" key={`${action.type}-${action.target}`} onClick={() => onAction(action.type)}>
                    {formatActionLabel(action.type)}
                  </button>
                ))}
              </div>
            )}
          </div>
          <form className="employee-chat-input" onSubmit={onSubmit}>
            <input
              value={text}
              onChange={(event) => onTextChange(event.target.value)}
              placeholder="Ask: which roles are worth applying today?"
              aria-label="Message the digital employee"
            />
            <button type="submit" disabled={sending || text.trim() === ""}>
              {sending ? "Sending..." : "Send"}
            </button>
          </form>
        </section>
      )}
    </aside>
  );
}

function ProductReadinessPanel({
  items,
  busy,
}: {
  items: Array<{
    id: string;
    label: string;
    detail: string;
    done: boolean;
    actionLabel: string;
    action: () => void | Promise<void>;
  }>;
  busy: boolean;
}) {
  const complete = items.filter((item) => item.done).length;
  return (
    <section className="readiness-panel">
      <div className="panel-header">
        <h2>Product Readiness</h2>
        <span>{complete} / {items.length} ready</span>
      </div>
      <div className="readiness-grid">
        {items.map((item) => (
          <div className={item.done ? "readiness-item ready" : "readiness-item"} key={item.id}>
            <div>
              <strong>{item.label}</strong>
              <span>{item.detail}</span>
            </div>
            <button type="button" onClick={item.action} disabled={busy}>
              {item.actionLabel}
            </button>
          </div>
        ))}
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
    feishu_webhook_url: settings.feishu_webhook_url || "",
    time_zone: settings.time_zone || defaultSettings.time_zone,
    auto_duty_report_enabled: Boolean(settings.auto_duty_report_enabled),
    auto_source_discovery_enabled: Boolean(settings.auto_source_discovery_enabled),
    source_discovery_interval_hours: String(settings.source_discovery_interval_hours || defaultSettings.source_discovery_interval_hours),
    duty_report_time: settings.duty_report_time || defaultSettings.duty_report_time,
    task_sla_hours: String(settings.task_sla_hours || defaultSettings.task_sla_hours),
  };
}

function profileToDraft(profile: CandidateProfile): CandidateProfileDraft {
  return {
    target_cities: safeSettingsList(profile.target_cities, defaultProfile.target_cities).join("\n"),
    target_directions: safeSettingsList(profile.target_directions, defaultProfile.target_directions).join("\n"),
    skills: safeSettingsList(profile.skills, defaultProfile.skills).join("\n"),
    education: profile.education || "",
    graduation_year: profile.graduation_year || "",
    internship_preference: profile.internship_preference || defaultProfile.internship_preference,
    preferred_companies: safeSettingsList(profile.preferred_companies, []).join("\n"),
    blocked_keywords: safeSettingsList(profile.blocked_keywords, defaultProfile.blocked_keywords).join("\n"),
    notes: profile.notes || "",
  };
}

function normalizeSettings(settings: Partial<Settings>): Settings {
  return {
    target_cities: safeSettingsList(settings.target_cities, defaultSettings.target_cities),
    target_directions: safeSettingsList(settings.target_directions, defaultSettings.target_directions),
    excluded_keywords: safeSettingsList(settings.excluded_keywords, defaultSettings.excluded_keywords),
    crawl_schedule: safeSettingsList(settings.crawl_schedule, defaultSettings.crawl_schedule),
    feishu_webhook_url: settings.feishu_webhook_url || "",
    feishu_configured: Boolean(settings.feishu_configured),
    time_zone: settings.time_zone || defaultSettings.time_zone,
    auto_duty_report_enabled: Boolean(settings.auto_duty_report_enabled),
    auto_source_discovery_enabled: settings.auto_source_discovery_enabled ?? defaultSettings.auto_source_discovery_enabled,
    source_discovery_interval_hours: settings.source_discovery_interval_hours || defaultSettings.source_discovery_interval_hours,
    duty_report_time: settings.duty_report_time || defaultSettings.duty_report_time,
    task_sla_hours: settings.task_sla_hours || defaultSettings.task_sla_hours,
    last_duty_report_sent_at: settings.last_duty_report_sent_at,
    last_source_discovery_at: settings.last_source_discovery_at,
    updated_at: settings.updated_at || "",
  };
}

function normalizeProfile(profile: Partial<CandidateProfile>): CandidateProfile {
  return {
    id: profile.id || defaultProfile.id,
    target_cities: safeSettingsList(profile.target_cities, defaultProfile.target_cities),
    target_directions: safeSettingsList(profile.target_directions, defaultProfile.target_directions),
    skills: safeSettingsList(profile.skills, defaultProfile.skills),
    education: profile.education || "",
    graduation_year: profile.graduation_year || "",
    internship_preference: profile.internship_preference || defaultProfile.internship_preference,
    preferred_companies: safeSettingsList(profile.preferred_companies, []),
    blocked_keywords: safeSettingsList(profile.blocked_keywords, defaultProfile.blocked_keywords),
    notes: profile.notes || "",
    updated_at: profile.updated_at || "",
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
    .split(/\r?\n|,|;|\//)
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

function formatDateTime(value: string) {
  if (!value) {
    return "not scheduled";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

function formatTaskStatus(status: string) {
  const labels: Record<string, string> = {
    open: "Open",
    stale: "Stale",
    escalated: "Escalated",
    snoozed: "Snoozed",
    done: "Done",
  };
  return labels[status] || status;
}

function formatActionLabel(action: string) {
  const labels: Record<string, string> = {
    add_recommended_and_crawl: "Add sources and crawl",
    run_crawl: "Run crawl",
    review_manual_check: "Review manual jobs",
    review_low_confidence: "Review low confidence",
    cleanup_landing_pages: "Clean landing pages",
    refresh_tasks: "Refresh tasks",
    discover_sources: "Discover sources",
    review_strong_matches: "Review strong matches",
    inspect_failed_sources: "Inspect sources",
    sync_application_plans: "Sync application plans",
    prepare_application: "Open applications",
    follow_up_application: "Follow up applications",
  };
  return labels[action] || action.replace(/_/g, " ");
}

function formatFitVerdict(verdict: string) {
  const labels: Record<string, string> = {
    strong_fit: "Strong fit",
    worth_reviewing: "Worth reviewing",
    manual_check: "Manual check",
    low_priority: "Low priority",
  };
  return labels[verdict] || verdict;
}

function formatDecisionAction(decision: { action: string; from_status: string; to_status: string }) {
  if (decision.action === "status_changed") {
    return `${decision.from_status || "unknown"} -> ${decision.to_status || "unknown"}`;
  }
  if (decision.action === "notes_updated") {
    return "Notes updated";
  }
  return decision.action.replace("_", " ");
}

function signedDisplay(value: number) {
  if (value > 0) {
    return `+${value}`;
  }
  return String(value);
}

function Metric({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
