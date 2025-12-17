package correlation

import (
	"testing"
	"time"
)

func TestBuildFileIndex(t *testing.T) {
	// Create a test history report
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-123": {
				BeadID: "bv-123",
				Title:  "Auth refactor",
				Status: "closed",
				Commits: []CorrelatedCommit{
					{
						SHA:       "abc123",
						ShortSHA:  "abc123",
						Timestamp: now.Add(-24 * time.Hour),
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 10, Deletions: 5},
							{Path: "pkg/auth/session.go", Insertions: 20, Deletions: 0},
						},
					},
					{
						SHA:       "def456",
						ShortSHA:  "def456",
						Timestamp: now.Add(-12 * time.Hour),
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 5, Deletions: 2},
						},
					},
				},
			},
			"bv-456": {
				BeadID: "bv-456",
				Title:  "API update",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "ghi789",
						ShortSHA:  "ghi789",
						Timestamp: now.Add(-6 * time.Hour),
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 3, Deletions: 1},
							{Path: "pkg/api/routes.go", Insertions: 50, Deletions: 10},
						},
					},
				},
			},
		},
		CommitIndex: CommitIndex{
			"abc123": {"bv-123"},
			"def456": {"bv-123"},
			"ghi789": {"bv-456"},
		},
	}

	index := BuildFileIndex(report)

	// Check stats
	if index.Stats.TotalFiles != 3 {
		t.Errorf("Expected 3 files, got %d", index.Stats.TotalFiles)
	}

	// token.go should have 2 beads, others should have 1
	tokenRefs := index.FileToBeads["pkg/auth/token.go"]
	if len(tokenRefs) != 2 {
		t.Errorf("Expected 2 beads for token.go, got %d", len(tokenRefs))
	}

	// Verify bead references for token.go
	var foundBv123, foundBv456 bool
	for _, ref := range tokenRefs {
		if ref.BeadID == "bv-123" {
			foundBv123 = true
			if len(ref.CommitSHAs) != 2 {
				t.Errorf("Expected 2 commits for bv-123, got %d", len(ref.CommitSHAs))
			}
			if ref.TotalChanges != 22 { // 10+5+5+2
				t.Errorf("Expected 22 total changes for bv-123, got %d", ref.TotalChanges)
			}
		}
		if ref.BeadID == "bv-456" {
			foundBv456 = true
		}
	}
	if !foundBv123 || !foundBv456 {
		t.Error("Expected both bv-123 and bv-456 in token.go references")
	}

	// Check files with multiple beads
	if index.Stats.FilesWithMultipleBeads != 1 {
		t.Errorf("Expected 1 file with multiple beads, got %d", index.Stats.FilesWithMultipleBeads)
	}
}

func TestBuildFileIndexEmpty(t *testing.T) {
	// Nil report
	index := BuildFileIndex(nil)
	if index == nil {
		t.Fatal("Expected non-nil index for nil report")
	}
	if len(index.FileToBeads) != 0 {
		t.Errorf("Expected empty FileToBeads, got %d entries", len(index.FileToBeads))
	}

	// Empty report
	emptyReport := &HistoryReport{
		Histories:   map[string]BeadHistory{},
		CommitIndex: CommitIndex{},
	}
	index = BuildFileIndex(emptyReport)
	if len(index.FileToBeads) != 0 {
		t.Errorf("Expected empty FileToBeads for empty report, got %d entries", len(index.FileToBeads))
	}
}

func TestNewFileLookupNil(t *testing.T) {
	// NewFileLookup should not panic with nil report
	lookup := NewFileLookup(nil)
	if lookup == nil {
		t.Fatal("Expected non-nil lookup for nil report")
	}

	// Should return empty results for any lookup
	result := lookup.LookupByFile("any/path.go")
	if result.TotalBeads != 0 {
		t.Errorf("Expected 0 beads for nil report lookup, got %d", result.TotalBeads)
	}

	// GetHotspots should return empty slice
	hotspots := lookup.GetHotspots(10)
	if len(hotspots) != 0 {
		t.Errorf("Expected 0 hotspots for nil report, got %d", len(hotspots))
	}

	// GetAllFiles should return empty slice
	files := lookup.GetAllFiles()
	if len(files) != 0 {
		t.Errorf("Expected 0 files for nil report, got %d", len(files))
	}

	// GetStats should return zero stats
	stats := lookup.GetStats()
	if stats.TotalFiles != 0 || stats.TotalBeadLinks != 0 {
		t.Errorf("Expected zero stats for nil report, got %+v", stats)
	}
}

