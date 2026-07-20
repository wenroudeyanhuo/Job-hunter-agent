package db

import (
	"database/sql"
	"embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaFS embed.FS

func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := applySchema(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return conn, nil
}

func applySchema(conn *sql.DB) error {
	content, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	statements := strings.Split(string(content), ";")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := conn.Exec(statement); err != nil {
			return fmt.Errorf("apply schema statement %q: %w", statement, err)
		}
	}
	if err := ensureJobSourceColumns(conn); err != nil {
		return err
	}
	if err := ensureCompanyColumns(conn); err != nil {
		return err
	}
	return nil
}

func ensureJobSourceColumns(conn *sql.DB) error {
	columns := map[string]string{
		"company_id":           "INTEGER NULL",
		"category":             "TEXT NOT NULL DEFAULT 'general'",
		"health_status":        "TEXT NOT NULL DEFAULT 'unknown'",
		"health_reason":        "TEXT NOT NULL DEFAULT ''",
		"consecutive_failures": "INTEGER NOT NULL DEFAULT 0",
		"last_success_at":      "TIMESTAMP NULL",
		"last_failure_at":      "TIMESTAMP NULL",
		"last_found_count":     "INTEGER NOT NULL DEFAULT 0",
	}
	existing := map[string]bool{}
	rows, err := conn.Query(`PRAGMA table_info(job_sources)`)
	if err != nil {
		return fmt.Errorf("inspect job_sources columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan job_sources column: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate job_sources columns: %w", err)
	}
	for name, definition := range columns {
		if existing[name] {
			continue
		}
		if _, err := conn.Exec(fmt.Sprintf("ALTER TABLE job_sources ADD COLUMN %s %s", name, definition)); err != nil {
			return fmt.Errorf("add job_sources.%s: %w", name, err)
		}
	}
	return nil
}

func ensureCompanyColumns(conn *sql.DB) error {
	columns := map[string]string{
		"enabled": "INTEGER NOT NULL DEFAULT 1",
	}
	existing := map[string]bool{}
	rows, err := conn.Query(`PRAGMA table_info(companies)`)
	if err != nil {
		return fmt.Errorf("inspect companies columns: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan companies column: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate companies columns: %w", err)
	}
	for name, definition := range columns {
		if existing[name] {
			continue
		}
		if _, err := conn.Exec(fmt.Sprintf("ALTER TABLE companies ADD COLUMN %s %s", name, definition)); err != nil {
			return fmt.Errorf("add companies.%s: %w", name, err)
		}
	}
	return nil
}
