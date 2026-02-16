# Dolt Integration for BV

BV can read beads data directly from the Dolt SQL server and run triage analysis on Dolt branches. This document covers the architecture, usage, and implementation details.

## Architecture

```
Dolt SQL Server (127.0.0.1:3307)
  |
  |-- 17 databases (one per rig: bv, hq, sfgastown, frankenterm, ...)
  |     |
  |     |-- issues table (55 columns)
  |     |-- dependencies table
  |     |-- labels table
  |     |-- comments table
  |     |-- dolt_branches (system table)
  |     |-- dolt_diff_* (system tables for branch diffs)
  |
  |-- Branch-based triage workflow:
        1. Create branch: triage-<rig>-<timestamp>
        2. Load issues from main, run bv analysis
        3. Write proposed changes to branch
        4. Commit on branch
        5. Query diffs (dolt_diff_issues, dolt_diff_labels)
        6. Review (TUI modal or JSON output)
        7. Merge or discard
```

## Dolt Branch Mechanics

Key findings from investigation:

- **Copy-on-write**: Branches cost zero disk space at creation (~140ms)
- **Full snapshot**: Each branch has ALL tables, not just a subset
- **Isolation**: Can query main and branch simultaneously via `USE db/branch`
- **Diff tables**: `dolt_diff_<table>` shows before/after for every column between commits
- **Access control**: `dolt_branch_control` table supports per-branch, per-user ACLs
- **Merge**: Requires clean working state on main (commit dirty state first)

## CLI Usage

### Read beads from Dolt (workspace mode)

```bash
# Load all rigs from Dolt instead of stale JSONL files
bv --workspace ~/gt/.beads/routes.jsonl --dolt

# Same, with robot JSON output
bv --workspace ~/gt/.beads/routes.jsonl --dolt --robot-triage
```

### Run triage analysis on a single rig

```bash
# Analyze frankenterm, output proposals as JSON
bv --dolt --triage-rig frankenterm

# Analyze and auto-merge proposals
bv --dolt --triage-rig frankenterm --triage-merge

# Analyze bv
bv --dolt --triage-rig bv
```

### Output format

The `--triage-rig` command outputs JSON to stdout:

```json
{
  "database": "frankenterm",
  "branch": "triage-frankenterm-20260216-1036",
  "created_at": "2026-02-16T10:36:43Z",
  "issues": 1563,
  "open_issues": 193,
  "proposals": [
    {
      "issue_id": "wa-nu4",
      "change_type": "label_add",
      "field": "label",
      "old_value": "",
      "new_value": "stale",
      "reason": "in_progress with no activity for 11 days",
      "score": 0.367
    }
  ],
  "diffs": [
    {
      "issue_id": "wa-nu4",
      "diff_type": "added",
      "field": "label",
      "from_value": "",
      "to_value": "stale"
    }
  ]
}
```

Status messages go to stderr so JSON stays clean on stdout.

## TUI Integration

The `TriageDiffModal` displays triage proposals in a scrollable review interface:

- **j/k**: Navigate proposals
- **Y**: Accept (merge branch to main)
- **N**: Reject (discard branch)
- **Esc**: Close without action

Each proposal shows a colored badge for change type (PRI/STS/LBL/DEP), the issue ID, the value change, and on the selected row, the reason and confidence score.

The modal is triggered by sending a `TriageDiffLoadedMsg` to the TUI model, or by calling `model.ShowTriageDiff(branch, diffs)` directly.

## Proposal Types

The triage engine generates four types of proposals:

### 1. Priority Upgrades (`ChangePriority`)
Based on triage score thresholds:
- Score >= 0.25 -> suggest P0
- Score >= 0.18 -> suggest P1
- Score >= 0.12 -> suggest P2
- Only upgrades, never downgrades

### 2. Stale Issue Flagging (`ChangeLabel: stale`)
- In-progress issues with no activity for 7+ days
- Open issues with no activity for 10+ days
- Skips closed, deferred, and pinned issues
- Skips issues already labeled "stale"

### 3. Critical-Path Labeling (`ChangeLabel: critical-path`)
- Issues that unblock 3+ downstream items (from `BlockersToClear`)
- Skips issues already labeled "critical-path"
- Score is proportional to unblock count

### 4. Quick-Win Labeling (`ChangeLabel: quick-win`)
- Issues identified by triage analysis as quick wins
- Low complexity, high impact opportunities
- Skips issues already labeled "quick-win"

## Package Structure

### `pkg/loader/dolt.go` — Dolt SQL Loader

```go
type DoltConfig struct {
    Host     string // default: 127.0.0.1
    Port     int    // default: 3307
    User     string // default: root
    Password string // default: ""
}

func DefaultDoltConfig() DoltConfig
func (c DoltConfig) DSN(database string) string
func LoadIssuesFromDolt(ctx, config, database) ([]model.Issue, error)
func ListDoltDatabases(ctx, config) ([]string, error)
```

Loads issues, dependencies, labels, and comments from Dolt SQL in parallel queries. Handles NULL columns with `sql.NullInt64`/`sql.NullString`/`sql.NullTime`.

### `pkg/triage/branch.go` — Branch Triage Engine

```go
type BranchManager struct { config loader.DoltConfig }

func NewBranchManager(config) *BranchManager
func (m *BranchManager) CreateTriageBranch(ctx, database) (*TriageBranch, error)
func (m *BranchManager) GetDiff(ctx, database, branchName) ([]DiffEntry, error)
func (m *BranchManager) MergeBranch(ctx, database, branchName) error
func (m *BranchManager) DeleteBranch(ctx, database, branchName) error
func (m *BranchManager) ListTriageBranches(ctx, database) ([]string, error)
```

### `pkg/ui/triage_diff_modal.go` — TUI Review Modal

```go
type TriageDiffModal struct { ... }

func NewTriageDiffModal(branch, diffs, theme) TriageDiffModal
func (m TriageDiffModal) Update(msg) (TriageDiffModal, tea.Cmd)
func (m TriageDiffModal) View() string
func (m TriageDiffModal) CenterModal(width, height) string
func (m TriageDiffModal) IsAccepted() bool
func (m TriageDiffModal) IsRejected() bool
```

### `pkg/workspace/loader.go` — Dolt-aware Workspace Loading

```go
func (l *AggregateLoader) SetDoltConfig(config *loader.DoltConfig)
func LoadAllFromConfigWithDolt(ctx, configPath, doltConfig) ([]model.Issue, []LoadResult, error)
```

When `DoltConfig` is set and a repo has `DoltDatabase` configured, issues are loaded from Dolt SQL instead of JSONL files.

## Test Results (Live Data)

| Rig | Issues | Open | Proposals | Types |
|-----|--------|------|-----------|-------|
| bv | 743 | 4 | 5 | 1 stale, 4 quick-wins |
| frankenterm | 1563 | 193 | 11 | 2 stale, 4 critical-path, 5 quick-wins |

All proposals are confirmed via `dolt_diff_labels` queries showing exact before/after state.

## Dolt Connection Details

```
Host: 127.0.0.1
Port: 3307
User: root
Password: (empty)
TLS: disabled
Data dir: ~/gt/.dolt-data/
```

CLI access:
```bash
dolt --host 127.0.0.1 --port 3307 --user root --password "" --no-tls sql -q "..."
```

## Commits

- `572f4c2` — feat(loader): add Dolt SQL backend for live beads loading
- `9e0dcab` — feat(triage): add Dolt branch triage engine with TUI diff view
- MR `bd-0kd` — submitted to refinery merge queue
