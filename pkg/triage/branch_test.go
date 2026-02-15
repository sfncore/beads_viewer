package triage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
)

func skipIfNoDolt(t *testing.T) loader.DoltConfig {
	t.Helper()
	config := loader.DefaultDoltConfig()
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

func TestCreateTriageBranch(t *testing.T) {
	config := skipIfNoDolt(t)
	mgr := NewBranchManager(config)
	ctx := context.Background()

	branch, err := mgr.CreateTriageBranch(ctx, "bv")
	if err != nil {
		t.Fatalf("CreateTriageBranch() error = %v", err)
	}

	t.Logf("Branch: %s", branch.BranchName)
	t.Logf("Issues: %d, Open: %d", branch.Report.IssueCount, branch.Report.OpenCount)
	t.Logf("Proposals: %d", len(branch.Proposals))

	for i, p := range branch.Proposals {
		if i >= 10 {
			t.Logf("  ... and %d more", len(branch.Proposals)-10)
			break
		}
		t.Logf("  [%s] %s: %s -> %s (score=%.3f) %s",
			p.ChangeType, p.IssueID, p.OldValue, p.NewValue, p.Score, p.Reason)
	}

	// Get the diff
	diffs, err := mgr.GetDiff(ctx, "bv", branch.BranchName)
	if err != nil {
		t.Fatalf("GetDiff() error = %v", err)
	}
	t.Logf("Diffs: %d", len(diffs))
	for i, d := range diffs {
		if i >= 5 {
			break
		}
		t.Logf("  %s %s: %s -> %s (%s)", d.IssueID, d.Field, d.FromValue, d.ToValue, d.DiffType)
	}

	// Clean up
	if err := mgr.DeleteBranch(ctx, "bv", branch.BranchName); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}
	t.Log("Branch cleaned up successfully")
}

func TestCreateTriageBranchFrankenterm(t *testing.T) {
	config := skipIfNoDolt(t)
	mgr := NewBranchManager(config)
	ctx := context.Background()

	branch, err := mgr.CreateTriageBranch(ctx, "frankenterm")
	if err != nil {
		t.Fatalf("CreateTriageBranch(frankenterm) error = %v", err)
	}

	t.Logf("Branch: %s", branch.BranchName)
	t.Logf("Issues: %d, Open: %d", branch.Report.IssueCount, branch.Report.OpenCount)
	t.Logf("Proposals: %d", len(branch.Proposals))

	for i, p := range branch.Proposals {
		if i >= 15 {
			t.Logf("  ... and %d more", len(branch.Proposals)-15)
			break
		}
		t.Logf("  [%s] %s: %s -> %s (score=%.3f) %s",
			p.ChangeType, p.IssueID, p.OldValue, p.NewValue, p.Score, p.Reason)
	}

	// Get the diff
	diffs, err := mgr.GetDiff(ctx, "frankenterm", branch.BranchName)
	if err != nil {
		t.Fatalf("GetDiff() error = %v", err)
	}
	t.Logf("Diffs from Dolt: %d", len(diffs))
	for i, d := range diffs {
		if i >= 10 {
			break
		}
		t.Logf("  %s %s: %s -> %s (%s)", d.IssueID, d.Field, d.FromValue, d.ToValue, d.DiffType)
	}

	// Clean up
	if err := mgr.DeleteBranch(ctx, "frankenterm", branch.BranchName); err != nil {
		t.Fatalf("DeleteBranch() error = %v", err)
	}
	t.Log("Branch cleaned up")
}

func TestListTriageBranches(t *testing.T) {
	config := skipIfNoDolt(t)
	mgr := NewBranchManager(config)
	ctx := context.Background()

	branches, err := mgr.ListTriageBranches(ctx, "bv")
	if err != nil {
		t.Fatalf("ListTriageBranches() error = %v", err)
	}
	t.Logf("Found %d triage branches", len(branches))
	for _, b := range branches {
		t.Logf("  %s", b)
	}
}
