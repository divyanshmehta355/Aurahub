package db

import (
	_ "embed"
	"fmt"
	"log"
	"strings"

	"gorm.io/gorm"
)

//go:embed schema.sql
var SchemaSQL string

// EnsureSchema idempotently initializes the database schema, enums, extensions, and indexes.
// It is safe to call on every server startup:
// - If tables and types are already present, it verifies in milliseconds and returns nil.
// - If starting against a fresh/empty database, it sets up the entire schema automatically.
func EnsureSchema(database *gorm.DB) error {
	log.Println("[Database] Verifying database schema and enums...")

	statements := SplitSQLStatements(SchemaSQL)
	for _, stmt := range statements {
		if err := database.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed executing schema statement: %w\nQuery: %s", err, stmt)
		}
	}

	log.Println("[Database] Database schema verified successfully.")
	return nil
}

// SplitSQLStatements parses raw SQL into executable statements while preserving DO $$ ... $$ blocks.
func SplitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inDollarQuote := false

	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		if strings.Count(line, "$$")%2 == 1 {
			inDollarQuote = !inDollarQuote
		}
		current.WriteString(line)
		current.WriteString("\n")

		if !inDollarQuote && strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimSuffix(stmt, ";")
			stmt = strings.TrimSpace(stmt)
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	leftover := strings.TrimSpace(current.String())
	if leftover != "" {
		statements = append(statements, leftover)
	}

	return statements
}
