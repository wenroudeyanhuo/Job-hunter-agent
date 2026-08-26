import { useEffect, useMemo, useState } from "react";
import { useLang } from "./useLang";
import type { TranslationKey } from "./i18n";
import {
  acceptSourceCandidate,
  cleanupLandingPages,
  createSource,
  getAutomationStatus,
  getAgentChatConfig,
  getAgentChatStatus,
  checkAgentChatModel,
  createTodayAgentPlan,
  getAgentBriefing,
  getAgentDutyReport,
  getAgentReview,
  getAgentReviewHistory,
  getAgentState,
  getAgentPreferenceInsights,
  getOnboardingHealth,
  getCandidateProfile,
  getJobDetail,
  getSettings,
  getSourceOperations,
  importURL,
  listAgentActionRequests,
  listAgentCycles,
  listAgentEvents,
  listAgentChatMessages,
  listAgentPlans,
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
  runAgentCycle,
  runCrawl,
  runAgentCommand,
  runRecommendedCrawl,
  runSourceDiscovery,
  rebuildSemanticMemory,
  saveAgentReviewSnapshot,
  searchSemanticMemory,
  sendFeishuReport,
  sendFeishuTest,
  seedRecommendedSources,
  syncApplicationPlans,
  updateAgentChatConfig,
  updateAgentTaskStatus,
  updateAgentActionRequest,
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
import type { AgentActionRequest, AgentAutomationDiagnostics, AgentBriefing, AgentChatHealthcheck, AgentChatMessage, AgentChatStatus, AgentCommandResult, AgentCycleRecord, AgentDutyReport, AgentEvent, AgentPlan, AgentPreferenceInsights, AgentReview, AgentReviewHistory, AgentState, AgentTask, ApplicationPlan, CandidateProfile, Company, Job, JobDetail, JobRun, JobRunSource, JobStatus, LLMConfig, OnboardingHealth, RunSummary, SemanticMemoryMatch, Settings, Source, SourceCandidate, SourceOperationsSummary } from "./types";

type AppView = "dashboard" | "opportunities" | "applications" | "memory" | "profile" | "companies" | "runs" | "settings";

const statusLabelKeys: Record<JobStatus | "all", TranslationKey> = {
  all: "status.all",
  new: "status.new",
  interested: "status.interested",
  applied: "status.applied",
  ignored: "status.ignored",
  manual_check: "status.manualCheck",
  expired: "status.expired",
};

const sourceHealthLabelKeys: Record<string, TranslationKey> = {
  healthy: "health.healthy",
  warning: "health.warning",
  broken: "health.broken",
  unknown: "health.unknown",
};

const appViewIds: AppView[] = ["dashboard", "opportunities", "applications", "memory", "profile", "companies", "runs", "settings"];

const appViewLabelKeys: Record<AppView, TranslationKey> = {
  dashboard: "view.dashboard",
  opportunities: "view.opportunities",
  applications: "view.applications",
  memory: "view.memory",
  profile: "view.profile",
  companies: "view.companies",
  runs: "view.runs",
  settings: "view.settings",
};

const categoryLabelKeys: Record<string, TranslationKey> = {
  all: "category.all",
  internet: "category.internet",
  ai: "category.ai",
  hardware: "category.hardware",
  fintech: "category.fintech",
  game: "category.game",
  new_energy: "category.newEnergy",
  software: "category.software",
  security: "category.security",
  logistics: "category.logistics",
  medical: "category.medical",
  manufacturing: "category.manufacturing",
  custom: "category.custom",
  general: "category.general",
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
  const { lang, toggleLang, t } = useLang();
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
  const [sourceOperations, setSourceOperations] = useState<SourceOperationsSummary | null>(null);
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
  const [llmConfig, setLLMConfig] = useState<LLMConfig | null>(null);
  const [llmDraft, setLLMDraft] = useState({ provider: "", apiKey: "", baseURL: "", model: "" });
  const [savingLLM, setSavingLLM] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [lastRun, setLastRun] = useState<RunSummary | null>(null);
  const [briefing, setBriefing] = useState<AgentBriefing | null>(null);
  const [agentState, setAgentState] = useState<AgentState | null>(null);
  const [dutyReport, setDutyReport] = useState<AgentDutyReport | null>(null);
  const [agentReview, setAgentReview] = useState<AgentReview | null>(null);
  const [agentReviewHistory, setAgentReviewHistory] = useState<AgentReviewHistory | null>(null);
  const [agentEvents, setAgentEvents] = useState<AgentEvent[]>([]);
  const [agentActionRequests, setAgentActionRequests] = useState<AgentActionRequest[]>([]);
  const [agentPlans, setAgentPlans] = useState<AgentPlan[]>([]);
  const [agentCycles, setAgentCycles] = useState<AgentCycleRecord[]>([]);
  const [preferenceInsights, setPreferenceInsights] = useState<AgentPreferenceInsights | null>(null);
  const [agentTasks, setAgentTasks] = useState<AgentTask[]>([]);
  const [applicationPlans, setApplicationPlans] = useState<ApplicationPlan[]>([]);
  const [automationStatus, setAutomationStatus] = useState<AgentAutomationDiagnostics | null>(null);
  const [onboardingHealth, setOnboardingHealth] = useState<OnboardingHealth | null>(null);
  const [chatStatus, setChatStatus] = useState<AgentChatStatus | null>(null);
  const [chatHealthcheck, setChatHealthcheck] = useState<AgentChatHealthcheck | null>(null);
  const [chatMessages, setChatMessages] = useState<AgentChatMessage[]>([]);
  const [chatActions, setChatActions] = useState<AgentCommandResult["actions"]>([]);
  const [chatText, setChatText] = useState("");
  const [memoryQuery, setMemoryQuery] = useState("Go backend AI application Shenzhen");
  const [memoryMatches, setMemoryMatches] = useState<SemanticMemoryMatch[]>([]);
  const [memoryBusy, setMemoryBusy] = useState(false);
  const [chatOpen, setChatOpen] = useState(true);
  const [chatSending, setChatSending] = useState(false);
  const [checkingModel, setCheckingModel] = useState(false);
  const [selectedJobDetail, setSelectedJobDetail] = useState<JobDetail | null>(null);
  const [loadingJobDetail, setLoadingJobDetail] = useState(false);
  const [refreshingTasks, setRefreshingTasks] = useState(false);
  const [syncingApplications, setSyncingApplications] = useState(false);
  const [commandText, setCommandText] = useState("");
  const [commandResult, setCommandResult] = useState<AgentCommandResult | null>(null);
  const [runningCommand, setRunningCommand] = useState(false);
  const [savingReviewSnapshot, setSavingReviewSnapshot] = useState(false);
  const [planningToday, setPlanningToday] = useState(false);
  const [runningAgentCycle, setRunningAgentCycle] = useState(false);

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

  async function refreshSourceOperations() {
    const data = await getSourceOperations();
    setSourceOperations(data);
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

  async function refreshAgentActionRequests() {
    const data = await listAgentActionRequests("pending");
    setAgentActionRequests(data);
  }

  async function refreshAgentPlans() {
    const data = await listAgentPlans();
    setAgentPlans(data);
  }

  async function refreshAgentCycles() {
    const data = await listAgentCycles();
    setAgentCycles(data);
  }

  async function refreshPreferenceInsights() {
    const data = await getAgentPreferenceInsights();
    setPreferenceInsights(data);
  }

  async function refreshChat() {
    const [status, messages] = await Promise.all([getAgentChatStatus(), listAgentChatMessages()]);
    setChatStatus(status);
    setChatMessages(messages);
  }

  async function handleCheckChatModel() {
    setCheckingModel(true);
    setError("");
    setNotice("");
    try {
      const result = await checkAgentChatModel();
      setChatHealthcheck(result);
      setNotice(result.status === "ok" ? t("notice.modelCheckPassed") : result.message);
      await refreshChat();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.checkModel"));
    } finally {
      setCheckingModel(false);
    }
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

  async function refreshOnboardingHealth() {
    const data = await getOnboardingHealth();
    setOnboardingHealth(data);
  }

  async function refreshLLMConfig() {
    const data = await getAgentChatConfig();
    setLLMConfig(data);
    setLLMDraft({ provider: data.provider, apiKey: "", baseURL: data.base_url, model: data.model });
  }

  async function handleSaveLLMConfig(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingLLM(true);
    try {
      const update: Record<string, string> = {};
      if (llmDraft.provider) update.provider = llmDraft.provider;
      if (llmDraft.apiKey) update.api_key = llmDraft.apiKey;
      if (llmDraft.baseURL) update.base_url = llmDraft.baseURL;
      if (llmDraft.model) update.model = llmDraft.model;
      const saved = await updateAgentChatConfig(update);
      setLLMConfig(saved);
      setLLMDraft({ provider: saved.provider, apiKey: "", baseURL: saved.base_url, model: saved.model });
      setNotice(t("model.noticeSaved"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("model.errorSave"));
    } finally {
      setSavingLLM(false);
    }
  }

  useEffect(() => {
    Promise.all([refresh(), refreshSources(), refreshSourceCandidates(), refreshSourceOperations(), refreshCompanies(), refreshRuns(), refreshSettings(), refreshProfile(), refreshBriefing(), refreshAgentState(), refreshDutyReport(), refreshAgentReview(), refreshAgentReviewHistory(), refreshAgentEvents(), refreshAgentActionRequests(), refreshAgentPlans(), refreshAgentCycles(), refreshPreferenceInsights(), refreshTasks(), refreshApplicationPlans(), refreshAutomationStatus(), refreshOnboardingHealth(), refreshChat(), refreshLLMConfig()])
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
      label: t("companyScope.label"),
      detail: companies.length > 0 ? t("companyScope.detailHas", { count: enabledCompanies }) : t("companyScope.detailEmpty"),
      done: companies.length > 0 && enabledCompanies > 0,
      actionLabel: companies.length > 0 ? t("companyScope.actionManage") : t("companyScope.actionAdd"),
      action: () => setActiveView("companies"),
    },
    {
      id: "preferences",
      label: t("preferences.label"),
      detail: `${settings.target_cities.join(", ")} / ${settings.target_directions.length} ${t("settings.directions")}`,
      done: settings.target_cities.length > 0 && settings.target_directions.length > 0,
      actionLabel: t("preferences.actionEdit"),
      action: () => setActiveView("settings"),
    },
    {
      id: "candidate_profile",
      label: t("candidateProfile.label"),
      detail: `${profile.skills.length} ${t("profile.skills")} / ${profile.preferred_companies.length} ${t("profile.preferredCompanies")}`,
      done: profile.skills.length > 0 && profile.target_directions.length > 0,
      actionLabel: t("candidateProfile.actionProfile"),
      action: () => setActiveView("profile"),
    },
    {
      id: "crawl_history",
      label: t("crawlHistory.label"),
      detail: runs.length > 0 ? t("crawlHistory.detailHas", { count: runs.length }) : t("crawlHistory.detailEmpty"),
      done: runs.length > 0,
      actionLabel: runs.length > 0 ? t("crawlHistory.actionViewRuns") : t("crawlHistory.actionRunCrawl"),
      action: runs.length > 0 ? () => setActiveView("runs") : handleRunCrawl,
    },
    {
      id: "feishu",
      label: t("feishu.label"),
      detail: settings.feishu_configured ? t("feishu.configured") : t("feishu.notConfigured"),
      done: settings.feishu_configured,
      actionLabel: t("feishu.actionSettings"),
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
        setNotice(t("notice.manualReview"));
        return;
      case "review_low_confidence":
        setActiveView("opportunities");
        setStatus("manual_check");
        setDirection("all");
        setScoreView("low_confidence");
        await refresh("manual_check");
        setNotice(t("notice.lowConfidence"));
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
        setNotice(t("notice.openedFollowUp"));
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
        setNotice(t("notice.strongMatches"));
        return;
      case "inspect_failed_sources":
        setActiveView("runs");
        if (runs.length > 0) {
          await selectRun(runs[0].id);
          setNotice(t("notice.openedLatestRun"));
        }
        return;
      default:
        setNotice(t("notice.monitoring"));
    }
  }

  async function handleRunCrawl() {
    setRunning(true);
    setError("");
    try {
      const summary = await runCrawl();
      setLastRun(summary);
      setNotice(t("notice.crawlFinished", { created: summary.jobs_created, cleaned: summary.landing_pages_ignored }));
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
      setError(err instanceof Error ? err.message : t("error.runFailed"));
    } finally {
      setRunning(false);
    }
  }

  async function handleOnboardingStep(stepKey: string) {
    switch (stepKey) {
      case "profile":
        setActiveView("profile");
        return;
      case "sources":
        setActiveView("companies");
        await handleRunSourceDiscovery();
        return;
      case "crawl":
        setActiveView("runs");
        await handleRunCrawl();
        return;
      case "model":
      case "reports":
        setActiveView("settings");
        return;
      default:
        setActiveView("dashboard");
    }
  }

  async function handleImportURL(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = importURLValue.trim();
    if (!value) {
      setError(t("error.pasteUrl"));
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
          ? t("notice.duplicateLink")
          : result.manual_only
            ? t("notice.manualOnly")
            : t("notice.imported"),
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
      setError(err instanceof Error ? err.message : t("error.importFailed"));
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
          ? t("notice.cleanupMoved", { count: result.ignored })
        : t("notice.cleanupNone"),
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
      setError(err instanceof Error ? err.message : t("error.cleanupFailed"));
    } finally {
      setCleaningLandingPages(false);
    }
  }

  async function handleAddSource(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = sourceURLValue.trim();
    if (!value) {
      setError(t("error.pasteSourceUrl"));
      return;
    }
    setAddingSource(true);
    setError("");
    setNotice("");
    try {
      await createSource(value);
      setSourceURLValue("");
      setNotice(t("notice.sourceAdded"));
      await refreshSources();
      await refreshSourceOperations();
      await refreshCompanies();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.addSource"));
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
    await refreshSourceOperations();
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
    await refreshSourceOperations();
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
          ? t("notice.sourcesAdded", { count: result.created })
      : t("notice.sourcesAlreadyAdded"),
      );
      await refreshSources();
      await refreshSourceOperations();
      await refreshCompanies();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshTasks();
      await refreshAgentState();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.addRecommended"));
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
      setNotice(t("notice.discoveryFinished", { created: result.created, duplicated: result.duplicated }));
      await refreshSourceCandidates();
      await refreshSourceOperations();
      await refreshAgentEvents();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.discoverSources"));
    } finally {
      setDiscoveringSources(false);
    }
  }

  async function handleAcceptSourceCandidate(candidate: SourceCandidate) {
    setError("");
    setNotice("");
    try {
      await acceptSourceCandidate(candidate.id);
      setNotice(t("notice.candidateAccepted", { name: candidate.name }));
      await refreshSourceCandidates();
      await refreshSources();
      await refreshSourceOperations();
      await refreshCompanies();
      await refreshBriefing();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.acceptCandidate"));
    }
  }

  async function handleRejectSourceCandidate(candidate: SourceCandidate) {
    setError("");
    setNotice("");
    try {
      await rejectSourceCandidate(candidate.id);
      setNotice(t("notice.candidateRejected", { name: candidate.name }));
      await refreshSourceCandidates();
      await refreshSourceOperations();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.rejectCandidate"));
    }
  }

  async function handleValidateSourceCandidate(candidate: SourceCandidate) {
    setValidatingSourceCandidateId(candidate.id);
    setError("");
    setNotice("");
    try {
      const validated = await validateSourceCandidate(candidate.id);
      setNotice(t("notice.candidateValidated", { name: candidate.name, status: validated.validation_status }));
      await refreshSourceCandidates();
      await refreshSourceOperations();
      await refreshAgentEvents();
      await refreshAgentState();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.validateCandidate"));
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
        t("notice.recommendedCrawlFinished", { sources: result.sources.created, jobs: result.summary.jobs_created, cleaned: result.summary.landing_pages_ignored }),
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
      setError(err instanceof Error ? err.message : t("error.recommendedCrawl"));
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
      setNotice(t("notice.settingsSaved"));
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshTasks();
      await refreshAgentState();
      await refreshAutomationStatus();
      await refreshOnboardingHealth();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.saveSettings"));
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
      setNotice(t("notice.profileSaved"));
      if (selectedJobDetail) {
        await handleOpenJobDetail(selectedJobDetail.job.id);
      }
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.saveProfile"));
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
      setNotice(t("notice.feishuTestSent"));
      await refreshSettings();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.sendFeishuTest"));
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
      setNotice(t("notice.feishuReportSent"));
      await refreshAgentEvents();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.sendFeishuReport"));
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
      setNotice(t("notice.autoReportSent"));
      await refreshSettings();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshAgentState();
      await refreshAutomationStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.runAutoReport"));
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
      setNotice(t("notice.tasksRefreshed"));
      await refreshApplicationPlans();
      await refreshDutyReport();
      await refreshAgentReview();
      await refreshAgentReviewHistory();
      await refreshAgentEvents();
      await refreshAgentState();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.refreshTasks"));
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
      setNotice(t("notice.plansSynced", { count: plans.length }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.syncPlans"));
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
      setNotice(t("notice.planStatusUpdated", { status }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.updatePlan"));
    }
  }

  async function handleApplicationPlanUpdate(plan: ApplicationPlan, update: Partial<ApplicationPlan>) {
    setError("");
    setNotice("");
    try {
      await updateApplicationPlan(plan.id, { ...plan, ...update });
      await refreshApplicationPlans();
      await refreshTasks();
      await refreshDutyReport();
      await refreshAgentEvents();
      await refreshAgentState();
      setNotice(t("notice.planMetadataSaved"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.updatePlan"));
    }
  }

  async function handleTaskDone(task: AgentTask) {
    await updateAgentTaskStatus(task.id, "done", { completion_reason: "Completed from dashboard" });
    setNotice(t("notice.taskCompleted"));
    await refreshAfterTaskMutation();
  }

  async function handleTaskSnooze(task: AgentTask) {
    const snoozedUntil = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
    await updateAgentTaskStatus(task.id, "snoozed", { snoozed_until: snoozedUntil });
    setNotice(t("notice.taskSnoozed"));
    await refreshAfterTaskMutation();
  }

  async function handleTaskIgnore(task: AgentTask) {
    await updateAgentTaskStatus(task.id, "done", { completion_reason: "Ignored from dashboard" });
    setNotice(t("notice.taskIgnored"));
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
      setError(t("error.typeCommand"));
      return;
    }
    setRunningCommand(true);
    setError("");
    setNotice("");
    try {
      const result = await runAgentCommand(value);
      setCommandResult(result);
      setCommandText("");
      setNotice(result.summary || t("notice.commandProcessed"));
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
      setError(err instanceof Error ? err.message : t("error.runCommand"));
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
      await refreshAgentActionRequests();
      await refreshAgentPlans();
      await refreshAgentEvents();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.talkToAgent"));
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
      setError(err instanceof Error ? err.message : t("error.loadJobDetail"));
    } finally {
      setLoadingJobDetail(false);
    }
  }

  async function handleSaveJobNotes(job: Job, notes: string) {
    await updateJobNotes(job.id, notes);
    await refresh();
    await handleOpenJobDetail(job.id);
    await refreshAgentEvents();
    setNotice(t("notice.jobNotesSaved"));
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
      setNotice(t("notice.snapshotSaved"));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.saveSnapshot"));
    } finally {
      setSavingReviewSnapshot(false);
    }
  }

  async function handleCreateTodayPlan() {
    setPlanningToday(true);
    setError("");
    setNotice("");
    try {
      const plan = await createTodayAgentPlan();
      await refreshAgentPlans();
      await refreshAgentActionRequests();
      await refreshAgentEvents();
      setNotice(t("notice.planCreated", { count: plan.steps.length }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.createPlan"));
    } finally {
      setPlanningToday(false);
    }
  }

  async function handleRebuildSemanticMemory() {
    setMemoryBusy(true);
    setError("");
    setNotice("");
    try {
      const result = await rebuildSemanticMemory();
      const matches = await searchSemanticMemory(memoryQuery);
      setMemoryMatches(matches);
      await refreshAgentState();
      await refreshAgentEvents();
      setNotice(t("notice.memoryRebuilt", { created: result.created, skipped: result.skipped }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.rebuildMemory"));
    } finally {
      setMemoryBusy(false);
    }
  }

  async function handleSearchSemanticMemory(event?: React.FormEvent<HTMLFormElement>) {
    event?.preventDefault();
    const query = memoryQuery.trim();
    if (!query) {
      setMemoryMatches([]);
      return;
    }
    setMemoryBusy(true);
    setError("");
    try {
      setMemoryMatches(await searchSemanticMemory(query));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.searchMemory"));
    } finally {
      setMemoryBusy(false);
    }
  }

  async function handleApproveActionRequest(request: AgentActionRequest) {
    setError("");
    setNotice("");
    try {
      await updateAgentActionRequest(request.id, "approved");
      await handleApprovedActionNavigation(request.action_type);
      await refreshAfterAgentActionExecution();
      setNotice(t("notice.actionApproved", { action: formatActionLabel(request.action_type, t) }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.approveAction"));
    }
  }

  async function handleApprovedActionNavigation(action: string) {
    switch (action) {
      case "add_recommended_and_crawl":
        setActiveView("companies");
        return;
      case "review_manual_check":
        setActiveView("opportunities");
        setScoreView("all");
        await handleStatusFilter("manual_check");
        return;
      case "review_strong_matches":
        setActiveView("opportunities");
        setStatus("all");
        setDirection("all");
        setScoreView("strong");
        await refresh("all");
        return;
      case "sync_application_plans":
        setActiveView("applications");
        return;
      case "discover_sources":
        setActiveView("companies");
        return;
      default:
        return;
    }
  }

  async function refreshAfterAgentActionExecution() {
    await Promise.all([
      refresh(),
      refreshSources(),
      refreshSourceCandidates(),
      refreshSourceOperations(),
      refreshRuns(),
      refreshBriefing(),
      refreshDutyReport(),
      refreshAgentReview(),
      refreshAgentReviewHistory(),
      refreshAgentEvents(),
      refreshAgentActionRequests(),
      refreshAgentPlans(),
      refreshAgentCycles(),
      refreshPreferenceInsights(),
      refreshTasks(),
      refreshApplicationPlans(),
      refreshAutomationStatus(),
      refreshAgentState(),
    ]);
  }

  async function handleRunAgentCycle() {
    setRunningAgentCycle(true);
    setError("");
    setNotice("");
    try {
      const result = await runAgentCycle();
      await Promise.all([refreshAgentCycles(), refreshAgentActionRequests(), refreshAgentPlans(), refreshAgentEvents(), refreshAgentState()]);
      setNotice(`Agent cycle completed. ${result.action_requests_created} approval requests created.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not run agent cycle");
    } finally {
      setRunningAgentCycle(false);
    }
  }

  async function handleDismissActionRequest(request: AgentActionRequest) {
    setError("");
    setNotice("");
    try {
      await updateAgentActionRequest(request.id, "dismissed");
      await refreshAgentActionRequests();
      await refreshAgentPlans();
      await refreshAgentEvents();
      setNotice(t("notice.actionDismissed", { action: formatActionLabel(request.action_type, t) }));
    } catch (err) {
      setError(err instanceof Error ? err.message : t("error.dismissAction"));
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
    await refreshPreferenceInsights();
  }

  return (
    <main className="app-shell">
      <header className="topbar">
        <div>
          <h1>{t("app.title")}</h1>
          <p>{t("app.subtitle")}</p>
        </div>
        <div className="topbar-actions">
          <button className="lang-toggle" onClick={toggleLang} aria-label="Switch language">
            {t("lang.toggle")}
          </button>
          <button className="primary-button" onClick={handleRunCrawl} disabled={running}>
            {running ? t("action.running") : t("action.runCrawl")}
          </button>
        </div>
      </header>

      <nav className="view-nav" aria-label={t("app.primaryViews")}>
        {appViewIds.map((viewId) => (
          <button key={viewId} className={activeView === viewId ? "active-view" : ""} onClick={() => setActiveView(viewId)}>
            {t(appViewLabelKeys[viewId])}
          </button>
        ))}
      </nav>

      {notice && <div className="notice-banner">{notice}</div>}
      {error && <div className="error-banner">{error}</div>}

      {activeView === "dashboard" && (
        <section className="dashboard-workbench">
          <div className="dashboard-main">
            <section className="summary-grid">
              <Metric label={t("metric.trackedJobs")} value={jobs.length} />
              <Metric label={t("metric.strongMatches")} value={strongMatches} />
              <Metric label={t("metric.enabledCompanies")} value={enabledCompanies} />
              <Metric label={t("metric.nextRuns")} value={settings.crawl_schedule.join(" / ")} />
            </section>

            <ProductReadinessPanel items={readinessItems} busy={running || seedingSources || recommendedRunning} />

            <AgentWorkPlansPanel plans={agentPlans} onCreateTodayPlan={handleCreateTodayPlan} busy={planningToday} />

            <PreferenceInsightsPanel insights={preferenceInsights} />

            <AgentActionRequestsPanel
              requests={agentActionRequests}
              onApprove={handleApproveActionRequest}
              onDismiss={handleDismissActionRequest}
              busy={running || recommendedRunning}
            />

            <AgentCyclesPanel cycles={agentCycles} onRunCycle={handleRunAgentCycle} busy={runningAgentCycle} />

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
                <span>{t("run.created", { count: lastRun.jobs_created })}</span>
                <span>{t("run.duplicated", { count: lastRun.jobs_duplicated })}</span>
                <span>{t("run.failedSources", { count: lastRun.sources_failed })}</span>
                <span>{t("run.manualCheck", { count: lastRun.manual_check_count })}</span>
                <span>{t("run.cleaned", { count: lastRun.landing_pages_ignored })}</span>
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

      {activeView === "memory" && (
        <section className="content-stack">
          <section className="panel">
            <div className="panel-heading">
              <div>
                <h2>{t("panel.semanticMemory")}</h2>
                <p>{t("panel.semanticMemoryDesc")}</p>
              </div>
              <button className="secondary-button" onClick={handleRebuildSemanticMemory} disabled={memoryBusy}>
                {memoryBusy ? t("action.indexing") : t("action.rebuildIndex")}
              </button>
            </div>
            <div className="summary-grid compact-grid">
              <Metric label={t("metric.memoryItems")} value={agentState?.memory.semantic_total_items ?? 0} />
              <Metric label={t("metric.jobMemories")} value={agentState?.memory.semantic_job_items ?? 0} />
              <Metric label={t("metric.provider")} value={agentState?.memory.semantic_provider || "local_hash"} />
              <Metric label={t("metric.dimension")} value={agentState?.memory.semantic_dimension || 64} />
            </div>
            <form className="memory-search" onSubmit={handleSearchSemanticMemory}>
              <input
                value={memoryQuery}
                onChange={(event) => setMemoryQuery(event.target.value)}
                placeholder={t("placeholder.searchMemory")}
              />
              <button type="submit" disabled={memoryBusy}>
                {t("action.search")}
              </button>
            </form>
          </section>

          <section className="panel">
            <div className="panel-heading">
              <div>
                <h2>{t("panel.retrievedMemories")}</h2>
                <p>{t("panel.semanticMatches", { count: memoryMatches.length })}</p>
              </div>
            </div>
            <div className="memory-results">
              {memoryMatches.length === 0 && <p className="empty-state">{t("empty.noMemory")}</p>}
              {memoryMatches.map((match) => (
                <article className="memory-result" key={`${match.kind}-${match.reference_id}`}>
                  <div className="memory-result-main">
                    <span className="score-pill">{Math.round(match.score * 100)}</span>
                    <div>
                      <h3>{match.title || `${match.kind} #${match.reference_id}`}</h3>
                      <p>{compactMemoryContent(match.content)}</p>
                      <div className="job-reasons">
                        <span>{match.kind}</span>
                        {match.metadata.city && <span>{match.metadata.city}</span>}
                        {match.metadata.status && <span>{match.metadata.status}</span>}
                        {match.metadata.score && <span>score {match.metadata.score}</span>}
                      </div>
                    </div>
                  </div>
                  {match.kind === "job" && (
                    <button className="ghost-button" onClick={() => handleOpenJobDetail(match.reference_id)}>
                      {t("action.open")}
                    </button>
                  )}
                </article>
              ))}
            </div>
          </section>
        </section>
      )}

      {activeView === "opportunities" && (
        <>
          <form className="import-bar" onSubmit={handleImportURL}>
            <input
              value={importURLValue}
              onChange={(event) => setImportURLValue(event.target.value)}
              placeholder={t("placeholder.pasteUrl")}
              aria-label={t("placeholder.pasteUrl")}
            />
            <button type="submit" disabled={importing}>
              {importing ? t("action.importing") : t("action.importURL")}
            </button>
            <button type="button" className="secondary-action" onClick={handleCleanupLandingPages} disabled={cleaningLandingPages}>
              {cleaningLandingPages ? t("action.cleaning") : t("action.cleanLandingPages")}
            </button>
          </form>

          <section className="workspace">
        <aside className="filters">
          <h2>{t("panel.filters")}</h2>
          <label>
            {t("column.status")}
            <select value={status} onChange={(event) => handleStatusFilter(event.target.value as JobStatus | "all")}>
              {(Object.entries(statusLabelKeys) as [string, TranslationKey][]).map(([value, key]) => (
                <option key={value} value={value}>
                  {t(key)}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("settings.directions")}
            <select value={direction} onChange={(event) => setDirection(event.target.value)}>
              {directionOptions.map((value) => (
                <option key={value} value={value}>
                  {value === "all" ? t("direction.all") : value.replace("_", " ")}
                </option>
              ))}
            </select>
          </label>
          <label>
            {t("column.score")}
            <select value={scoreView} onChange={(event) => setScoreView(event.target.value as "all" | "strong" | "low_confidence")}>
              <option value="all">{t("status.all")}</option>
              <option value="strong">{t("metric.strongMatches")}</option>
              <option value="low_confidence">{t("fitVerdict.manualCheck")}</option>
            </select>
          </label>
        </aside>

        <section className="job-panel">
          <div className="panel-header">
            <h2>{t("view.opportunities")}</h2>
            {loading && <span>{t("action.running")}</span>}
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>{t("column.score")}</th>
                  <th>{t("column.company")}</th>
                  <th>{t("column.role")}</th>
                  <th>{t("column.city")}</th>
                  <th>{t("column.tags")}</th>
                  <th>{t("column.status")}</th>
                  <th>{t("column.actions")}</th>
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
                    <td>{job.city || t("health.unknown")}</td>
                    <td>
                      <div className="tags">
                        {job.direction_tags.map((tag) => (
                          <span key={tag}>{tag.replace("_", " ")}</span>
                        ))}
                      </div>
                    </td>
                    <td>{t(statusLabelKeys[job.status])}</td>
                    <td>
                      <div className="row-actions">
                        <button onClick={() => handleOpenJobDetail(job.id)} disabled={loadingJobDetail}>
                          {t("action.details")}
                        </button>
                        <button onClick={() => setJobStatus(job.id, "interested")}>{t("action.interested")}</button>
                        <button onClick={() => setJobStatus(job.id, "applied")}>{t("action.applied")}</button>
                        <button onClick={() => setJobStatus(job.id, "ignored")}>{t("action.ignore")}</button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!loading && visibleJobs.length === 0 && (
                  <tr>
                    <td colSpan={7} className="empty-state">
                      {t("empty.noJobs")}
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
          onUpdate={handleApplicationPlanUpdate}
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
          <h2>{t("panel.companies")}</h2>
          <span>{t("label.enabledCompanies", { enabled: enabledCompanies, total: companies.length })}</span>
        </div>
        {sourceOperations && <SourceOperationsPanel summary={sourceOperations} onAction={handleAgentAction} busy={discoveringSources || recommendedRunning} />}
        <div className="company-toolbar">
          <input
            value={companyQuery}
            onChange={(event) => setCompanyQuery(event.target.value)}
            placeholder={t("placeholder.searchCompany")}
            aria-label={t("placeholder.searchCompany")}
          />
          <select value={companyCategoryFilter} onChange={(event) => setCompanyCategoryFilter(event.target.value)}>
            {companyCategories.map((category) => (
              <option key={category} value={category}>
                {categoryLabelKeys[category] ? t(categoryLabelKeys[category]) : category}
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
                  <span>{categoryLabelKeys[company.category] ? t(categoryLabelKeys[company.category]) : company.category || t("category.general")}</span>
                  <span>{t("label.companySources", { count: company.source_count })}</span>
                  {company.broken_count > 0 && <span>{t("label.companyBroken", { count: company.broken_count })}</span>}
                  {company.warning_count > 0 && <span>{t("label.companyWarning", { count: company.warning_count })}</span>}
                </div>
              </div>
              <button className={company.enabled ? "toggle-on" : "toggle-off"} onClick={() => toggleCompany(company)}>
                {company.enabled ? t("action.enabled") : t("action.disabled")}
              </button>
            </div>
          ))}
          {visibleCompanies.length === 0 && <div className="empty-source">{t("empty.noCompanies")}</div>}
        </div>
        <div className="source-actions">
          <button type="button" onClick={handleSeedRecommendedSources} disabled={seedingSources || recommendedRunning}>
            {seedingSources ? t("action.adding") : t("action.addRecommended")}
          </button>
          <button type="button" onClick={handleRunSourceDiscovery} disabled={discoveringSources || recommendedRunning}>
            {discoveringSources ? t("action.discovering") : t("action.discoverSources")}
          </button>
          <button type="button" className="strong-action" onClick={handleRecommendedCrawl} disabled={recommendedRunning || seedingSources}>
            {recommendedRunning ? t("action.running") : t("action.addCrawl")}
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
            placeholder={t("placeholder.addSourceUrl")}
            aria-label={t("placeholder.addSourceUrl")}
          />
          <button type="submit" disabled={addingSource}>
            {addingSource ? t("action.adding") : t("action.addSource")}
          </button>
        </form>
        <div className="source-list">
          {visibleSources.map((source) => (
            <div className="source-row" key={source.id}>
              <div>
                <strong>{source.name}</strong>
                <div className="source-meta">
                  <span>{categoryLabelKeys[source.category] ? t(categoryLabelKeys[source.category]) : source.category || t("category.general")}</span>
                  <span>{source.parser_type || t("label.generic")}</span>
                </div>
                <a href={source.url} target="_blank" rel="noreferrer">
                  {source.url}
                </a>
                <div className="source-health">
                  <span className={`health-badge health-${source.health_status || "unknown"}`}>
                    {sourceHealthLabelKeys[source.health_status] ? t(sourceHealthLabelKeys[source.health_status]) : source.health_status || t("health.unknown")}
                  </span>
                  <span>{source.health_reason || t("notice.waitingFirstCrawl")}</span>
                  <span>{t("run.found")} {source.last_found_count ?? 0}</span>
                  {source.consecutive_failures > 0 && <span>{t("run.failedSources", { count: source.consecutive_failures })}</span>}
                </div>
              </div>
              <button className={source.enabled ? "toggle-on" : "toggle-off"} onClick={() => toggleSource(source)}>
                {source.enabled ? t("action.enabled") : t("action.disabled")}
              </button>
            </div>
          ))}
          {visibleSources.length === 0 && <div className="empty-source">{t("empty.noSources")}</div>}
        </div>
      </section>
      )}

      {activeView === "settings" && (
      <section className="settings-panel">
        <div className="panel-header">
          <h2>{t("panel.settings")}</h2>
          <span>{settings.feishu_configured ? t("feishu.configured") : t("feishu.notConfigured")}</span>
        </div>
        {automationStatus && (
          <div className={automationStatus.ready_for_automatic_report ? "automation-diagnostic ready" : "automation-diagnostic"}>
            <div>
              <strong>{automationStatus.ready_for_automatic_report ? t("label.dutyReportEnabled") : t("label.dutyReportDisabled")}</strong>
              <span>{automationStatus.reason}</span>
            </div>
            <div className="automation-diagnostic-grid">
              <span>{automationStatus.scheduler_expected ? t("label.schedulerExpected") : t("label.schedulerNotExpected")}</span>
              <span>{automationStatus.webhook_configured ? t("label.webhookConfigured") : t("label.webhookMissing")}</span>
              <span>{automationStatus.duty_report_enabled ? t("label.dutyReportEnabled") : t("label.dutyReportDisabled")}</span>
              <span>{automationStatus.time_zone} / {automationStatus.duty_report_time}</span>
              <span>{t("label.notScheduled")} {formatDateTime(automationStatus.next_duty_report_at)}</span>
              <span>{automationStatus.last_duty_report_sent_at ? formatDateTime(automationStatus.last_duty_report_sent_at) : t("notice.noAutoReport")}</span>
            </div>
          </div>
        )}
        {onboardingHealth && (
          <div className="automation-diagnostic onboarding-health-card">
            <div>
              <strong>{t("onboarding.title")} · {onboardingHealth.readiness_score}/100</strong>
              <span>{t("onboarding.openTasks", { count: onboardingHealth.open_tasks })}</span>
            </div>
            <div className="automation-diagnostic-grid">
              <span>{onboardingHealth.database_ready ? t("onboarding.databaseReady") : t("onboarding.databaseMissing")}</span>
              <span>{onboardingHealth.source_pool_ready ? t("onboarding.sourcesReady") : t("onboarding.sourcesMissing")}</span>
              <span>{onboardingHealth.profile_ready ? t("onboarding.profileReady") : t("onboarding.profileMissing")}</span>
              <span>{onboardingHealth.has_crawl_history ? t("onboarding.crawlReady") : t("onboarding.crawlMissing")}</span>
              <span>{onboardingHealth.feishu_configured ? t("label.webhookConfigured") : t("label.webhookMissing")}</span>
              <span>{onboardingHealth.model_configured ? t("chat.modelOnline") : t("chat.localRules")}</span>
            </div>
            {onboardingHealth.next_steps.length > 0 && (
              <div className="onboarding-next-steps">
                {onboardingHealth.next_steps.slice(0, 4).map((step) => (
                  <small key={step}>{step}</small>
                ))}
              </div>
            )}
            <div className="onboarding-wizard">
              {(onboardingHealth.wizard_steps || []).map((step, index) => (
                <div className={step.done ? "wizard-step done" : "wizard-step"} key={step.key || step.title}>
                  <span>{index + 1}</span>
                  <div>
                    <strong>{step.title}</strong>
                    <small>{step.detail}</small>
                  </div>
                  <button type="button" onClick={() => handleOnboardingStep(step.key)} disabled={step.done || running || discoveringSources}>
                    {step.done ? t("action.done") : step.action}
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}
        <form className="settings-grid" onSubmit={handleSaveSettings}>
          <label>
            {t("settings.targetCities")}
            <textarea
              value={settingsDraft.target_cities}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, target_cities: event.target.value }))}
            />
          </label>
          <label>
            {t("settings.directions")}
            <textarea
              value={settingsDraft.target_directions}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, target_directions: event.target.value }))}
            />
          </label>
          <label>
            {t("settings.excludedKeywords")}
            <textarea
              value={settingsDraft.excluded_keywords}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, excluded_keywords: event.target.value }))}
            />
          </label>
          <label>
            {t("settings.crawlSchedule")}
            <textarea
              value={settingsDraft.crawl_schedule}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, crawl_schedule: event.target.value }))}
            />
          </label>
          <label>
            {t("settings.timeZone")}
            <input
              value={settingsDraft.time_zone}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, time_zone: event.target.value }))}
              placeholder="Asia/Shanghai"
            />
          </label>
          <label className="settings-wide">
            {t("settings.feishuWebhook")}
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
            {t("settings.autoDutyReport")}
          </label>
          <label className="settings-toggle">
            <input
              type="checkbox"
              checked={settingsDraft.auto_source_discovery_enabled}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, auto_source_discovery_enabled: event.target.checked }))}
            />
            {t("settings.autoSourceDiscovery")}
          </label>
          <label>
            {t("settings.dutyReportTime")}
            <input
              value={settingsDraft.duty_report_time}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, duty_report_time: event.target.value }))}
              placeholder="18:00"
            />
          </label>
          <label>
            {t("settings.discoveryInterval")}
            <input
              type="number"
              min="1"
              value={settingsDraft.source_discovery_interval_hours}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, source_discovery_interval_hours: event.target.value }))}
            />
          </label>
          <label>
            {t("settings.taskSlaHours")}
            <input
              type="number"
              min="1"
              value={settingsDraft.task_sla_hours}
              onChange={(event) => setSettingsDraft((current) => ({ ...current, task_sla_hours: event.target.value }))}
            />
          </label>
          <button type="submit" disabled={savingSettings}>
            {savingSettings ? t("action.saving") : t("action.saveSettings")}
          </button>
          <button type="button" className="secondary-settings-action" onClick={handleSendFeishuTest} disabled={testingFeishu || !settings.feishu_configured}>
            {testingFeishu ? t("action.sending") : t("action.sendFeishuTest")}
          </button>
        </form>

        <form className="settings-grid model-config-form" onSubmit={handleSaveLLMConfig}>
          <div className="model-config-header">
            <strong>{t("model.title")}</strong>
            {llmConfig && (
              <span className={llmConfig.has_api_key ? "api-key-set" : "api-key-not-set"}>
                {llmConfig.has_api_key ? t("model.apiKeySet") : t("model.apiKeyNotSet")}
              </span>
            )}
          </div>
          <label>
            {t("model.provider")}
            <select
              value={llmDraft.provider}
              onChange={(event) => setLLMDraft((current) => ({ ...current, provider: event.target.value }))}
            >
              <option value="deepseek">{t("model.deepseek")}</option>
              <option value="openai_compatible">{t("model.openai")}</option>
            </select>
          </label>
          <label>
            {t("model.model")}
            <input
              value={llmDraft.model}
              onChange={(event) => setLLMDraft((current) => ({ ...current, model: event.target.value }))}
              placeholder={t("model.placeholderModel")}
            />
          </label>
          <label className="settings-wide">
            {t("model.baseUrl")}
            <input
              value={llmDraft.baseURL}
              onChange={(event) => setLLMDraft((current) => ({ ...current, baseURL: event.target.value }))}
              placeholder={t("model.placeholderBaseUrl")}
            />
          </label>
          <label className="settings-wide">
            {t("model.apiKey")}
            <input
              type="password"
              value={llmDraft.apiKey}
              onChange={(event) => setLLMDraft((current) => ({ ...current, apiKey: event.target.value }))}
              placeholder={t("model.placeholderApiKey")}
            />
            <small>{t("model.apiKeyHint")}</small>
          </label>
          <button type="submit" disabled={savingLLM}>
            {savingLLM ? t("model.saving") : t("model.save")}
          </button>
        </form>
      </section>
      )}

      {activeView === "runs" && (
      <section className="runs-panel">
        <div className="panel-header">
          <h2>{t("panel.crawlRuns")}</h2>
          <span>{t("run.recorded", { count: runs.length })}</span>
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
                  +{run.jobs_created} / {t("run.dup")} {run.jobs_duplicated} / {t("run.filtered")} {run.sources_failed}
                </span>
              </button>
            ))}
            {runs.length === 0 && <div className="empty-source">{t("run.noRuns")}</div>}
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
                  <span>{t("run.found")} {source.jobs_found}</span>
                  <span>{t("run.new")} {source.jobs_created}</span>
                  <span>{t("run.dup")} {source.jobs_duplicated}</span>
                  <span>{t("run.filtered")} {source.jobs_filtered}</span>
                  <span>{t("run.manual")} {source.manual_check_count}</span>
                </div>
              </div>
            ))}
            {selectedRunId !== null && runSources.length === 0 && <div className="empty-source">{t("run.noSourceResults")}</div>}
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
        checkingModel={checkingModel}
        healthcheck={chatHealthcheck}
        activeView={activeView}
        onToggle={() => setChatOpen((current) => !current)}
        onTextChange={setChatText}
        onSubmit={handleSendChat}
        onCheckModel={handleCheckChatModel}
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
  const { t } = useLang();
  return (
    <section className="profile-panel">
      <div className="panel-header">
        <div>
          <h2>{t("panel.candidateProfile")}</h2>
          <span>{formatDateTime(profile.updated_at)}</span>
        </div>
      </div>
      <form className="profile-grid" onSubmit={onSubmit}>
        <label>
          {t("profile.targetCities")}
          <textarea value={draft.target_cities} onChange={(event) => onDraftChange((current) => ({ ...current, target_cities: event.target.value }))} />
        </label>
        <label>
          {t("profile.targetDirections")}
          <textarea value={draft.target_directions} onChange={(event) => onDraftChange((current) => ({ ...current, target_directions: event.target.value }))} />
        </label>
        <label>
          {t("profile.skills")}
          <textarea value={draft.skills} onChange={(event) => onDraftChange((current) => ({ ...current, skills: event.target.value }))} />
        </label>
        <label>
          {t("profile.preferredCompanies")}
          <textarea value={draft.preferred_companies} onChange={(event) => onDraftChange((current) => ({ ...current, preferred_companies: event.target.value }))} />
        </label>
        <label>
          {t("profile.education")}
          <input value={draft.education} onChange={(event) => onDraftChange((current) => ({ ...current, education: event.target.value }))} placeholder={t("placeholder.education")} />
        </label>
        <label>
          {t("profile.graduationYear")}
          <input value={draft.graduation_year} onChange={(event) => onDraftChange((current) => ({ ...current, graduation_year: event.target.value }))} placeholder={t("placeholder.graduationYear")} />
        </label>
        <label>
          {t("profile.blockedKeywords")}
          <input value={draft.internship_preference} onChange={(event) => onDraftChange((current) => ({ ...current, internship_preference: event.target.value }))} />
        </label>
        <label>
          {t("profile.blockedKeywords")}
          <textarea value={draft.blocked_keywords} onChange={(event) => onDraftChange((current) => ({ ...current, blocked_keywords: event.target.value }))} />
        </label>
        <label className="profile-wide">
          {t("profile.notes")}
          <textarea value={draft.notes} onChange={(event) => onDraftChange((current) => ({ ...current, notes: event.target.value }))} placeholder={t("placeholder.profileNotes")} />
        </label>
        <button type="submit" disabled={saving}>
          {saving ? t("action.saving") : t("action.saveProfile")}
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
  onUpdate,
  onOpenJob,
}: {
  plans: ApplicationPlan[];
  jobs: Job[];
  syncing: boolean;
  onSync: () => void | Promise<void>;
  onStatus: (plan: ApplicationPlan, status: ApplicationPlan["status"]) => void | Promise<void>;
  onUpdate: (plan: ApplicationPlan, update: Partial<ApplicationPlan>) => void | Promise<void>;
  onOpenJob: (id: number) => void | Promise<void>;
}) {
  const { t } = useLang();
  const jobsByID = new Map(jobs.map((job) => [job.id, job]));
  const activePlans = plans.filter((plan) => plan.status !== "applied" && plan.status !== "paused");
  const columns = [
    { status: "prepare", label: t("kanban.prepare") },
    { status: "ready", label: t("kanban.ready") },
    { status: "applied", label: t("kanban.applied") },
    { status: "paused", label: t("kanban.paused") },
  ];
  return (
    <section className="applications-panel">
      <div className="panel-header">
        <div>
          <h2>{t("panel.applicationWorkspace")}</h2>
          <span>{t("label.activePlans", { active: activePlans.length, total: plans.length })}</span>
        </div>
        <button type="button" onClick={onSync} disabled={syncing}>
          {syncing ? t("action.syncing") : t("action.syncPlans")}
        </button>
      </div>
      <div className="application-board">
        {columns.map((column) => {
          const columnPlans = plans.filter((plan) => plan.status === column.status);
          return (
            <div className="application-column" key={column.status}>
              <div className="application-column-header">
                <strong>{column.label}</strong>
                <span>{columnPlans.length}</span>
              </div>
              {columnPlans.map((plan) => (
                <ApplicationPlanCard
                  key={plan.id}
                  plan={plan}
                  job={jobsByID.get(plan.job_id)}
                  onStatus={onStatus}
                  onUpdate={onUpdate}
                  onOpenJob={onOpenJob}
                />
              ))}
            </div>
          );
        })}
        {plans.length === 0 && <div className="empty-source">{t("empty.noApplicationPlans")}</div>}
      </div>
    </section>
  );
}

