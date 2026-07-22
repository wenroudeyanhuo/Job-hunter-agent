package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsAPIPersistsUpdates(t *testing.T) {
	_, handler := testRouter(t, nil)

	body := bytes.NewBufferString(`{
		"target_cities":["Shenzhen","Guangzhou"],
		"target_directions":["backend","go","ai_application"],
		"excluded_keywords":["outsourcing","training"],
		"crawl_schedule":["09:00","18:00"],
		"feishu_webhook_url":"https://open.feishu.cn/open-apis/bot/v2/hook/test",
		"time_zone":"Asia/Shanghai",
		"auto_duty_report_enabled":true,
		"auto_source_discovery_enabled":true,
		"source_discovery_interval_hours":8,
		"duty_report_time":"18:30",
		"task_sla_hours":6
	}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/settings", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/settings", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var settings struct {
		TargetCities     []string `json:"target_cities"`
		TargetDirections []string `json:"target_directions"`
		ExcludedKeywords []string `json:"excluded_keywords"`
		CrawlSchedule    []string `json:"crawl_schedule"`
		FeishuWebhookURL string   `json:"feishu_webhook_url"`
		FeishuConfigured bool     `json:"feishu_configured"`
		TimeZone         string   `json:"time_zone"`
		AutoDutyReport   bool     `json:"auto_duty_report_enabled"`
		AutoDiscovery    bool     `json:"auto_source_discovery_enabled"`
		DiscoveryHours   int      `json:"source_discovery_interval_hours"`
		DutyReportTime   string   `json:"duty_report_time"`
		TaskSLAHours     int      `json:"task_sla_hours"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &settings); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if len(settings.TargetCities) != 2 || settings.TargetCities[1] != "Guangzhou" {
		t.Fatalf("unexpected target cities: %#v", settings.TargetCities)
	}
	if len(settings.ExcludedKeywords) != 2 || settings.ExcludedKeywords[0] != "outsourcing" {
		t.Fatalf("unexpected excluded keywords: %#v", settings.ExcludedKeywords)
	}
	if len(settings.CrawlSchedule) != 2 || settings.CrawlSchedule[1] != "18:00" {
		t.Fatalf("unexpected crawl schedule: %#v", settings.CrawlSchedule)
	}
	if settings.FeishuWebhookURL == "" || !settings.FeishuConfigured {
		t.Fatalf("expected Feishu settings to be returned, got %#v", settings)
	}
	if settings.TimeZone != "Asia/Shanghai" {
		t.Fatalf("expected timezone setting, got %#v", settings)
	}
	if !settings.AutoDutyReport || settings.DutyReportTime != "18:30" || settings.TaskSLAHours != 6 {
		t.Fatalf("expected automation settings, got %#v", settings)
	}
	if !settings.AutoDiscovery || settings.DiscoveryHours != 8 {
		t.Fatalf("expected source discovery settings, got %#v", settings)
	}
}