func TestFileLookupByFile(t *testing.T) {
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-123": {
				BeadID: "bv-123",
				Title:  "Auth fix",
				Status: "closed",
				Commits: []CorrelatedCommit{
					{
						SHA:       "abc123",
						ShortSHA:  "abc123",
						Timestamp: now,
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 10, Deletions: 5},
						},
					},
				},
			},
			"bv-456": {
				BeadID: "bv-456",
				Title:  "Token refresh",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "def456",
						ShortSHA:  "def456",
						Timestamp: now,
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 5, Deletions: 2},
						},
					},
				},
			},
		},
		CommitIndex: CommitIndex{
			"abc123": {"bv-123"},
			"def456": {"bv-456"},
		},
	}

	lookup := NewFileLookup(report)

	// Exact file lookup
	result := lookup.LookupByFile("pkg/auth/token.go")
	if result.TotalBeads != 2 {
		t.Errorf("Expected 2 total beads, got %d", result.TotalBeads)
	}
	if len(result.OpenBeads) != 1 {
		t.Errorf("Expected 1 open bead, got %d", len(result.OpenBeads))
	}
	if len(result.ClosedBeads) != 1 {
		t.Errorf("Expected 1 closed bead, got %d", len(result.ClosedBeads))
	}

	// Non-existent file
	result = lookup.LookupByFile("nonexistent.go")
	if result.TotalBeads != 0 {
		t.Errorf("Expected 0 beads for nonexistent file, got %d", result.TotalBeads)
	}
}

func TestFileLookupByDirectory(t *testing.T) {
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-123": {
				BeadID: "bv-123",
				Title:  "Auth fix",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "abc123",
						ShortSHA:  "abc123",
						Timestamp: now,
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 10, Deletions: 5},
							{Path: "pkg/auth/session.go", Insertions: 5, Deletions: 2},
						},
					},
				},
			},
			"bv-456": {
				BeadID: "bv-456",
				Title:  "API update",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "def456",
						ShortSHA:  "def456",
						Timestamp: now,
						Files: []FileChange{
							{Path: "pkg/api/routes.go", Insertions: 20, Deletions: 10},
						},
					},
				},
			},
		},
		CommitIndex: CommitIndex{
			"abc123": {"bv-123"},
			"def456": {"bv-456"},
		},
	}

	lookup := NewFileLookup(report)

	// Directory lookup for pkg/auth
	result := lookup.LookupByFile("pkg/auth")
	if result.TotalBeads != 1 {
		t.Errorf("Expected 1 bead for pkg/auth directory, got %d", result.TotalBeads)
	}
	if len(result.OpenBeads) != 1 || result.OpenBeads[0].BeadID != "bv-123" {
		t.Error("Expected bv-123 in open beads for pkg/auth")
	}
}

func TestFileLookupByGlob(t *testing.T) {
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-123": {
				BeadID: "bv-123",
				Title:  "Go files fix",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "abc123",
						ShortSHA:  "abc123",
						Timestamp: now,
						Files: []FileChange{
							{Path: "pkg/auth/token.go", Insertions: 10, Deletions: 5},
							{Path: "pkg/api/routes.go", Insertions: 20, Deletions: 10},
						},
					},
				},
			},
		},
		CommitIndex: CommitIndex{
			"abc123": {"bv-123"},
		},
	}

	lookup := NewFileLookup(report)

	// Glob pattern for all .go files in pkg/auth
	result := lookup.LookupByFileGlob("pkg/auth/*.go")
	if result.TotalBeads != 1 {
		t.Errorf("Expected 1 bead for glob pattern, got %d", result.TotalBeads)
	}
}

func TestGetHotspots(t *testing.T) {
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-1": {
				BeadID: "bv-1",
				Title:  "Fix 1",
				Status: "closed",
				Commits: []CorrelatedCommit{
					{
						SHA:       "a",
						ShortSHA:  "a",
						Timestamp: now,
						Files:     []FileChange{{Path: "hot.go"}},
					},
				},
			},
			"bv-2": {
				BeadID: "bv-2",
				Title:  "Fix 2",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "b",
						ShortSHA:  "b",
						Timestamp: now,
						Files:     []FileChange{{Path: "hot.go"}},
					},
				},
			},
			"bv-3": {
				BeadID: "bv-3",
				Title:  "Fix 3",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "c",
						ShortSHA:  "c",
						Timestamp: now,
						Files:     []FileChange{{Path: "hot.go"}, {Path: "cold.go"}},
					},
				},
			},
		},
		CommitIndex: CommitIndex{
			"a": {"bv-1"},
			"b": {"bv-2"},
			"c": {"bv-3"},
		},
	}

	lookup := NewFileLookup(report)

	hotspots := lookup.GetHotspots(2)
	if len(hotspots) != 2 {
		t.Fatalf("Expected 2 hotspots, got %d", len(hotspots))
	}

	// hot.go should be first with 3 beads
	if hotspots[0].FilePath != "hot.go" {
		t.Errorf("Expected hot.go as top hotspot, got %s", hotspots[0].FilePath)
	}
	if hotspots[0].TotalBeads != 3 {
		t.Errorf("Expected 3 beads for hot.go, got %d", hotspots[0].TotalBeads)
	}
	if hotspots[0].OpenBeads != 2 {
		t.Errorf("Expected 2 open beads for hot.go, got %d", hotspots[0].OpenBeads)
	}
	if hotspots[0].ClosedBeads != 1 {
		t.Errorf("Expected 1 closed bead for hot.go, got %d", hotspots[0].ClosedBeads)
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"./pkg/auth/token.go", "pkg/auth/token.go"},
		{".\\pkg\\auth\\token.go", "pkg/auth/token.go"},
		{"pkg/auth/token.go", "pkg/auth/token.go"},
		{"pkg/auth/", "pkg/auth"},
		{"./", ""},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetAllFiles(t *testing.T) {
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-1": {
				BeadID: "bv-1",
				Title:  "Fix 1",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "a",
						ShortSHA:  "a",
						Timestamp: now,
						Files: []FileChange{
							{Path: "b.go"},
							{Path: "a.go"},
							{Path: "c.go"},
						},
					},
				},
			},
		},
		CommitIndex: CommitIndex{"a": {"bv-1"}},
	}

	lookup := NewFileLookup(report)
	files := lookup.GetAllFiles()

	if len(files) != 3 {
		t.Fatalf("Expected 3 files, got %d", len(files))
	}

	// Should be sorted
	expected := []string{"a.go", "b.go", "c.go"}
	for i, f := range files {
		if f != expected[i] {
			t.Errorf("File %d: expected %q, got %q", i, expected[i], f)
		}
	}
}

