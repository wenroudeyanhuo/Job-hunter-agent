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
	return nil
}
