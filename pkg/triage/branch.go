// Package triage implements Dolt-branch-based triage proposals.
//
// The triage workflow:
//  1. Create a Dolt branch from main for a single rig
//  2. Load issues and run bv analysis (PageRank, betweenness, etc.)
//  3. Propose changes on the branch (reprioritize, flag stale, suggest deps)
//  4. Commit proposed changes to the branch
//  5. Generate a diff report (before/after for each change)
//  6. TUI or agent reviews and merges or discards
package triage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/Dicklesworthstone/beads_viewer/pkg/analysis"
	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

// BranchManager handles Dolt branch lifecycle for triage.
type BranchManager struct {
	config loader.DoltConfig
}

// NewBranchManager creates a manager for the given Dolt server.
func NewBranchManager(config loader.DoltConfig) *BranchManager {
	return &BranchManager{config: config}
}

// TriageBranch represents a triage branch with its proposed changes.
type TriageBranch struct {
	Database   string          // Dolt database name (rig)
	BranchName string          // Branch name (e.g., "triage-bv-20260216-1011")
	CreatedAt  time.Time       // When the branch was created
	Proposals  []ProposedChange // Changes proposed by analysis
	Report     *TriageReport   // Full analysis report
}

// ProposedChange represents a single proposed modification.
type ProposedChange struct {
	IssueID    string     `json:"issue_id"`
	ChangeType ChangeType `json:"change_type"`
	Field      string     `json:"field"`     // "priority", "status", "label", "dependency"
	OldValue   string     `json:"old_value"` // Before (human-readable)
	NewValue   string     `json:"new_value"` // After (human-readable)
	Reason     string     `json:"reason"`    // Why this change is proposed
	Score      float64    `json:"score"`     // Confidence/impact score
}

// ChangeType categorizes the kind of change.
type ChangeType string

const (
	ChangePriority   ChangeType = "priority"
	ChangeStatus     ChangeType = "status"
	ChangeLabel      ChangeType = "label_add"
	ChangeLabelDel   ChangeType = "label_del"
	ChangeDependency ChangeType = "dependency_add"
)

// TriageReport is the full output of a triage analysis run.
type TriageReport struct {
	Database        string           `json:"database"`
	BranchName      string           `json:"branch_name"`
	GeneratedAt     time.Time        `json:"generated_at"`
	IssueCount      int              `json:"issue_count"`
	OpenCount       int              `json:"open_count"`
	ProposalCount   int              `json:"proposal_count"`
	Proposals       []ProposedChange `json:"proposals"`
	CrossRigDeps    []CrossRigDep    `json:"cross_rig_deps,omitempty"`
	Triage          *analysis.TriageResult `json:"triage,omitempty"`
}

// CrossRigDep represents a dependency that crosses rig boundaries.
type CrossRigDep struct {
	IssueID     string `json:"issue_id"`
	DependsOnID string `json:"depends_on_id"`
	LocalRig    string `json:"local_rig"`
	RemoteRig   string `json:"remote_rig"`
}

// DiffEntry represents a single row change visible via dolt_diff.
type DiffEntry struct {
	IssueID   string `json:"issue_id"`
	DiffType  string `json:"diff_type"` // "modified", "added", "removed"
	Field     string `json:"field"`
	FromValue string `json:"from_value"`
	ToValue   string `json:"to_value"`
}

