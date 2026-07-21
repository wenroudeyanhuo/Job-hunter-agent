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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