func TestImpactAnalysisEmpty(t *testing.T) {
	lookup := NewFileLookup(nil)
	result := lookup.ImpactAnalysis([]string{})
	if result.Summary != "No files to analyze" {
		t.Errorf("Expected 'No files to analyze', got %q", result.Summary)
	}
	if len(result.AffectedBeads) != 0 {
		t.Errorf("Expected 0 affected beads, got %d", len(result.AffectedBeads))
	}
}

func TestImpactAnalysisWithOpenBeads(t *testing.T) {
	now := time.Now()
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-1": {
				BeadID: "bv-1",
				Title:  "In progress work",
				Status: "in_progress",
				Commits: []CorrelatedCommit{
					{
						SHA:       "a",
						ShortSHA:  "a",
						Timestamp: now.Add(-1 * time.Hour),
						Files:     []FileChange{{Path: "auth/token.go", Insertions: 10}},
					},
				},
			},
			"bv-2": {
				BeadID: "bv-2",
				Title:  "Open task",
				Status: "open",
				Commits: []CorrelatedCommit{
					{
						SHA:       "b",
						ShortSHA:  "b",
						Timestamp: now.Add(-2 * time.Hour),
						Files:     []FileChange{{Path: "auth/token.go", Insertions: 5}},
					},
				},
			},
			"bv-3": {
				BeadID: "bv-3",
				Title:  "Closed task",
				Status: "closed",
				Commits: []CorrelatedCommit{
					{
						SHA:       "c",
						ShortSHA:  "c",
						Timestamp: now.Add(-24 * time.Hour),
						Files:     []FileChange{{Path: "auth/token.go", Insertions: 3}},
					},
				},
			},
		},
		CommitIndex: CommitIndex{
			"a": {"bv-1"},
			"b": {"bv-2"},
			"c": {"bv-3"},
		},
	}

	lookup := NewFileLookup(report)
	result := lookup.ImpactAnalysis([]string{"auth/token.go"})

	// Should have high risk due to in_progress bead
	if result.RiskLevel != "high" && result.RiskLevel != "critical" {
		t.Errorf("Expected high or critical risk, got %q", result.RiskLevel)
	}

	// Should have warnings
	if len(result.Warnings) == 0 {
		t.Error("Expected warnings for in_progress bead")
	}

	// in_progress should be first
	if len(result.AffectedBeads) < 1 || result.AffectedBeads[0].Status != "in_progress" {
		t.Error("Expected in_progress bead to be first")
	}
}

func TestImpactAnalysisRiskLevels(t *testing.T) {
	now := time.Now()

	// Test low risk (only old closed beads)
	report := &HistoryReport{
		Histories: map[string]BeadHistory{
			"bv-1": {
				BeadID: "bv-1",
				Title:  "Old closed",
				Status: "closed",
				Commits: []CorrelatedCommit{
					{
						SHA:       "a",
						ShortSHA:  "a",
						Timestamp: now.Add(-10 * 24 * time.Hour), // 10 days ago
						Files:     []FileChange{{Path: "old.go"}},
					},
				},
			},
		},
		CommitIndex: CommitIndex{"a": {"bv-1"}},
	}

	lookup := NewFileLookup(report)
	result := lookup.ImpactAnalysis([]string{"old.go"})

	// Should be low risk since the closed bead is > 7 days old
	if result.RiskLevel != "low" {
		t.Errorf("Expected low risk for old closed beads, got %q", result.RiskLevel)
	}
}
