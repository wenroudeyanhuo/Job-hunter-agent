package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenMigratesApplicationPlanColumns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job-hunter-agent.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw sqlite: %v", err)
	}
	if _, err := conn.Exec(`
		CREATE TABLE application_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id INTEGER NOT NULL UNIQUE,
			status TEXT NOT NULL DEFAULT 'prepare',
			priority INTEGER NOT NULL DEFAULT 0,
			next_action TEXT NOT NULL DEFAULT '',
			checklist TEXT NOT NULL DEFAULT '[]',
			blocker_notes TEXT NOT NULL DEFAULT '',
			target_apply_date TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		t.Fatalf("create old application_plans table: %v", err)
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close raw sqlite: %v", err)
	}

	migrated, err := Open(path)
	if err != nil {
		t.Fatalf("open migrated db: %v", err)
	}
	defer migrated.Close()

	columns := tableColumns(t, migrated, "application_plans")
	for _, name := range []string{"resume_version", "draft_notes", "follow_up_date"} {
		if !columns[name] {
			t.Fatalf("expected migrated column %q in application_plans, got %#v", name, columns)
		}
	}
}

func tableColumns(t *testing.T, conn *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("inspect %s columns: %v", table, err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan %s column: %v", table, err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate %s columns: %v", table, err)
	}
	return columns
}
