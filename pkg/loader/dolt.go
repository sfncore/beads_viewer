package loader

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

// DoltConfig holds connection parameters for a Dolt SQL server.
type DoltConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// DefaultDoltConfig returns the standard local Dolt server config.
func DefaultDoltConfig() DoltConfig {
	return DoltConfig{
		Host:     "127.0.0.1",
		Port:     3307,
		User:     "root",
		Password: "",
	}
}

// DSN returns the MySQL-compatible DSN for connecting to a specific database.
func (c DoltConfig) DSN(database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&timeout=5s&readTimeout=30s",
		c.User, c.Password, c.Host, c.Port, database)
}

// LoadIssuesFromDolt loads all non-ephemeral, non-deleted issues from a Dolt database.
// It queries issues, dependencies, labels, and comments, assembling them into
// model.Issue structs compatible with the rest of the bv pipeline.
func LoadIssuesFromDolt(ctx context.Context, config DoltConfig, database string) ([]model.Issue, error) {
	db, err := sql.Open("mysql", config.DSN(database))
	if err != nil {
		return nil, fmt.Errorf("connecting to dolt %s: %w", database, err)
	}
	defer db.Close()

	db.SetMaxOpenConns(2)
	db.SetConnMaxLifetime(30 * time.Second)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("pinging dolt %s: %w", database, err)
	}

	issues, err := queryIssues(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("querying issues from %s: %w", database, err)
	}

	if len(issues) == 0 {
		return nil, nil
	}

	// Build index for attaching related data
	issueMap := make(map[string]*model.Issue, len(issues))
	for i := range issues {
		issueMap[issues[i].ID] = &issues[i]
	}

	if err := attachDependencies(ctx, db, issueMap); err != nil {
		return nil, fmt.Errorf("querying dependencies from %s: %w", database, err)
	}

	if err := attachLabels(ctx, db, issueMap); err != nil {
		return nil, fmt.Errorf("querying labels from %s: %w", database, err)
	}

	if err := attachComments(ctx, db, issueMap); err != nil {
		return nil, fmt.Errorf("querying comments from %s: %w", database, err)
	}

	return issues, nil
}

// ListDoltDatabases returns the list of databases on the Dolt server,
// excluding system databases.
func ListDoltDatabases(ctx context.Context, config DoltConfig) ([]string, error) {
	db, err := sql.Open("mysql", config.DSN("information_schema"))
	if err != nil {
		return nil, fmt.Errorf("connecting to dolt: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("listing databases: %w", err)
	}
	defer rows.Close()

	skip := map[string]bool{
		"information_schema": true,
		"mysql":              true,
	}

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if !skip[name] {
			databases = append(databases, name)
		}
	}
	return databases, rows.Err()
}

func queryIssues(ctx context.Context, db *sql.DB) ([]model.Issue, error) {
	query := `SELECT
		id, title, description, design, acceptance_criteria, notes,
		status, priority, issue_type, assignee, estimated_minutes,
		created_at, updated_at, due_at, closed_at, external_ref,
		compaction_level, compacted_at, compacted_at_commit, original_size,
		source_repo
	FROM issues
	WHERE ephemeral = 0 AND deleted_at IS NULL`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []model.Issue
	for rows.Next() {
		var i model.Issue
		var (
			assignee          sql.NullString
			estimatedMinutes  sql.NullInt64
			dueAt             sql.NullTime
			closedAt          sql.NullTime
			externalRef       sql.NullString
			compactionLevel   sql.NullInt64
			compactedAt       sql.NullTime
			compactedAtCommit sql.NullString
			originalSize      sql.NullInt64
			design            sql.NullString
			acceptance        sql.NullString
			notes             sql.NullString
			sourceRepo        sql.NullString
			status            string
			issueType         string
		)

		err := rows.Scan(
			&i.ID, &i.Title, &i.Description, &design, &acceptance, &notes,
			&status, &i.Priority, &issueType, &assignee, &estimatedMinutes,
			&i.CreatedAt, &i.UpdatedAt, &dueAt, &closedAt, &externalRef,
			&compactionLevel, &compactedAt, &compactedAtCommit, &originalSize,
			&sourceRepo,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning issue row: %w", err)
		}

		i.Status = model.Status(strings.TrimSpace(strings.ToLower(status)))
		i.IssueType = model.IssueType(strings.TrimSpace(strings.ToLower(issueType)))
		i.Design = nullStr(design)
		i.AcceptanceCriteria = nullStr(acceptance)
		i.Notes = nullStr(notes)
		i.Assignee = nullStr(assignee)
		i.SourceRepo = nullStr(sourceRepo)

		if compactionLevel.Valid {
			i.CompactionLevel = int(compactionLevel.Int64)
		}
		if originalSize.Valid {
			i.OriginalSize = int(originalSize.Int64)
		}
		if estimatedMinutes.Valid {
			v := int(estimatedMinutes.Int64)
			i.EstimatedMinutes = &v
		}
		if dueAt.Valid {
			i.DueDate = &dueAt.Time
		}
		if closedAt.Valid {
			i.ClosedAt = &closedAt.Time
		}
		if externalRef.Valid && externalRef.String != "" {
			i.ExternalRef = &externalRef.String
		}
		if compactedAt.Valid {
			i.CompactedAt = &compactedAt.Time
		}
		if compactedAtCommit.Valid && compactedAtCommit.String != "" {
			i.CompactedAtCommit = &compactedAtCommit.String
		}

		issues = append(issues, i)
	}

	return issues, rows.Err()
}

func attachDependencies(ctx context.Context, db *sql.DB, issueMap map[string]*model.Issue) error {
	rows, err := db.QueryContext(ctx,
		"SELECT issue_id, depends_on_id, type, created_at, created_by FROM dependencies")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var d model.Dependency
		var depType string
		err := rows.Scan(&d.IssueID, &d.DependsOnID, &depType, &d.CreatedAt, &d.CreatedBy)
		if err != nil {
			return fmt.Errorf("scanning dependency row: %w", err)
		}
		d.Type = model.DependencyType(depType)

		if issue, ok := issueMap[d.IssueID]; ok {
			dep := d // copy for pointer
			issue.Dependencies = append(issue.Dependencies, &dep)
		}
	}
	return rows.Err()
}

func attachLabels(ctx context.Context, db *sql.DB, issueMap map[string]*model.Issue) error {
	rows, err := db.QueryContext(ctx, "SELECT issue_id, label FROM labels")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var issueID, label string
		if err := rows.Scan(&issueID, &label); err != nil {
			return fmt.Errorf("scanning label row: %w", err)
		}
		if issue, ok := issueMap[issueID]; ok {
			issue.Labels = append(issue.Labels, label)
		}
	}
	return rows.Err()
}

func attachComments(ctx context.Context, db *sql.DB, issueMap map[string]*model.Issue) error {
	rows, err := db.QueryContext(ctx, "SELECT id, issue_id, author, text, created_at FROM comments")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var c model.Comment
		err := rows.Scan(&c.ID, &c.IssueID, &c.Author, &c.Text, &c.CreatedAt)
		if err != nil {
			return fmt.Errorf("scanning comment row: %w", err)
		}
		if issue, ok := issueMap[c.IssueID]; ok {
			comment := c
			issue.Comments = append(issue.Comments, &comment)
		}
	}
	return rows.Err()
}

func nullStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}
