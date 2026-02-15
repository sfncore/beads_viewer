package loader

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

// These tests require a running Dolt server on localhost:3307.
// They are skipped if the server is not available.

func skipIfNoDolt(t *testing.T) DoltConfig {
	t.Helper()
	config := DefaultDoltConfig()
	db, err := sql.Open("mysql", config.DSN("information_schema"))
	if err != nil {
		t.Skipf("skipping dolt test: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skipping dolt test: server not reachable: %v", err)
	}
	return config
}

func TestListDoltDatabases(t *testing.T) {
	config := skipIfNoDolt(t)

	ctx := context.Background()
	databases, err := ListDoltDatabases(ctx, config)
	if err != nil {
		t.Fatalf("ListDoltDatabases() error = %v", err)
	}

	if len(databases) == 0 {
		t.Fatal("expected at least one database")
	}

	// Should contain bv
	found := false
	for _, db := range databases {
		if db == "bv" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'bv' in databases, got: %v", databases)
	}

	// Should not contain system databases
	for _, db := range databases {
		if db == "information_schema" || db == "mysql" {
			t.Errorf("unexpected system database in list: %s", db)
		}
	}

	t.Logf("found %d databases: %v", len(databases), databases)
}

func TestLoadIssuesFromDolt(t *testing.T) {
	config := skipIfNoDolt(t)

	ctx := context.Background()
	issues, err := LoadIssuesFromDolt(ctx, config, "bv")
	if err != nil {
		t.Fatalf("LoadIssuesFromDolt() error = %v", err)
	}

	if len(issues) == 0 {
		t.Fatal("expected at least one issue from bv database")
	}

	t.Logf("loaded %d issues from bv", len(issues))

	// Check that issues have required fields
	for _, issue := range issues {
		if issue.ID == "" {
			t.Error("issue has empty ID")
		}
		if issue.Title == "" {
			t.Errorf("issue %s has empty title", issue.ID)
		}
		if !issue.Status.IsValid() {
			t.Errorf("issue %s has invalid status: %q", issue.ID, issue.Status)
		}
		if !issue.IssueType.IsValid() {
			t.Errorf("issue %s has invalid type: %q", issue.ID, issue.IssueType)
		}
	}

	// Check that dependencies were attached
	depsFound := 0
	for _, issue := range issues {
		depsFound += len(issue.Dependencies)
	}
	t.Logf("total dependencies attached: %d", depsFound)

	// Check that labels were attached
	labelsFound := 0
	for _, issue := range issues {
		labelsFound += len(issue.Labels)
	}
	t.Logf("total labels attached: %d", labelsFound)

	// Check that comments were attached
	commentsFound := 0
	for _, issue := range issues {
		commentsFound += len(issue.Comments)
	}
	t.Logf("total comments attached: %d", commentsFound)
}

func TestLoadIssuesFromDolt_MultipleDBs(t *testing.T) {
	config := skipIfNoDolt(t)

	ctx := context.Background()

	// Load from multiple databases to verify cross-rig loading works
	databases := []string{"bv", "hq", "sfgastown"}
	for _, db := range databases {
		issues, err := LoadIssuesFromDolt(ctx, config, db)
		if err != nil {
			t.Errorf("LoadIssuesFromDolt(%s) error = %v", db, err)
			continue
		}
		t.Logf("  %s: %d issues", db, len(issues))
	}
}