// CreateTriageBranch creates a new branch, runs analysis, proposes changes,
// and commits them. Returns the branch with all proposals.
func (m *BranchManager) CreateTriageBranch(ctx context.Context, database string) (*TriageBranch, error) {
	now := time.Now()
	branchName := fmt.Sprintf("triage-%s-%s", database, now.Format("20060102-1504"))

	db, err := sql.Open("mysql", m.config.DSN(database))
	if err != nil {
		return nil, fmt.Errorf("connecting to dolt %s: %w", database, err)
	}
	defer db.Close()

	// 1. Create the branch
	if _, err := db.ExecContext(ctx, "CALL DOLT_BRANCH(?)", branchName); err != nil {
		return nil, fmt.Errorf("creating branch %s: %w", branchName, err)
	}

	// 2. Load issues from main for analysis
	issues, err := loader.LoadIssuesFromDolt(ctx, m.config, database)
	if err != nil {
		// Clean up branch on failure
		db.ExecContext(ctx, "CALL DOLT_BRANCH('-D', ?)", branchName)
		return nil, fmt.Errorf("loading issues from %s: %w", database, err)
	}

	// 3. Run triage analysis
	triageResult := analysis.ComputeTriageWithOptions(issues, analysis.TriageOptions{
		WaitForPhase2: true,
		UseFastConfig: true,
	})

	// 4. Generate proposals from analysis
	proposals := generateProposals(issues, &triageResult, now)

	// 5. Apply proposals to the branch
	if len(proposals) > 0 {
		branchDB, err := sql.Open("mysql", m.config.DSN(database))
		if err != nil {
			return nil, fmt.Errorf("connecting to branch: %w", err)
		}
		defer branchDB.Close()

		// Switch to branch
		branchDBName := fmt.Sprintf("%s/%s", database, branchName)
		if _, err := branchDB.ExecContext(ctx, "USE `"+branchDBName+"`"); err != nil {
			return nil, fmt.Errorf("switching to branch %s: %w", branchName, err)
		}

		if err := applyProposals(ctx, branchDB, proposals); err != nil {
			return nil, fmt.Errorf("applying proposals: %w", err)
		}

		// Commit on the branch
		commitMsg := fmt.Sprintf("triage: %d proposed changes for %s", len(proposals), database)
		if _, err := branchDB.ExecContext(ctx, "CALL DOLT_COMMIT('-am', ?)", commitMsg); err != nil {
			return nil, fmt.Errorf("committing proposals: %w", err)
		}
	}

	// 6. Build report
	openCount := 0
	for _, issue := range issues {
		if issue.Status.IsOpen() || issue.Status == model.StatusBlocked {
			openCount++
		}
	}

	report := &TriageReport{
		Database:      database,
		BranchName:    branchName,
		GeneratedAt:   now,
		IssueCount:    len(issues),
		OpenCount:     openCount,
		ProposalCount: len(proposals),
		Proposals:     proposals,
		Triage:        &triageResult,
	}

	return &TriageBranch{
		Database:   database,
		BranchName: branchName,
		CreatedAt:  now,
		Proposals:  proposals,
		Report:     report,
	}, nil
}