function ApplicationPlanCard({
  plan,
  job,
  onStatus,
  onUpdate,
  onOpenJob,
}: {
  plan: ApplicationPlan;
  job?: Job;
  onStatus: (plan: ApplicationPlan, status: ApplicationPlan["status"]) => void | Promise<void>;
  onUpdate: (plan: ApplicationPlan, update: Partial<ApplicationPlan>) => void | Promise<void>;
  onOpenJob: (id: number) => void | Promise<void>;
}) {
  const { t } = useLang();
  const [resumeVersion, setResumeVersion] = useState(plan.resume_version || "default");
  const [draftNotes, setDraftNotes] = useState(plan.draft_notes || "");
  const [followUpDate, setFollowUpDate] = useState(plan.follow_up_date || "");

  useEffect(() => {
    setResumeVersion(plan.resume_version || "default");
    setDraftNotes(plan.draft_notes || "");
    setFollowUpDate(plan.follow_up_date || "");
  }, [plan.id, plan.resume_version, plan.draft_notes, plan.follow_up_date]);

  return (
    <div className={`application-card application-${plan.status}`}>
      <div className="candidate-title">
        <strong>{job ? `${job.company} / ${job.title}` : `#${plan.job_id}`}</strong>
        <b>{plan.priority}</b>
      </div>
      <div className="source-meta">
        <span>{plan.target_apply_date || t("job.noTargetDate")}</span>
        {job?.city && <span>{job.city}</span>}
      </div>
      <p>{plan.next_action || t("job.noNextAction")}</p>
      <div className="application-edit-grid">
        <label>
          {t("profile.resume")}
          <input value={resumeVersion} onChange={(event) => setResumeVersion(event.target.value)} />
        </label>
        <label>
          {t("profile.followUp")}
          <input value={followUpDate} onChange={(event) => setFollowUpDate(event.target.value)} placeholder={t("placeholder.followUpDate")} />
        </label>
        <label className="application-edit-wide">
          {t("profile.draftNotes")}
          <textarea value={draftNotes} onChange={(event) => setDraftNotes(event.target.value)} />
        </label>
      </div>
      <div className="application-checklist">
        {plan.checklist.slice(0, 5).map((item) => (
          <span key={item}>{item}</span>
        ))}
      </div>
      <div className="candidate-actions">
        <button type="button" onClick={() => onOpenJob(plan.job_id)}>
          {t("action.details")}
        </button>
        <button type="button" onClick={() => onUpdate(plan, { resume_version: resumeVersion, draft_notes: draftNotes, follow_up_date: followUpDate })}>
          {t("action.save")}
        </button>
        {plan.status !== "ready" && (
          <button type="button" onClick={() => onStatus(plan, "ready")}>
            {t("action.ready")}
          </button>
        )}
        {plan.status !== "applied" && (
          <button type="button" onClick={() => onStatus(plan, "applied")}>
            {t("action.applied")}
          </button>
        )}
        {plan.status !== "paused" && (
          <button type="button" onClick={() => onStatus(plan, "paused")}>
            {t("action.pause")}
          </button>
        )}
      </div>
      {plan.blocker_notes && <small>{plan.blocker_notes}</small>}
    </div>
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
  const { t } = useLang();
  const [notes, setNotes] = useState(detail.job.notes || "");

  useEffect(() => {
    setNotes(detail.job.notes || "");
  }, [detail.job.id, detail.job.notes]);

  return (
    <section className="job-detail-panel">
      <div className="panel-header">
        <div>
          <h2>{detail.job.company} / {detail.job.title}</h2>
          <span>{detail.job.city || t("job.unknownCity")} / {formatFitVerdict(detail.fit.verdict, t)}</span>
        </div>
        <button type="button" className="secondary-detail-action" onClick={onClose}>
          {t("action.close")}
        </button>
      </div>
      <div className="job-detail-grid">
        <div className="fit-card">
          <span>{t("job.profileFit")}</span>
          <strong>{detail.fit.score}</strong>
          <small>{detail.suggested_action.reason}</small>
          <div className="job-detail-actions">
            <button type="button" onClick={() => onStatus(detail.job.id, "interested")} disabled={busy}>
              {t("action.interested")}
            </button>
            <button type="button" onClick={() => onStatus(detail.job.id, "applied")} disabled={busy}>
              {t("action.applied")}
            </button>
            <button type="button" onClick={() => onStatus(detail.job.id, "ignored")} disabled={busy}>
              {t("action.ignore")}
            </button>
          </div>
        </div>
        <div className="detail-column">
          <h3>{t("job.whyItFits")}</h3>
          {[...detail.fit.profile_signals, ...detail.fit.strengths].slice(0, 8).map((item) => (
            <span className="detail-pill" key={item}>{item}</span>
          ))}
          {detail.fit.profile_signals.length === 0 && detail.fit.strengths.length === 0 && <div className="empty-source">{t("empty.noFitSignals")}</div>}
        </div>
        <div className="detail-column">
          <h3>{t("job.risks")}</h3>
          {detail.fit.risks.slice(0, 8).map((item) => (
            <span className="risk-pill" key={item}>{item}</span>
          ))}
          {detail.fit.risks.length === 0 && <div className="empty-source">{t("empty.noRisks")}</div>}
        </div>
        <div className="detail-column">
          <h3>{t("job.applicationPlan")}</h3>
          {detail.application_plan ? (
            <>
              <span className="detail-pill">{detail.application_plan.status}</span>
              <span className="detail-pill">{detail.application_plan.target_apply_date || t("job.noTargetDate")}</span>
              <small>{detail.application_plan.next_action}</small>
              {detail.application_plan.checklist.slice(0, 4).map((item) => (
                <span className="detail-pill" key={item}>{item}</span>
              ))}
            </>
          ) : (
            <div className="empty-source">{t("empty.noPlanHint")}</div>
          )}
        </div>
      </div>
      <div className="job-detail-bottom">
        <form className="notes-box" onSubmit={(event) => { event.preventDefault(); onSaveNotes(detail.job, notes); }}>
          <label>
            {t("job.decisionNotes")}
            <textarea value={notes} onChange={(event) => setNotes(event.target.value)} />
          </label>
          <button type="submit" disabled={busy}>
            {t("action.saveNotes")}
          </button>
        </form>
        <div className="decision-timeline">
          <h3>{t("job.decisionHistory")}</h3>
          {detail.decisions.map((decision) => (
            <div className="decision-row" key={decision.id}>
              <strong>{formatDecisionAction(decision, t)}</strong>
              <span>{decision.notes || decision.reason || `${decision.from_status || "unknown"} -> ${decision.to_status || "unknown"}`}</span>
              <time>{formatDateTime(decision.created_at, t)}</time>
            </div>
          ))}
          {detail.decisions.length === 0 && <div className="empty-source">{t("empty.noDecisionHistory")}</div>}
        </div>
      </div>
      <div className="detail-links">
        {detail.job.apply_url && <a href={detail.job.apply_url} target="_blank" rel="noreferrer">{t("job.applyUrl")}</a>}
        {detail.job.source_url && <a href={detail.job.source_url} target="_blank" rel="noreferrer">{t("job.sourceUrl")}</a>}
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
  const { t } = useLang();
  return (
    <section className={`agent-briefing agent-${briefing.tone}`}>
      <div>
        <div className="agent-kicker">{t("panel.agentBriefing")}</div>
        <h2>{briefing.headline}</h2>
        <div className="agent-highlights">
          {briefing.highlights.length > 0 ? (
            briefing.highlights.map((highlight) => <span key={highlight}>{highlight}</span>)
          ) : (
            <span>{t("empty.waitingCrawl")}</span>
          )}
        </div>
      </div>
      <div className="agent-metrics">
        <Metric label={t("dutyReport.strong")} value={briefing.metrics.strong_matches} />
        <Metric label={t("dutyReport.manual")} value={briefing.metrics.manual_check_jobs} />
        <Metric label={t("dutyReport.lowConf")} value={briefing.metrics.low_confidence_jobs} />
        <Metric label={t("dutyReport.sources")} value={briefing.metrics.enabled_sources} />
        <Metric label={t("dutyReport.broken")} value={briefing.metrics.broken_sources} />
      </div>
      <div className="agent-actions">
        {briefing.next_actions.map((action) => (
          <div className="agent-action" key={action.action}>
            <strong>{action.label}</strong>
            <span>{action.reason}</span>
            <button type="button" onClick={() => onAction(action.action)} disabled={busy}>
              {t("action.doIt")}
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
  const { t } = useLang();
  const topDecision = report.needs_decision.slice(0, 3);
  const sourceIssues = report.source_issues.slice(0, 3);
  const recommended = (report.recommended_jobs || []).slice(0, 3);
  return (
    <section className={`duty-report duty-${report.tone}`}>
      <div className="panel-header">
        <div>
          <h2>{t("panel.todaysWork")}</h2>
          <span>{report.headline}</span>
        </div>
        <div className="duty-actions">
          <button type="button" onClick={() => onAction(report.next_best_action.action)} disabled={busy}>
            {report.next_best_action.label}
          </button>
          <button type="button" className="secondary-duty-action" onClick={onSendFeishu} disabled={sendingFeishu || !feishuReady}>
            {sendingFeishu ? t("action.sending") : t("action.sendToFeishu")}
          </button>
        </div>
      </div>
      <div className="duty-grid">
        <div className="duty-column">
          <h3>{t("dutyReport.queue")}</h3>
          {report.todays_work.map((item) => (
            <div className="duty-item" key={item.kind}>
              <div>
                <strong>{item.title}</strong>
                <span>{item.detail}</span>
              </div>
              <b>{item.count}</b>
            </div>
          ))}
          {report.todays_work.length === 0 && <div className="empty-source">{t("empty.noActiveWork")}</div>}
        </div>
        <div className="duty-column">
          <h3>{t("dutyReport.needsDecision")}</h3>
          {topDecision.map((item) => (
            <div className="decision-item" key={`${item.job_id}-${item.job_title}`}>
              <strong>{item.company} / {item.job_title}</strong>
              <span>{item.city} / {t("label.score")} {item.score}</span>
              <small>{item.reason}</small>
            </div>
          ))}
          {topDecision.length === 0 && <div className="empty-source">{t("empty.noManualDecisions")}</div>}
        </div>
        <div className="duty-column">
          <h3>{t("dutyReport.sourceIssues")}</h3>
          {sourceIssues.map((issue) => (
            <div className={`source-issue issue-${issue.status}`} key={issue.source_id || issue.url}>
              <strong>{issue.name}</strong>
              <span>{sourceHealthLabelKeys[issue.status] ? t(sourceHealthLabelKeys[issue.status]) : issue.status} / {issue.reason}</span>
              <small>{t("run.found")} {issue.last_found_count} / {t("run.failedSources", { count: issue.consecutive_failures })}</small>
            </div>
          ))}
          {sourceIssues.length === 0 && <div className="empty-source">{t("empty.sourcesStable")}</div>}
        </div>
      </div>
      {(report.learning_summary || recommended.length > 0) && (
        <div className="duty-learning">
          {report.learning_summary && <strong>{report.learning_summary}</strong>}
          <div className="duty-recommendations">
            {recommended.map((job) => (
              <div className="duty-recommendation" key={`${job.job_id}-${job.title}`}>
                <div>
                  <span>{job.company} / {job.title}</span>
                  <small>{job.city || "Unknown city"} / {t("label.score")} {job.score}</small>
                </div>
                <p>{job.reasons.slice(0, 2).join(" · ") || "Ranked by profile and decision history."}</p>
              </div>
            ))}
          </div>
        </div>
      )}
      <div className="duty-summary">
        <span>{report.summary.new_jobs} {t("run.new")}</span>
        <span>{report.summary.strong_matches} {t("dutyReport.strong")}</span>
        <span>{report.summary.manual_check} {t("dutyReport.manual")}</span>
        <span>{report.summary.source_issues} {t("metric.sourceIssues")}</span>
        <span>{report.summary.open_tasks} {t("tasks.open")}</span>
        <span>{report.summary.stale_tasks} {t("tasks.stale")}</span>
        <span>{report.summary.escalated_tasks} {t("tasks.escalated")}</span>
        <span>{report.summary.done_tasks} {t("tasks.done")}</span>
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
  const { t } = useLang();
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
          {t("action.takeAction")}
        </button>
        <button className="secondary-review-action" type="button" onClick={onSaveSnapshot} disabled={busy || savingSnapshot}>
          {savingSnapshot ? t("action.saving") : t("action.saveSnapshot")}
        </button>
      </div>
      <div className="review-trend">
        <div>
          <strong>{t("dutyReport.trendReview")}</strong>
          <span>{history?.summary || t("empty.noTrendMemory")}</span>
        </div>
        <div className="trend-metrics">
          <TrendMetric label={t("metric.jobs")} value={history?.delta.tracked_jobs || 0} />
          <TrendMetric label={t("metric.strong")} value={history?.delta.strong_matches || 0} />
          <TrendMetric label={t("dutyReport.manual")} value={history?.delta.manual_decisions || 0} />
          <TrendMetric label={t("dutyReport.sources")} value={history?.delta.source_issues || 0} inverse />
          <TrendMetric label={t("metric.tasks")} value={history?.delta.open_tasks || 0} inverse />
          <TrendMetric label={t("metric.applied")} value={history?.delta.applied_jobs || 0} />
        </div>
      </div>
      <div className="review-body">
        <div className="review-column">
          <h3>{t("dutyReport.findings")}</h3>
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
          <h3>{t("dutyReport.needsDecisionShort")}</h3>
          {decisions.map((decision) => (
            <div className="review-decision" key={decision.question}>
              <strong>{decision.question}</strong>
              <span>{decision.context}</span>
              <button type="button" onClick={() => onAction(decision.action)} disabled={busy}>
                {t("action.decide")}
              </button>
            </div>
          ))}
          {decisions.length === 0 && <div className="empty-source">{t("empty.noBlockingDecision")}</div>}
        </div>
        <div className="review-column">
          <h3>{t("dutyReport.recentMemory")}</h3>
          {recentSnapshots.map((snapshot) => (
            <div className="review-memory" key={snapshot.id}>
              <strong>{snapshot.health_label} / {snapshot.focus_title}</strong>
              <span>{snapshot.trigger_type} / {formatDateTime(snapshot.captured_at, t)}</span>
              <small>{snapshot.stats.strong_matches} {t("dutyReport.strong")} / {snapshot.stats.source_issues} {t("metric.sourceIssues")} / {snapshot.stats.open_tasks} {t("metric.tasks")}</small>
            </div>
          ))}
          {recentSnapshots.length === 0 && <div className="empty-source">{t("empty.noSnapshots")}</div>}
          <h3 className="review-next-heading">{t("dutyReport.nextSteps")}</h3>
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
  const { t } = useLang();
  const pending = candidates.filter((candidate) => candidate.status === "pending");
  const recent = candidates.filter((candidate) => candidate.status !== "pending").slice(0, 4);
  return (
    <section className="candidate-panel">
      <div className="panel-header">
        <h2>{t("panel.sourceDiscovery")}</h2>
        <span>{pending.length} {t("label.pending")} / {candidates.length} {t("status.all").toLowerCase()}</span>
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
                <span>{categoryLabelKeys[candidate.category] ? t(categoryLabelKeys[candidate.category]) : candidate.category}</span>
                <span>{candidate.parser_type || t("label.generic")}</span>
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
                {validatingId === candidate.id ? t("action.checking") : t("action.validate")}
              </button>
              <button type="button" onClick={() => onAccept(candidate)} disabled={busy}>
                {t("action.accept")}
              </button>
              <button type="button" onClick={() => onReject(candidate)} disabled={busy}>
                {t("action.reject")}
              </button>
            </div>
          </div>
        ))}
        {pending.length === 0 && <div className="empty-source">{t("sourceDiscovery.noPending")}</div>}
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

function SourceOperationsPanel({
  summary,
  onAction,
  busy,
}: {
  summary: SourceOperationsSummary;
  onAction: (action: string) => void | Promise<void>;
  busy: boolean;
}) {
  const { t } = useLang();
  return (
    <section className="source-ops-panel">
      <div className="source-ops-metrics">
        <Metric label={t("dutyReport.sources")} value={`${summary.enabled_sources}/${summary.total_sources}`} />
        <Metric label={t("metric.sourceQuality")} value={summary.source_quality_score} />
        <Metric label={t("metric.healthy")} value={summary.healthy_sources} />
        <Metric label={t("metric.unhealthy")} value={summary.warning_sources + summary.broken_sources} />
        <Metric label={t("metric.candidates")} value={summary.pending_candidates} />
      </div>
      <div className="source-ops-body">
        <div>
          <h3>{t("sourceDiscovery.needsAttention")}</h3>
          {summary.needs_attention.slice(0, 4).map((source) => (
            <div className="source-ops-row" key={source.id}>
              <strong>{source.name}</strong>
              <span>{source.status} / {source.reason || t("label.noReasonRecorded")}</span>
            </div>
          ))}
          {summary.needs_attention.length === 0 && <span className="source-ops-empty">{t("sourceDiscovery.noUnhealthy")}</span>}
        </div>
        <div>
          <h3>{t("sourceDiscovery.promoteCandidates")}</h3>
          {summary.recommended_promotes.slice(0, 4).map((candidate) => (
            <div className="source-ops-row" key={candidate.id}>
              <strong>{candidate.name}</strong>
              <span>{candidate.validation_status} / {t("label.confidence")} {candidate.confidence}</span>
            </div>
          ))}
          {summary.recommended_promotes.length === 0 && <span className="source-ops-empty">{t("sourceDiscovery.noHighConfidence")}</span>}
        </div>
        <div className="source-ops-actions">
          {summary.actions.map((action) => (
            <button type="button" key={`${action.type}-${action.target}`} onClick={() => onAction(action.type)} disabled={busy}>
              {formatActionLabel(action.type, t)}
            </button>
          ))}
        </div>
      </div>
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
  const { t } = useLang();
  return (
    <section className="activity-panel">
      <div className="panel-header">
        <h2>{t("panel.activityLog")}</h2>
        <span>{t("run.recorded", { count: events.length })}</span>
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
        {events.length === 0 && <div className="empty-source">{t("empty.noActivity")}</div>}
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
  const { t } = useLang();
  const openTasks = tasks.filter((task) => task.status !== "done");
  const doneTasks = tasks.length - openTasks.length;
  const staleTasks = tasks.filter((task) => task.status === "stale").length;
  const escalatedTasks = tasks.filter((task) => task.status === "escalated").length;
  return (
    <section className="tasks-panel">
      <div className="panel-header">
        <h2>{t("panel.dailyTasks")}</h2>
        <span>{t("tasks.active", { open: openTasks.length, stale: staleTasks, escalated: escalatedTasks, done: doneTasks })}</span>
      </div>
      <div className="tasks-toolbar">
        <span>{tasks.length > 0 ? `${t("label.workDate")} ${tasks[0].task_date}` : t("label.noTaskQueue")}</span>
        <button type="button" onClick={onRefresh} disabled={refreshing || busy}>
          {refreshing ? t("action.refreshing") : t("action.refreshTasks")}
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
                  <b className={`task-status status-${task.status}`}>{formatTaskStatus(task.status, t)}</b>
                </div>
                <span>{task.detail}</span>
                {task.snoozed_until && <small>{t("label.snoozedUntil")} {formatDateTime(task.snoozed_until, t)}</small>}
                {task.escalated_at && <small>{t("label.escalatedAt")} {formatDateTime(task.escalated_at, t)}</small>}
                {task.completion_reason && <small>{task.completion_reason}</small>}
              </div>
              <div className="task-actions">
                {task.action && (
                  <button type="button" onClick={() => onAction(task.action)} disabled={busy}>
                    {t("action.open")}
                  </button>
                )}
                <button type="button" onClick={() => onSnooze(task)} disabled={busy || isDone || isSnoozed}>
                  {isSnoozed ? t("action.snoozed") : t("action.snooze")}
                </button>
                <button type="button" onClick={() => onComplete(task)} disabled={busy || isDone}>
                  {isDone ? t("action.done") : t("action.complete")}
                </button>
                <button type="button" onClick={() => onIgnore(task)} disabled={busy || isDone}>
                  {t("action.ignore")}
                </button>
              </div>
            </div>
          );
        })}
        {tasks.length === 0 && <div className="empty-source">{t("tasks.refreshHint")}</div>}
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
  const { t } = useLang();
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
          {t("label.commandCenter")}
          <textarea
            value={commandText}
            onChange={(event) => onCommandTextChange(event.target.value)}
            placeholder={t("placeholder.command")}
          />
        </label>
        <button type="submit" disabled={runningCommand}>
          {runningCommand ? t("action.working") : t("action.runCommand")}
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
        <span>{t("sidebar.currentFocus")}</span>
        <strong>{state.focus}</strong>
      </div>

      <div className="employee-memory">
        <span>{t("sidebar.memory")}</span>
        <strong>{state.memory.last_focus_title || t("sidebar.noReviewMemory")}</strong>
        <small>{state.memory.trend_summary}</small>
        <small>
          {state.memory.last_review_at ? `${t("label.lastSent")} ${formatDateTime(state.memory.last_review_at, t)}` : t("plans.snapshotHint")}
          {state.memory.recent_action_count > 0 ? ` / ${state.memory.recent_action_count}` : ""}
        </small>
      </div>

      <div className="employee-cycle-summary">
        <span>Latest Agent Cycle</span>
        <strong>{state.cycle.summary || "No cycle recorded yet"}</strong>
        <small>{state.cycle.last_cycle_at ? formatDateTime(state.cycle.last_cycle_at, t) : "Run a crawl or cycle to create the first trace"}</small>
        <div className="employee-cycle-metrics">
          <Metric label="Readiness" value={state.cycle.readiness_score || 0} />
          <Metric label="Agents" value={state.cycle.trace_count || 0} />
          <Metric label="Actions" value={state.cycle.action_count || 0} />
        </div>
      </div>

      <div className="employee-score">
        <div>
          <span>{t("sidebar.maturity")}</span>
          <strong>{state.maturity_score}</strong>
        </div>
        <div className="score-track" aria-label={t("sidebar.maturity")}>
          <span style={{ width: `${state.maturity_score}%` }} />
        </div>
      </div>

      <div className="employee-workload">
        <Metric label={t("metric.openTasks")} value={state.workload.open_tasks} />
        <Metric label={t("metric.plans")} value={state.workload.active_plans} />
        <Metric label={t("metric.approvals")} value={state.workload.pending_approvals} />
        <Metric label={t("metric.strong")} value={state.workload.strong_matches} />
        <Metric label={t("metric.decisions")} value={state.workload.manual_decisions} />
        <Metric label={t("metric.sourceIssues")} value={state.workload.source_issues} />
      </div>

      <div className="employee-actions">
        <button type="button" onClick={onRefreshTasks} disabled={refreshingTasks}>
          {refreshingTasks ? t("action.refreshing") : t("action.refreshWorkQueue")}
        </button>
        <button type="button" onClick={onSendFeishu} disabled={sendingFeishu || !feishuReady}>
          {sendingFeishu ? t("action.sending") : t("action.sendDutyReport")}
        </button>
        <button type="button" onClick={onRunAutomationDutyReport} disabled={sendingFeishu || !feishuReady || !state.automation.duty_report_enabled}>
          {t("action.runAutoReport")}
        </button>
      </div>

      <section className="employee-section">
        <h3>{t("sidebar.automation")}</h3>
        <div className="automation-panel">
          <div>
            <strong>{state.automation.duty_report_enabled ? t("label.dutyReportArmed") : t("label.dutyReportPaused")}</strong>
            <span>{t("label.next")} {formatDateTime(state.automation.next_duty_report_at, t)} / {t("label.sla")} {state.automation.task_sla_hours}h</span>
          </div>
          <div>
            <strong>{state.automation.source_discovery_enabled ? t("label.sourceDiscoveryArmed") : t("label.sourceDiscoveryPaused")}</strong>
            <span>{t("label.next")} {formatDateTime(state.automation.next_source_discovery_due_at, t)} / {t("label.every")} {state.automation.source_discovery_interval_hours}h</span>
          </div>
          <div>
            <strong>{t("label.staleTasks", { count: state.automation.stale_task_count })}</strong>
            <span>{state.automation.last_report_sent_at ? `${t("label.lastSent")} ${formatDateTime(state.automation.last_report_sent_at, t)}` : t("notice.noAutoReport")}</span>
          </div>
        </div>
        {state.automation.stale_tasks.length > 0 && (
          <div className="stale-task-list">
            {state.automation.stale_tasks.slice(0, 3).map((task) => (
              <div className="stale-task" key={task.id}>
                <strong>{task.title}</strong>
                <span>{task.age_hours}h {t("label.pending")} / {task.detail}</span>
              </div>
            ))}
          </div>
        )}
      </section>

      <section className="employee-section">
        <h3>{t("sidebar.capabilities")}</h3>
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
        <h3>{t("sidebar.gaps")}</h3>
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
        <h3>{t("sidebar.operatingCycle")}</h3>
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
  checkingModel,
  healthcheck,
  activeView,
  onToggle,
  onTextChange,
  onSubmit,
  onCheckModel,
  actions,
  onAction,
}: {
  state: AgentState | null;
  status: AgentChatStatus | null;
  messages: AgentChatMessage[];
  text: string;
  open: boolean;
  sending: boolean;
  checkingModel: boolean;
  healthcheck: AgentChatHealthcheck | null;
  activeView: string;
  onToggle: () => void;
  onTextChange: (value: string) => void;
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
  onCheckModel: () => void | Promise<void>;
  actions: AgentCommandResult["actions"];
  onAction: (action: string) => void | Promise<void>;
}) {
  const { t } = useLang();
  const modeLabel = status?.configured ? `${formatProviderLabel(status.provider, t)}: ${status.model}` : t("chat.localRules");
  return (
    <aside className={open ? "global-employee open" : "global-employee"} aria-label={t("chat.digitalEmployee")}>
      <button type="button" className="employee-fab" onClick={onToggle} aria-label={t("chat.toggle")}>
        <DigitalEmployee3D active={open} thinking={sending} />
        <span className="employee-fab-status">{sending ? t("chat.analyzing") : status?.configured ? t("chat.modelOnline") : t("chat.localOnline")}</span>
        <strong>{state?.profile.name || t("chat.agentName")}</strong>
      </button>
      {open && (
        <section className="employee-chat-card">
          <div className="employee-chat-header">
            <div>
              <strong>{state?.profile.name || t("chat.agentName")}</strong>
              <span>{modeLabel} / {activeView}</span>
            </div>
            <button type="button" onClick={onCheckModel} disabled={checkingModel} aria-label={t("chat.checkModel")}>
              {checkingModel ? t("action.checking") : t("action.checkModel")}
            </button>
            <button type="button" onClick={onToggle} aria-label={t("chat.close")}>
              {t("action.close")}
            </button>
          </div>
          {healthcheck && (
            <div className={`model-health model-${healthcheck.status}`}>
              <strong>{formatExecutionStatus(healthcheck.status, t)}</strong>
              <span>{healthcheck.message}</span>
              <small>{formatProviderLabel(healthcheck.provider, t)} / {healthcheck.model || t("chat.noModel")} / {healthcheck.base_url}</small>
            </div>
          )}
          <div className="employee-chat-messages">
            {messages.map((message) => (
              <div className={`chat-message chat-${message.role}`} key={message.id}>
                <span>{message.role === "assistant" ? state?.profile.name || t("chat.agent") : t("chat.you")}</span>
                <p>{message.content}</p>
                <small>{message.source} / {formatDateTime(message.created_at, t)}</small>
              </div>
            ))}
            {messages.length === 0 && (
              <div className="chat-empty">
                <strong>{t("chat.iAmHere")}</strong>
                <span>{t("chat.welcome")}</span>
              </div>
            )}
            {actions.length > 0 && (
              <div className="chat-actions">
                {actions.map((action) => (
                  <button type="button" key={`${action.type}-${action.target}`} onClick={() => onAction(action.type)}>
                    {formatActionLabel(action.type, t)}
                  </button>
                ))}
              </div>
            )}
          </div>
          <form className="employee-chat-input" onSubmit={onSubmit}>
            <input
              value={text}
              onChange={(event) => onTextChange(event.target.value)}
              placeholder={t("placeholder.chatInput")}
              aria-label={t("placeholder.chatInput")}
            />
            <button type="submit" disabled={sending || text.trim() === ""}>
              {sending ? t("action.sending") : t("action.send")}
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
  const { t } = useLang();
  const complete = items.filter((item) => item.done).length;
  return (
    <section className="readiness-panel">
      <div className="panel-header">
        <h2>{t("panel.productReadiness")}</h2>
        <span>{t("readiness.complete", { done: complete, total: items.length })}</span>
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

function PreferenceInsightsPanel({ insights }: { insights: AgentPreferenceInsights | null }) {
  const { t } = useLang();
  const learnedPositive = [
    ...(insights?.interested_companies || []),
    ...(insights?.interested_directions || []),
  ].slice(0, 6);
  const learnedNegative = [
    ...(insights?.ignored_companies || []),
    ...(insights?.ignored_directions || []),
  ].slice(0, 6);
  const recommendations = insights?.recommended_jobs?.slice(0, 3) || [];

  return (
    <section className="preference-insights-panel">
      <div className="panel-header">
        <div>
          <h2>Learning Insights</h2>
          <span>{insights?.summary || "Waiting for your first Interested / Ignore decision"}</span>
        </div>
        <small>{insights ? `${insights.total_decisions} decisions` : "0 decisions"}</small>
      </div>
      <div className="preference-insights-grid">
        <div className="preference-signal-card">
          <strong>Preference memory</strong>
          <div className="preference-signal-list">
            {learnedPositive.map((signal) => (
              <span className="preference-chip positive" title={signal.evidence} key={`positive-${signal.label}`}>
                {signal.label} x{signal.count}
              </span>
            ))}
            {learnedPositive.length === 0 && <small>No positive preference learned yet.</small>}
          </div>
        </div>
        <div className="preference-signal-card">
          <strong>Avoidance memory</strong>
          <div className="preference-signal-list">
            {learnedNegative.map((signal) => (
              <span className="preference-chip negative" title={signal.evidence} key={`negative-${signal.label}`}>
                {signal.label} x{signal.count}
              </span>
            ))}
            {learnedNegative.length === 0 && <small>No avoidance signal learned yet.</small>}
          </div>
        </div>
      </div>
      <div className="preference-recommendations">
        {recommendations.map((job) => (
          <article className="preference-job-card" key={job.job_id}>
            <div className="preference-job-head">
              <div>
                <strong>{job.company} / {job.title}</strong>
                <span>{job.city || "Unknown city"} / {t(statusLabelKeys[job.status])}</span>
              </div>
              <b>{job.score}</b>
            </div>
            <div className="preference-job-reasons">
              {job.reasons.slice(0, 3).map((reason) => <span className="reason-positive" key={`${job.job_id}-${reason}`}>{reason}</span>)}
              {job.warnings.slice(0, 2).map((warning) => <span className="reason-warning" key={`${job.job_id}-${warning}`}>{warning}</span>)}
            </div>
          </article>
        ))}
        {recommendations.length === 0 && (
          <div className="empty-source">Mark jobs as Interested or Ignore, then the employee will explain what it learned and why it ranks future jobs.</div>
        )}
      </div>
    </section>
  );
}

function AgentActionRequestsPanel({
  requests,
  onApprove,
  onDismiss,
  busy,
}: {
  requests: AgentActionRequest[];
  onApprove: (request: AgentActionRequest) => void | Promise<void>;
  onDismiss: (request: AgentActionRequest) => void | Promise<void>;
  busy: boolean;
}) {
  const { t } = useLang();
  return (
    <section className="action-requests-panel">
      <div className="panel-header">
        <div>
          <h2>{t("panel.suggestedActions")}</h2>
          <span>{t("actions.pending", { count: requests.length })}</span>
        </div>
      </div>
      <div className="action-request-list">
        {requests.slice(0, 6).map((request) => (
          <div className="action-request-row" key={request.id}>
            <div>
              <strong>{formatActionLabel(request.action_type, t)}</strong>
              <span>{request.detail || request.target || t("actions.agentSuggested")}</span>
              {(request.tool_preview || request.tool_description || request.risk_level) && (
                <div className="tool-preview">
                  <span className={`tool-risk risk-${request.risk_level || "low"}`}>
                    {t("actions.risk")}: {formatRiskLevel(request.risk_level, t)}
                  </span>
                  <small>{request.tool_preview || request.tool_description}</small>
                  {request.requires_approval && <small>{t("actions.requiresApproval")}</small>}
                </div>
              )}
              {request.execution_status && request.execution_status !== "not_run" && (
                <small className={`execution-receipt receipt-${request.execution_status}`}>
                  {formatExecutionStatus(request.execution_status, t)}: {request.execution_message || t("actions.noExecutionDetail")}
                </small>
              )}
              <small>{request.source} / {formatDateTime(request.created_at, t)}</small>
            </div>
            <div className="action-request-actions">
              <button type="button" onClick={() => onApprove(request)} disabled={busy}>
                {t("action.approve")}
              </button>
              <button type="button" onClick={() => onDismiss(request)} disabled={busy}>
                {t("action.ignore")}
              </button>
            </div>
          </div>
        ))}
        {requests.length === 0 && <div className="empty-source">{t("notice.noSuggestedActions")}</div>}
      </div>
    </section>
  );
}

function AgentCyclesPanel({
  cycles,
  onRunCycle,
  busy,
}: {
  cycles: AgentCycleRecord[];
  onRunCycle: () => void | Promise<void>;
  busy: boolean;
}) {
  const latest = cycles[0];
  const visibleCycles = cycles.slice(0, 4);
  return (
    <section className="agent-cycles-panel">
      <div className="panel-header">
        <div>
          <h2>Agent Cycles</h2>
          <span>{latest ? `${latest.readiness_score} readiness / ${latest.actions.length} proposed actions` : "No cycle recorded yet"}</span>
        </div>
        <button type="button" onClick={onRunCycle} disabled={busy}>
          {busy ? "Running..." : "Run cycle"}
        </button>
      </div>
      <div className="agent-cycle-list">
        {visibleCycles.map((cycle) => (
          <article className="agent-cycle-card" key={cycle.id}>
            <div className="agent-cycle-head">
              <div>
                <strong>{cycle.summary || "Multi-agent recruiting cycle"}</strong>
                <span>{cycle.orchestrator_provider} / {cycle.orchestrator_pattern}</span>
              </div>
              <b>{cycle.readiness_score}</b>
            </div>
            <div className="agent-trace-grid">
              {cycle.trace.map((trace) => (
                <div className="agent-trace-item" key={`${cycle.id}-${trace.agent_key}`}>
                  <strong>{formatAgentKey(trace.agent_key)}</strong>
                  <span>{trace.observation}</span>
                  <small>{trace.decision}</small>
                </div>
              ))}
            </div>
            {cycle.actions.length > 0 && (
              <div className="agent-cycle-actions">
                {cycle.actions.map((action) => (
                  <span key={`${cycle.id}-${action.type}-${action.target}`}>{formatActionLabel(action.type)}</span>
                ))}
              </div>
            )}
            {cycle.autonomy_plan?.steps?.length > 0 && (
              <div className="agent-autonomy-plan">
                <div className="agent-autonomy-title">
                  <strong>Autonomy Plan</strong>
                  <span>{cycle.autonomy_plan.needs_approval ? "Approval gated" : "Observe only"} · {cycle.autonomy_plan.replan_after_execution ? "Re-plan after execution" : "No re-plan needed"}</span>
                </div>
                {cycle.autonomy_plan.summary && <small>{cycle.autonomy_plan.summary}</small>}
                <div className="agent-autonomy-steps">
                  {cycle.autonomy_plan.steps.slice(0, 4).map((step) => (
                    <div className="agent-autonomy-step" key={`${cycle.id}-${step.order}-${step.tool}`}>
                      <span>{step.order}</span>
                      <div>
                        <strong>{formatActionLabel(step.tool)}</strong>
                        <small>{step.detail || step.target || step.observer_hint}</small>
                      </div>
                      <b className={`risk-pill risk-${step.risk_level}`}>{step.risk_level}</b>
                    </div>
                  ))}
                </div>
              </div>
            )}
            <footer>{formatDateTime(cycle.generated_at)}</footer>
          </article>
        ))}
        {visibleCycles.length === 0 && <div className="empty-source">Run a cycle to let the employee inspect sources, jobs, memory, and plans.</div>}
      </div>
    </section>
  );
}

function AgentWorkPlansPanel({
  plans,
  onCreateTodayPlan,
  busy,
}: {
  plans: AgentPlan[];
  onCreateTodayPlan: () => void | Promise<void>;
  busy: boolean;
}) {
  const { t } = useLang();
  const visiblePlans = plans.slice(0, 4);
  return (
    <section className="agent-plans-panel">
      <div className="panel-header">
        <div>
          <h2>{t("panel.workPlans")}</h2>
          <span>{t("plans.recent", { count: plans.length })}</span>
        </div>
        <button type="button" onClick={onCreateTodayPlan} disabled={busy}>
          {busy ? t("action.planning") : t("action.planToday")}
        </button>
      </div>
      <div className="agent-plan-list">
        {visiblePlans.map((plan) => (
          <article className="agent-plan-card" key={plan.id}>
            <div className="agent-plan-head">
              <div>
                <strong>{plan.goal || t("plan.goalFallback")}</strong>
                <span>{plan.summary || t("plan.summaryFallback")}</span>
              </div>
              <small className={`plan-status status-${plan.status}`}>{formatPlanStatus(plan.status, t)}</small>
            </div>
            <div className="agent-plan-steps">
              {(plan.steps || []).map((step) => (
                <div className={`agent-plan-step step-${step.status}`} key={`${plan.id}-${step.order}-${step.action_type}`}>
                  <span>{step.order}</span>
                  <div>
                    <strong>{formatActionLabel(step.action_type, t)}</strong>
                    <small>{step.detail || step.target || t("label.noDetailRecorded")}</small>
                    {step.message && <small className="step-message">{step.message}</small>}
                  </div>
                </div>
              ))}
            </div>
            <footer>
              <span>{plan.needs_approval ? t("actions.approvalRequired") : t("actions.noApprovalNeeded")}</span>
              <span>{formatDateTime(plan.created_at, t)}</span>
            </footer>
          </article>
        ))}
        {visiblePlans.length === 0 && <div className="empty-source">{t("plans.noPlans")}</div>}
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

function formatDateTime(value: string, t?: (key: TranslationKey) => string) {
  if (!value) {
    return t ? t("label.notScheduled") : "not scheduled";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString();
}

const taskStatusLabelKeys: Record<string, TranslationKey> = {
  open: "taskStatus.open",
  stale: "taskStatus.stale",
  escalated: "taskStatus.escalated",
  snoozed: "taskStatus.snoozed",
  done: "taskStatus.done",
};

function formatTaskStatus(status: string, t?: (key: TranslationKey) => string) {
  if (t && taskStatusLabelKeys[status]) return t(taskStatusLabelKeys[status]);
  return status;
}

const planStatusLabelKeys: Record<string, TranslationKey> = {
  draft: "planStatus.draft",
  waiting_approval: "planStatus.waitingApproval",
  executing: "planStatus.executing",
  done: "planStatus.done",
  failed: "planStatus.failed",
};

function formatPlanStatus(status: string, t?: (key: TranslationKey) => string) {
  if (t && planStatusLabelKeys[status]) return t(planStatusLabelKeys[status]);
  return status;
}

const actionLabelKeys: Record<string, TranslationKey> = {
  add_recommended_and_crawl: "actionLabel.addRecommendedAndCrawl",
  run_crawl: "actionLabel.runCrawl",
  review_manual_check: "actionLabel.reviewManualCheck",
  review_low_confidence: "actionLabel.reviewLowConfidence",
  cleanup_landing_pages: "actionLabel.cleanupLandingPages",
  refresh_tasks: "actionLabel.refreshTasks",
  discover_sources: "actionLabel.discoverSources",
  rebuild_semantic_memory: "actionLabel.rebuildSemanticMemory",
  review_strong_matches: "actionLabel.reviewStrongMatches",
  inspect_failed_sources: "actionLabel.inspectSources",
  sync_application_plans: "actionLabel.syncApplicationPlans",
  prepare_application: "actionLabel.openApplications",
  follow_up_application: "actionLabel.followUpApplications",
};

function formatActionLabel(action: string, t?: (key: TranslationKey) => string) {
  if (t && actionLabelKeys[action]) return t(actionLabelKeys[action]);
  return action.replace(/_/g, " ");
}

function formatAgentKey(key: string) {
  return key
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

const executionStatusLabelKeys: Record<string, TranslationKey> = {
  succeeded: "executionStatus.succeeded",
  failed: "executionStatus.failed",
  not_run: "executionStatus.notRun",
};

function formatExecutionStatus(status: string, t?: (key: TranslationKey) => string) {
  if (t && executionStatusLabelKeys[status]) return t(executionStatusLabelKeys[status]);
  return status.replace(/_/g, " ");
}

function formatRiskLevel(risk: string, t?: (key: TranslationKey) => string) {
  const key = risk === "high" ? "risk.high" : risk === "medium" ? "risk.medium" : "risk.low";
  return t ? t(key) : risk || "low";
}

function compactMemoryContent(value: string) {
  const normalized = value.replace(/\s+/g, " ").trim();
  if (normalized.length <= 220) {
    return normalized;
  }
  return `${normalized.slice(0, 220)}...`;
}

const providerLabelKeys: Record<string, TranslationKey> = {
  deepseek: "provider.deepseek",
  openai_compatible: "provider.openaiCompatible",
};

function formatProviderLabel(provider: string, t?: (key: TranslationKey) => string) {
  if (t && providerLabelKeys[provider]) return t(providerLabelKeys[provider]);
  return provider || (t ? t("provider.local") : "Local");
}

const fitVerdictLabelKeys: Record<string, TranslationKey> = {
  strong_fit: "fitVerdict.strongFit",
  worth_reviewing: "fitVerdict.worthReviewing",
  manual_check: "fitVerdict.manualCheck",
  low_priority: "fitVerdict.lowPriority",
};

function formatFitVerdict(verdict: string, t?: (key: TranslationKey) => string) {
  if (t && fitVerdictLabelKeys[verdict]) return t(fitVerdictLabelKeys[verdict]);
  return verdict;
}

function formatDecisionAction(decision: { action: string; from_status: string; to_status: string }, t?: (key: TranslationKey) => string) {
  if (decision.action === "status_changed") {
    return `${decision.from_status || "unknown"} -> ${decision.to_status || "unknown"}`;
  }
  if (decision.action === "notes_updated") {
    return t ? t("label.notesUpdated") : "Notes updated";
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
