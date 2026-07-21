package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const appSettingsKey = "app_settings"

type Settings struct {
	TargetCities          []string   `json:"target_cities"`
	TargetDirections      []string   `json:"target_directions"`
	ExcludedKeywords      []string   `json:"excluded_keywords"`
	CrawlSchedule         []string   `json:"crawl_schedule"`
	FeishuWebhookURL      string     `json:"feishu_webhook_url"`
	AutoDutyReportEnabled bool       `json:"auto_duty_report_enabled"`
	DutyReportTime        string     `json:"duty_report_time"`
	TaskSLAHours          int        `json:"task_sla_hours"`
	LastDutyReportSentAt  *time.Time `json:"last_duty_report_sent_at,omitempty"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

func DefaultSettings() Settings {
	return Settings{
		TargetCities:     []string{"Shenzhen"},
		TargetDirections: []string{"frontend", "backend", "java", "go", "algorithm", "ai_application"},
		ExcludedKeywords: []string{"outsourcing", "training", "bootcamp"},
		CrawlSchedule:    []string{"09:00", "12:00", "18:00"},
		DutyReportTime:   "18:00",
		TaskSLAHours:     24,
		UpdatedAt:        time.Now().UTC(),
	}
}

func (r *Repository) GetSettings(ctx context.Context) (Settings, error) {
	var raw string
	var updatedAt time.Time
	err := r.db.QueryRowContext(ctx, `
		SELECT value, updated_at
		FROM settings
		WHERE key = ?
	`, appSettingsKey).Scan(&raw, &updatedAt)
	if err != nil {
		return DefaultSettings(), nil
	}

	settings := DefaultSettings()
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	settings.UpdatedAt = updatedAt
	return normalizeSettings(settings), nil
}

func (r *Repository) SaveSettings(ctx context.Context, settings Settings) (Settings, error) {
	settings = normalizeSettings(settings)
	settings.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(settings)
	if err != nil {
		return Settings{}, fmt.Errorf("encode settings: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			value = excluded.value,
			updated_at = excluded.updated_at
	`, appSettingsKey, string(data), settings.UpdatedAt)
	if err != nil {
		return Settings{}, fmt.Errorf("save settings: %w", err)
	}
	return r.GetSettings(ctx)
}

func normalizeSettings(settings Settings) Settings {
	defaults := DefaultSettings()
	settings.TargetCities = cleanStringList(settings.TargetCities)
	if len(settings.TargetCities) == 0 {
		settings.TargetCities = defaults.TargetCities
	}
	settings.TargetDirections = cleanStringList(settings.TargetDirections)
	if len(settings.TargetDirections) == 0 {
		settings.TargetDirections = defaults.TargetDirections
	}
	settings.ExcludedKeywords = cleanStringList(settings.ExcludedKeywords)
	if len(settings.ExcludedKeywords) == 0 {
		settings.ExcludedKeywords = defaults.ExcludedKeywords
	}
	settings.CrawlSchedule = cleanStringList(settings.CrawlSchedule)
	if len(settings.CrawlSchedule) == 0 {
		settings.CrawlSchedule = defaults.CrawlSchedule
	}
	settings.FeishuWebhookURL = strings.TrimSpace(settings.FeishuWebhookURL)
	settings.DutyReportTime = strings.TrimSpace(settings.DutyReportTime)
	if settings.DutyReportTime == "" {
		settings.DutyReportTime = defaults.DutyReportTime
	}
	if settings.TaskSLAHours <= 0 {
		settings.TaskSLAHours = defaults.TaskSLAHours
	}
	return settings
}

func cleanStringList(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}