// GetDiff returns the diff between main and the triage branch.
// Checks both the issues table (priority/status changes) and labels table (label add/remove).
func (m *BranchManager) GetDiff(ctx context.Context, database, branchName string) ([]DiffEntry, error) {
	db, err := sql.Open("mysql", m.config.DSN(database))
	if err != nil {
		return nil, fmt.Errorf("connecting to dolt: %w", err)
	}
	defer db.Close()

	branchDBName := fmt.Sprintf("%s/%s", database, branchName)
	if _, err := db.ExecContext(ctx, "USE `"+branchDBName+"`"); err != nil {
		return nil, fmt.Errorf("switching to branch: %w", err)
	}

	var diffs []DiffEntry

	// 1. Issue-level changes (priority, status)
	issueQuery := `SELECT to_id, from_priority, to_priority, from_status, to_status, diff_type
		FROM dolt_diff_issues
		WHERE from_commit = HASHOF('main') AND to_commit = HASHOF(?)`

	rows, err := db.QueryContext(ctx, issueQuery, branchName)
	if err != nil {
		return nil, fmt.Errorf("querying issue diff: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var issueID, diffType string
		var fromPri, toPri sql.NullInt64
		var fromStatus, toStatus sql.NullString

		if err := rows.Scan(&issueID, &fromPri, &toPri, &fromStatus, &toStatus, &diffType); err != nil {
			return nil, fmt.Errorf("scanning issue diff row: %w", err)
		}

		if fromPri.Valid && toPri.Valid && fromPri.Int64 != toPri.Int64 {
			diffs = append(diffs, DiffEntry{
				IssueID:   issueID,
				DiffType:  diffType,
				Field:     "priority",
				FromValue: fmt.Sprintf("P%d", fromPri.Int64),
				ToValue:   fmt.Sprintf("P%d", toPri.Int64),
			})
		}
		if fromStatus.Valid && toStatus.Valid && fromStatus.String != toStatus.String {
			diffs = append(diffs, DiffEntry{
				IssueID:   issueID,
				DiffType:  diffType,
				Field:     "status",
				FromValue: fromStatus.String,
				ToValue:   toStatus.String,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2. Label changes (add/remove)
	labelQuery := `SELECT to_issue_id, from_label, to_label, diff_type
		FROM dolt_diff_labels
		WHERE from_commit = HASHOF('main') AND to_commit = HASHOF(?)`

	lrows, err := db.QueryContext(ctx, labelQuery, branchName)
	if err != nil {
		// labels table might not exist in all databases — not fatal
		return diffs, nil
	}
	defer lrows.Close()

	for lrows.Next() {
		var issueID, diffType string
		var fromLabel, toLabel sql.NullString

		if err := lrows.Scan(&issueID, &fromLabel, &toLabel, &diffType); err != nil {
			return nil, fmt.Errorf("scanning label diff row: %w", err)
		}

		switch diffType {
		case "added":
			diffs = append(diffs, DiffEntry{
				IssueID:   issueID,
				DiffType:  diffType,
				Field:     "label",
				FromValue: "",
				ToValue:   toLabel.String,
			})
		case "removed":
			diffs = append(diffs, DiffEntry{
				IssueID:   issueID,
				DiffType:  diffType,
				Field:     "label",
				FromValue: fromLabel.String,
				ToValue:   "",
			})
		}
	}

	return diffs, lrows.Err()
}

// MergeBranch merges a triage branch into main.
func (m *BranchManager) MergeBranch(ctx context.Context, database, branchName string) error {
	db, err := sql.Open("mysql", m.config.DSN(database))
	if err != nil {
		return fmt.Errorf("connecting to dolt: %w", err)
	}
	defer db.Close()

	// Commit any dirty state on main first
	db.ExecContext(ctx, "CALL DOLT_COMMIT('-am', 'auto: commit working state before triage merge')")

	// Merge the branch
	if _, err := db.ExecContext(ctx, "CALL DOLT_MERGE(?)", branchName); err != nil {
		return fmt.Errorf("merging branch %s: %w", branchName, err)
	}

	return nil
}

// DeleteBranch removes a triage branch without merging.
func (m *BranchManager) DeleteBranch(ctx context.Context, database, branchName string) error {
	db, err := sql.Open("mysql", m.config.DSN(database))
	if err != nil {
		return fmt.Errorf("connecting to dolt: %w", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(ctx, "CALL DOLT_BRANCH('-D', ?)", branchName); err != nil {
		return fmt.Errorf("deleting branch %s: %w", branchName, err)
	}
	return nil
}

// ListTriageBranches lists all triage branches for a database.
func (m *BranchManager) ListTriageBranches(ctx context.Context, database string) ([]string, error) {
	db, err := sql.Open("mysql", m.config.DSN(database))
	if err != nil {
		return nil, fmt.Errorf("connecting to dolt: %w", err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, "SELECT name FROM dolt_branches WHERE name LIKE 'triage-%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		branches = append(branches, name)
	}
	return branches, rows.Err()
}

// generateProposals analyzes issues and creates proposed changes.
func generateProposals(issues []model.Issue, triage *analysis.TriageResult, now time.Time) []ProposedChange {
	var proposals []ProposedChange

	// Index issues by ID for lookup
	issueMap := make(map[string]*model.Issue, len(issues))
	for i := range issues {
		issueMap[issues[i].ID] = &issues[i]
	}

	// Track already-labeled issues
	hasLabel := make(map[string]map[string]bool)
	for i := range issues {
		labels := make(map[string]bool)
		for _, l := range issues[i].Labels {
			labels[l] = true
		}
		hasLabel[issues[i].ID] = labels
	}

	// Proposal 1: Reprioritize based on triage score
	// Any open issue in recommendations with a score suggesting higher priority
	for _, rec := range triage.Recommendations {
		issue, ok := issueMap[rec.ID]
		if !ok || issue.Status.IsClosed() {
			continue
		}

		suggestedPriority := suggestPriority(rec.Score, issue.Priority)
		if suggestedPriority != issue.Priority {
			proposals = append(proposals, ProposedChange{
				IssueID:    issue.ID,
				ChangeType: ChangePriority,
				Field:      "priority",
				OldValue:   fmt.Sprintf("P%d", issue.Priority),
				NewValue:   fmt.Sprintf("P%d", suggestedPriority),
				Reason:     fmt.Sprintf("Triage score %.3f suggests higher priority (unblocks %d items)", rec.Score, len(rec.UnblocksIDs)),
				Score:      rec.Score,
			})
		}
	}

	// Proposal 2: Flag stale issues (7+ days for in-progress, 10+ for open)
	for i := range issues {
		issue := &issues[i]
		if issue.Status.IsClosed() || issue.Status == model.StatusDeferred || issue.Status == model.StatusPinned {
			continue
		}
		if hasLabel[issue.ID]["stale"] {
			continue // already flagged
		}

		daysSince := int(now.Sub(issue.UpdatedAt).Hours() / 24)
		threshold := 10
		if issue.Status == model.StatusInProgress {
			threshold = 7 // Stricter for in-progress
		}

		if daysSince >= threshold {
			proposals = append(proposals, ProposedChange{
				IssueID:    issue.ID,
				ChangeType: ChangeLabel,
				Field:      "label",
				OldValue:   "",
				NewValue:   "stale",
				Reason:     fmt.Sprintf("%s with no activity for %d days", issue.Status, daysSince),
				Score:      float64(daysSince) / 30.0,
			})
		}
	}

	// Proposal 3: High-blocker issues that aren't labeled as blockers
	for _, blocker := range triage.BlockersToClear {
		issue, ok := issueMap[blocker.ID]
		if !ok || issue.Status.IsClosed() {
			continue
		}
		if hasLabel[issue.ID]["critical-path"] {
			continue
		}
		if blocker.UnblocksCount >= 3 {
			proposals = append(proposals, ProposedChange{
				IssueID:    issue.ID,
				ChangeType: ChangeLabel,
				Field:      "label",
				OldValue:   "",
				NewValue:   "critical-path",
				Reason:     fmt.Sprintf("Unblocks %d downstream items", blocker.UnblocksCount),
				Score:      float64(blocker.UnblocksCount) / 10.0,
			})
		}
	}

	// Proposal 4: Quick wins that aren't labeled
	for _, qw := range triage.QuickWins {
		issue, ok := issueMap[qw.ID]
		if !ok || issue.Status.IsClosed() {
			continue
		}
		if hasLabel[issue.ID]["quick-win"] {
			continue
		}
		proposals = append(proposals, ProposedChange{
			IssueID:    issue.ID,
			ChangeType: ChangeLabel,
			Field:      "label",
			OldValue:   "",
			NewValue:   "quick-win",
			Reason:     fmt.Sprintf("Quick win: %s (score %.2f)", qw.Reason, qw.Score),
			Score:      qw.Score,
		})
	}

	return proposals
}

// suggestPriority maps triage score to a suggested priority.
func suggestPriority(score float64, currentPriority int) int {
	var suggested int
	switch {
	case score >= 0.25:
		suggested = 0 // P0
	case score >= 0.18:
		suggested = 1 // P1
	case score >= 0.12:
		suggested = 2 // P2
	default:
		return currentPriority // Don't downgrade
	}
	// Only suggest upgrading, never downgrading
	if suggested < currentPriority {
		return suggested
	}
	return currentPriority
}

// applyProposals writes proposed changes to the branch database.
func applyProposals(ctx context.Context, db *sql.DB, proposals []ProposedChange) error {
	for _, p := range proposals {
		switch p.ChangeType {
		case ChangePriority:
			var newPri int
			fmt.Sscanf(p.NewValue, "P%d", &newPri)
			_, err := db.ExecContext(ctx,
				"UPDATE issues SET priority = ?, updated_at = NOW() WHERE id = ?",
				newPri, p.IssueID)
			if err != nil {
				return fmt.Errorf("updating priority for %s: %w", p.IssueID, err)
			}

		case ChangeStatus:
			_, err := db.ExecContext(ctx,
				"UPDATE issues SET status = ?, updated_at = NOW() WHERE id = ?",
				strings.ToLower(p.NewValue), p.IssueID)
			if err != nil {
				return fmt.Errorf("updating status for %s: %w", p.IssueID, err)
			}

		case ChangeLabel:
			_, err := db.ExecContext(ctx,
				"INSERT IGNORE INTO labels (issue_id, label) VALUES (?, ?)",
				p.IssueID, p.NewValue)
			if err != nil {
				return fmt.Errorf("adding label for %s: %w", p.IssueID, err)
			}

		case ChangeLabelDel:
			_, err := db.ExecContext(ctx,
				"DELETE FROM labels WHERE issue_id = ? AND label = ?",
				p.IssueID, p.OldValue)
			if err != nil {
				return fmt.Errorf("removing label for %s: %w", p.IssueID, err)
			}
		}
	}
	return nil
}
