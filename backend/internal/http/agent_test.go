package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/domain"
	"github.com/wenroudeyanhuo/job-hunter-agent/backend/internal/jobs"
)

func TestAgentBriefingAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/briefing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response struct {
		Tone    string `json:"tone"`
		Metrics struct {
			TotalJobs int `json:"total_jobs"`
		} `json:"metrics"`
		NextActions []struct {
			Action string `json:"action"`
		} `json:"next_actions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Metrics.TotalJobs != 1 {
		t.Fatalf("expected one job in briefing, got %#v", response)
	}
	if len(response.NextActions) == 0 {
		t.Fatalf("expected next actions, got %#v", response)
	}
}

func TestAgentDutyReportAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.CreateSource(t.Context(), jobs.SourceInput{
		Name:       "Meituan Campus",
		URL:        "https://campus.meituan.com/",
		Enabled:    true,
		ParserType: "meituan_api",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if err := repo.UpdateSourceHealthByURL(t.Context(), "https://campus.meituan.com/", jobs.SourceHealthInput{
		Status:  jobs.SourceHealthBroken,
		Reason:  "HTTP 502",
		Success: false,
	}); err != nil {
		t.Fatalf("mark source broken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/report", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentDutyReport
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Summary.StrongMatches != 1 || response.Summary.SourceIssues != 1 {
		t.Fatalf("unexpected report summary: %#v", response.Summary)
	}
	if response.NextBestAction.Action != "inspect_failed_sources" {
		t.Fatalf("expected source inspection action, got %#v", response.NextBestAction)
	}
}

func TestAgentReviewAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "DJI",
		Title:      "AI Application Engineer",
		City:       "Shenzhen",
		MatchScore: 91,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.CreateSource(t.Context(), jobs.SourceInput{
		Name:       "DJI Careers",
		URL:        "https://we.dji.com/",
		Enabled:    true,
		ParserType: "generic",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if _, err := repo.SyncAgentTasks(t.Context(), time.Now().UTC()); err != nil {
		t.Fatalf("sync tasks: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/review", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentReview
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode review: %v", err)
	}
	if response.Focus.Action != "run_crawl" && response.Focus.Action != "review_strong_matches" {
		t.Fatalf("expected actionable focus, got %#v", response.Focus)
	}
	if len(response.Findings) == 0 || len(response.NextSteps) == 0 {
		t.Fatalf("expected review findings and next steps, got %#v", response)
	}
}

func TestAgentReviewSnapshotAndHistoryAPI(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.CreateSource(t.Context(), jobs.SourceInput{
		Name:       "Tencent Careers",
		URL:        "https://careers.tencent.com/",
		Enabled:    true,
		ParserType: "tencent_api",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if _, err := repo.SyncAgentTasks(t.Context(), time.Now().UTC()); err != nil {
		t.Fatalf("sync tasks: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/review/snapshot", strings.NewReader(`{"trigger_type":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "DJI",
		Title:      "AI Application Engineer",
		City:       "Shenzhen",
		MatchScore: 93,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed second job: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "/api/agent/review/snapshot", strings.NewReader(`{"trigger_type":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 second snapshot, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agent/review/history", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentReviewHistory
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode history: %v", err)
	}
	if len(response.Snapshots) != 2 {
		t.Fatalf("expected two snapshots, got %#v", response.Snapshots)
	}
	if response.Delta.StrongMatches != 1 || response.Delta.TrackedJobs != 1 {
		t.Fatalf("expected positive trend delta, got %#v", response.Delta)
	}
}

func TestAgentTasksAPIRefreshesAndCompletesTasks(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/tasks/refresh", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var tasks []jobs.AgentTask
	if err := json.Unmarshal(rec.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode tasks: %v", err)
	}
	if len(tasks) == 0 || tasks[0].Status != jobs.AgentTaskStatusOpen {
		t.Fatalf("expected open tasks, got %#v", tasks)
	}

	req = httptest.NewRequest(http.MethodPatch, "/api/agent/tasks/"+strconv.FormatInt(tasks[0].ID, 10), strings.NewReader(`{"status":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/agent/tasks", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	tasks = nil
	if err := json.Unmarshal(rec.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode listed tasks: %v", err)
	}
	if tasks[0].Status != jobs.AgentTaskStatusDone {
		t.Fatalf("expected completed task to be listed, got %#v", tasks[0])
	}
}

func TestAgentTaskAPIStoresSnoozeAndCompletionReason(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	tasks, err := repo.SyncAgentTasks(t.Context(), time.Now().UTC())
	if err != nil {
		t.Fatalf("sync tasks: %v", err)
	}
	snoozedUntil := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/agent/tasks/"+strconv.FormatInt(tasks[0].ID, 10),
		strings.NewReader(`{"status":"snoozed","snoozed_until":"`+snoozedUntil+`"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	updated, err := repo.GetAgentTask(t.Context(), tasks[0].ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if updated.Status != jobs.AgentTaskStatusSnoozed || updated.SnoozedUntil == nil {
		t.Fatalf("expected snoozed task, got %#v", updated)
	}

	req = httptest.NewRequest(
		http.MethodPatch,
		"/api/agent/tasks/"+strconv.FormatInt(tasks[0].ID, 10),
		strings.NewReader(`{"status":"done","completion_reason":"Not a fit"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	updated, err = repo.GetAgentTask(t.Context(), tasks[0].ID)
	if err != nil {
		t.Fatalf("get completed task: %v", err)
	}
	if updated.Status != jobs.AgentTaskStatusDone || updated.CompletionReason != "Not a fit" {
		t.Fatalf("expected completed task reason, got %#v", updated)
	}
}

func TestAgentDutyReportIncludesDailyTaskState(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.SyncAgentTasks(t.Context(), time.Now().UTC()); err != nil {
		t.Fatalf("sync tasks: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/report", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentDutyReport
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if response.Summary.OpenTasks == 0 || len(response.Tasks) == 0 {
		t.Fatalf("expected report task state, got summary=%#v tasks=%#v", response.Summary, response.Tasks)
	}
}

func TestAgentStateAPIReportsDigitalEmployeeReadiness(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	if _, err := repo.CreateSource(t.Context(), jobs.SourceInput{
		Name:       "Tencent Careers",
		URL:        "https://careers.tencent.com/",
		Enabled:    true,
		ParserType: "tencent_api",
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	if _, err := repo.SyncAgentTasks(t.Context(), time.Now().UTC()); err != nil {
		t.Fatalf("sync tasks: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/state", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentState
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	if response.Profile.Name == "" || response.MaturityScore == 0 {
		t.Fatalf("expected profile and maturity score, got %#v", response)
	}
	if response.Workload.OpenTasks == 0 {
		t.Fatalf("expected open task workload, got %#v", response.Workload)
	}
	if len(response.Capabilities) == 0 || len(response.Gaps) == 0 {
		t.Fatalf("expected capabilities and gaps, got %#v", response)
	}
	if response.Automation.TaskSLAHours == 0 {
		t.Fatalf("expected automation state, got %#v", response.Automation)
	}
}

func TestAgentCommandAPIUpdatesPreferencesAndRefreshesTasks(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/commands", strings.NewReader(`{"text":"只看深圳和广州 Go 后端，刷新任务"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentCommandResult
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode command result: %v", err)
	}
	if len(response.Actions) < 2 {
		t.Fatalf("expected settings and task actions, got %#v", response)
	}
	settings, err := repo.GetSettings(t.Context())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if !containsString(settings.TargetCities, "深圳") || !containsString(settings.TargetCities, "广州") {
		t.Fatalf("expected command to update cities, got %#v", settings.TargetCities)
	}
	if !containsString(settings.TargetDirections, "go") || !containsString(settings.TargetDirections, "backend") {
		t.Fatalf("expected command to update directions, got %#v", settings.TargetDirections)
	}
	tasks, err := repo.ListAgentTasks(t.Context(), time.Now().UTC().Format("2006-01-02"))
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatalf("expected refreshed tasks")
	}
}

func TestAgentCommandAPISyncsApplicationPlans(t *testing.T) {
	repo, handler := testRouter(t, nil)
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusInterested,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/commands", strings.NewReader(`{"text":"同步投递计划，准备投递感兴趣岗位"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	plans, err := repo.ListApplicationPlans(t.Context(), "")
	if err != nil {
		t.Fatalf("list application plans: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected synced application plan, got %#v", plans)
	}
}

func TestAgentAutomationDutyReportSendsAndRecordsLastSent(t *testing.T) {
	var received string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Content struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode webhook: %v", err)
		}
		received = payload.Content.Text
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	repo, handler := testRouter(t, nil)
	settings := jobs.DefaultSettings()
	settings.FeishuWebhookURL = server.URL
	settings.AutoDutyReportEnabled = true
	if _, err := repo.SaveSettings(t.Context(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	if _, err := repo.CreateJob(t.Context(), domain.Job{
		Company:    "Tencent",
		Title:      "Go Backend Engineer",
		City:       "Shenzhen",
		MatchScore: 88,
		Status:     domain.StatusNew,
	}); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/agent/automation/duty-report", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(received, "Job Hunter Agent duty report") {
		t.Fatalf("expected duty report payload, got %q", received)
	}
	updated, err := repo.GetSettings(t.Context())
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if updated.LastDutyReportSentAt == nil {
		t.Fatalf("expected last report timestamp")
	}
}

func TestAgentAutomationStatusReportsFeishuReadiness(t *testing.T) {
	repo, handler := testRouter(t, nil)
	settings := jobs.DefaultSettings()
	settings.FeishuWebhookURL = "https://open.feishu.cn/open-apis/bot/v2/hook/test"
	settings.AutoDutyReportEnabled = true
	settings.TimeZone = "Asia/Shanghai"
	if _, err := repo.SaveSettings(t.Context(), settings); err != nil {
		t.Fatalf("save settings: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/agent/automation/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var response jobs.AgentAutomationDiagnostics
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode automation status: %v", err)
	}
	if !response.ReadyForAutomaticReport || !response.WebhookConfigured || response.TimeZone != "Asia/Shanghai" {
		t.Fatalf("expected ready automation diagnostics, got %#v", response)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
