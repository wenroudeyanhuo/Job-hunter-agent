package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Company struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Category     string    `json:"category"`
	Enabled      bool      `json:"enabled"`
	Priority     int       `json:"priority"`
	Notes        string    `json:"notes"`
	SourceCount  int       `json:"source_count"`
	HealthyCount int       `json:"healthy_count"`
	WarningCount int       `json:"warning_count"`
	BrokenCount  int       `json:"broken_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CompanyInput struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Enabled  bool   `json:"enabled"`
	Priority int    `json:"priority"`
	Notes    string `json:"notes"`
}

func (r *Repository) CreateOrUpdateCompany(ctx context.Context, input CompanyInput) (Company, error) {
	input = normalizeCompanyInput(input)
	var existingID int64
	err := r.db.QueryRowContext(ctx, `SELECT id FROM companies WHERE lower(trim(name)) = lower(trim(?))`, input.Name).Scan(&existingID)
	if err == nil {
		_, err = r.db.ExecContext(ctx, `
			UPDATE companies
			SET category = ?, priority = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, input.Category, input.Priority, input.Notes, existingID)
		if err != nil {
			return Company{}, fmt.Errorf("update company %q: %w", input.Name, err)
		}
		return r.GetCompany(ctx, existingID)
	}
	if err != sql.ErrNoRows {
		return Company{}, fmt.Errorf("find company %q: %w", input.Name, err)
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO companies (name, category, enabled, priority, notes)
		VALUES (?, ?, ?, ?, ?)
	`, input.Name, input.Category, boolToInt(input.Enabled), input.Priority, input.Notes)
	if err != nil {
		return Company{}, fmt.Errorf("insert company: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Company{}, fmt.Errorf("read company id: %w", err)
	}
	return r.GetCompany(ctx, id)
}

func (r *Repository) GetCompany(ctx context.Context, id int64) (Company, error) {
	row := r.db.QueryRowContext(ctx, selectCompanySQL()+` WHERE c.id = ? GROUP BY c.id`, id)
	company, err := scanCompany(row)
	if err != nil {
		return Company{}, fmt.Errorf("get company %d: %w", id, err)
	}
	return company, nil
}

func (r *Repository) ListCompanies(ctx context.Context) ([]Company, error) {
	rows, err := r.db.QueryContext(ctx, selectCompanySQL()+` GROUP BY c.id ORDER BY c.enabled DESC, c.priority DESC, c.updated_at DESC, c.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	out := []Company{}
	for rows.Next() {
		company, err := scanCompany(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, company)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate companies: %w", err)
	}
	return out, nil
}

func (r *Repository) UpdateCompanyEnabled(ctx context.Context, id int64, enabled bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin company toggle: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		UPDATE companies
		SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, boolToInt(enabled), id)
	if err != nil {
		return fmt.Errorf("update company enabled: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read company rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE job_sources
		SET enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE company_id = ?
	`, boolToInt(enabled), id); err != nil {
		return fmt.Errorf("update company sources enabled: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit company toggle: %w", err)
	}
	return nil
}

func normalizeCompanyInput(input CompanyInput) CompanyInput {
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Category == "" {
		input.Category = "general"
	}
	return input
}

func selectCompanySQL() string {
	return `
		SELECT c.id, c.name, c.category, c.enabled, c.priority, c.notes,
			COUNT(s.id) AS source_count,
			COALESCE(SUM(CASE WHEN s.health_status = 'healthy' THEN 1 ELSE 0 END), 0) AS healthy_count,
			COALESCE(SUM(CASE WHEN s.health_status = 'warning' THEN 1 ELSE 0 END), 0) AS warning_count,
			COALESCE(SUM(CASE WHEN s.health_status = 'broken' THEN 1 ELSE 0 END), 0) AS broken_count,
			c.created_at, c.updated_at
		FROM companies c
		LEFT JOIN job_sources s ON s.company_id = c.id`
}

func scanCompany(scanner jobScanner) (Company, error) {
	var company Company
	var enabled int
	if err := scanner.Scan(
		&company.ID,
		&company.Name,
		&company.Category,
		&enabled,
		&company.Priority,
		&company.Notes,
		&company.SourceCount,
		&company.HealthyCount,
		&company.WarningCount,
		&company.BrokenCount,
		&company.CreatedAt,
		&company.UpdatedAt,
	); err != nil {
		return Company{}, fmt.Errorf("scan company: %w", err)
	}
	company.Enabled = enabled == 1
	return company, nil
}
