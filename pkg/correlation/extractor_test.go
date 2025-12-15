package correlation

import (
	"testing"
	"time"
)

func TestSplitByCommits(t *testing.T) {
	// Mock git log output with two commits
	data := []byte(`abc123def456789012345678901234567890abcd|2025-01-15T10:00:00Z|Alice|alice@example.com|First commit
diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
--- a/.beads/beads.jsonl
+++ b/.beads/beads.jsonl
+{"id":"bv-001","title":"First bead","status":"open"}
def456789012345678901234567890abcdef1234|2025-01-16T11:00:00Z|Bob|bob@example.com|Second commit
diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
--- a/.beads/beads.jsonl
+++ b/.beads/beads.jsonl
-{"id":"bv-001","title":"First bead","status":"open"}
+{"id":"bv-001","title":"First bead","status":"in_progress"}
`)

	commits := splitByCommits(data)

	if len(commits) != 2 {
		t.Fatalf("Expected 2 commits, got %d", len(commits))
	}

	// Check first commit contains "First commit"
	if string(commits[0][:20]) != "abc123def45678901234" {
		t.Errorf("First commit should start with SHA")
	}

	// Check second commit starts with its SHA
	if len(commits[1]) < 40 || string(commits[1][:3]) != "def" {
		t.Errorf("Second commit should start with SHA 'def...', got: %s", string(commits[1][:20]))
	}
}

func TestParseCommitInfo(t *testing.T) {
	data := []byte(`abc123def456789012345678901234567890abcd|2025-01-15T10:30:00Z|Alice Smith|alice@example.com|feat: add login feature
diff --git ...
`)

	info, diffStart, err := parseCommitInfo(data)
	if err != nil {
		t.Fatalf("parseCommitInfo failed: %v", err)
	}

	if info.SHA != "abc123def456789012345678901234567890abcd" {
		t.Errorf("SHA mismatch: got %s", info.SHA)
	}
	if info.Author != "Alice Smith" {
		t.Errorf("Author mismatch: got %s", info.Author)
	}
	if info.AuthorEmail != "alice@example.com" {
		t.Errorf("AuthorEmail mismatch: got %s", info.AuthorEmail)
	}
	if info.Message != "feat: add login feature" {
		t.Errorf("Message mismatch: got %s", info.Message)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	if !info.Timestamp.Equal(expectedTime) {
		t.Errorf("Timestamp mismatch: got %v, want %v", info.Timestamp, expectedTime)
	}

	if diffStart <= 0 {
		t.Errorf("diffStart should be positive, got %d", diffStart)
	}
}

func TestParseCommitInfo_InvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"missing parts", []byte("abc123|2025-01-15\n")},
		{"no newline", []byte("abc123|2025-01-15|author|email|msg")},
		{"invalid timestamp", []byte("abc123|not-a-date|author|email|msg\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseCommitInfo(tt.data)
			if err == nil {
				t.Error("Expected error for invalid input")
			}
		})
	}
}

func TestParseBeadJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wantID   string
		wantOK   bool
	}{
		{
			name:   "valid bead",
			json:   `{"id":"bv-123","title":"Test","status":"open"}`,
			wantID: "bv-123",
			wantOK: true,
		},
		{
			name:   "valid bead with extra fields",
			json:   `{"id":"bv-456","title":"Feature","status":"closed","priority":1,"labels":["urgent"]}`,
			wantID: "bv-456",
			wantOK: true,
		},
		{
			name:   "missing id",
			json:   `{"title":"No ID","status":"open"}`,
			wantID: "",
			wantOK: false,
		},
		{
			name:   "invalid json",
			json:   `{not valid json}`,
			wantID: "",
			wantOK: false,
		},
		{
			name:   "empty object",
			json:   `{}`,
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap, ok := parseBeadJSON(tt.json)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && snap.ID != tt.wantID {
				t.Errorf("ID = %s, want %s", snap.ID, tt.wantID)
			}
		})
	}
}

func TestDetermineStatusEvent(t *testing.T) {
	tests := []struct {
		oldStatus string
		newStatus string
		want      EventType
	}{
		{"open", "in_progress", EventClaimed},
		{"in_progress", "closed", EventClosed},
		{"open", "closed", EventClosed},
		{"closed", "open", EventReopened},
		{"closed", "in_progress", EventClaimed},
		{"open", "blocked", EventModified},
		{"in_progress", "open", EventModified},
	}

	for _, tt := range tests {
		t.Run(tt.oldStatus+"->"+tt.newStatus, func(t *testing.T) {
			got := determineStatusEvent(tt.oldStatus, tt.newStatus)
			if got != tt.want {
				t.Errorf("determineStatusEvent(%s, %s) = %v, want %v", tt.oldStatus, tt.newStatus, got, tt.want)
			}
		})
	}
}

func TestReverseEvents(t *testing.T) {
	events := []BeadEvent{
		{BeadID: "a", EventType: EventCreated},
		{BeadID: "b", EventType: EventClaimed},
		{BeadID: "c", EventType: EventClosed},
	}

	reverseEvents(events)

	if events[0].BeadID != "c" || events[1].BeadID != "b" || events[2].BeadID != "a" {
		t.Errorf("reverseEvents failed: got %v, %v, %v", events[0].BeadID, events[1].BeadID, events[2].BeadID)
	}
}

func TestGetBeadMilestones(t *testing.T) {
	now := time.Now()
	events := []BeadEvent{
		{BeadID: "bv-1", EventType: EventCreated, Timestamp: now},
		{BeadID: "bv-1", EventType: EventClaimed, Timestamp: now.Add(time.Hour)},
		{BeadID: "bv-1", EventType: EventClosed, Timestamp: now.Add(2 * time.Hour)},
		{BeadID: "bv-1", EventType: EventReopened, Timestamp: now.Add(3 * time.Hour)},
		{BeadID: "bv-1", EventType: EventClosed, Timestamp: now.Add(4 * time.Hour)},
	}

	milestones := GetBeadMilestones(events)

	if milestones.Created == nil {
		t.Error("Created should not be nil")
	}
	if milestones.Claimed == nil {
		t.Error("Claimed should not be nil")
	}
	if milestones.Closed == nil {
		t.Error("Closed should not be nil")
	}
	if milestones.Reopened == nil {
		t.Error("Reopened should not be nil")
	}

	// Check that Closed is the latest close event
	if !milestones.Closed.Timestamp.Equal(now.Add(4 * time.Hour)) {
		t.Error("Closed should be the latest close event")
	}
}

func TestCalculateCycleTime(t *testing.T) {
	now := time.Now()
	created := BeadEvent{EventType: EventCreated, Timestamp: now}
	claimed := BeadEvent{EventType: EventClaimed, Timestamp: now.Add(24 * time.Hour)}
	closed := BeadEvent{EventType: EventClosed, Timestamp: now.Add(48 * time.Hour)}

	t.Run("with all milestones", func(t *testing.T) {
		milestones := BeadMilestones{
			Created: &created,
			Claimed: &claimed,
			Closed:  &closed,
		}

		ct := CalculateCycleTime(milestones)

		if ct == nil {
			t.Fatal("CycleTime should not be nil")
		}
		if ct.ClaimToClose == nil {
			t.Error("ClaimToClose should not be nil")
		}
		if ct.CreateToClose == nil {
			t.Error("CreateToClose should not be nil")
		}
		if ct.CreateToClaim == nil {
			t.Error("CreateToClaim should not be nil")
		}

		expectedClaimToClose := 24 * time.Hour
		if *ct.ClaimToClose != expectedClaimToClose {
			t.Errorf("ClaimToClose = %v, want %v", *ct.ClaimToClose, expectedClaimToClose)
		}
	})

	t.Run("without closed milestone", func(t *testing.T) {
		milestones := BeadMilestones{
			Created: &created,
			Claimed: &claimed,
		}

		ct := CalculateCycleTime(milestones)

		if ct != nil {
			t.Error("CycleTime should be nil for unclosed beads")
		}
	})
}

func TestInsertBefore(t *testing.T) {
	slice := []string{"a", "b", "--", "c", "d"}

	result := insertBefore(slice, "--", "x")

	expected := []string{"a", "b", "x", "--", "c", "d"}
	if len(result) != len(expected) {
		t.Fatalf("length mismatch: got %d, want %d", len(result), len(expected))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("result[%d] = %s, want %s", i, result[i], v)
		}
	}
}

func TestInsertBefore_NoMarker(t *testing.T) {
	slice := []string{"a", "b", "c"}

	result := insertBefore(slice, "--", "x")

	// Should return original slice unchanged
	if len(result) != len(slice) {
		t.Errorf("length changed when marker not found: got %d, want %d", len(result), len(slice))
	}
}

func TestBuildGitLogArgs(t *testing.T) {
	e := NewExtractor("/test/repo")

	t.Run("basic args", func(t *testing.T) {
		args := e.buildGitLogArgs(ExtractOptions{})

		// Should contain -p, --follow, --format
		found := map[string]bool{}
		for _, arg := range args {
			if arg == "-p" {
				found["-p"] = true
			}
			if arg == "--follow" {
				found["--follow"] = true
			}
		}
		if !found["-p"] {
			t.Error("missing -p flag")
		}
		if !found["--follow"] {
			t.Error("missing --follow flag")
		}
	})

	t.Run("with limit", func(t *testing.T) {
		args := e.buildGitLogArgs(ExtractOptions{Limit: 10})

		found := false
		for _, arg := range args {
			if arg == "-n10" {
				found = true
				break
			}
		}
		if !found {
			t.Error("missing -n10 flag")
		}
	})

	t.Run("with time filters", func(t *testing.T) {
		since := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		until := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

		args := e.buildGitLogArgs(ExtractOptions{
			Since: &since,
			Until: &until,
		})

		foundSince := false
		foundUntil := false
		for _, arg := range args {
			if len(arg) > 8 && arg[:8] == "--since=" {
				foundSince = true
			}
			if len(arg) > 8 && arg[:8] == "--until=" {
				foundUntil = true
			}
		}
		if !foundSince {
			t.Error("missing --since flag")
		}
		if !foundUntil {
			t.Error("missing --until flag")
		}
	})
}

// TestParseDiff tests the diff parsing logic with mock data
func TestParseDiff(t *testing.T) {
	e := NewExtractor("/test/repo")

	info := commitInfo{
		SHA:         "abc123",
		Timestamp:   time.Now(),
		Author:      "Test",
		AuthorEmail: "test@test.com",
		Message:     "Test commit",
	}

	t.Run("new bead creation", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
--- a/.beads/beads.jsonl
+++ b/.beads/beads.jsonl
+{"id":"bv-new","title":"New bead","status":"open"}
`)

		events := e.parseDiff(diffData, info, "")

		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}
		if events[0].EventType != EventCreated {
			t.Errorf("Expected EventCreated, got %v", events[0].EventType)
		}
		if events[0].BeadID != "bv-new" {
			t.Errorf("Expected bv-new, got %s", events[0].BeadID)
		}
	})

	t.Run("status change to in_progress", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
--- a/.beads/beads.jsonl
+++ b/.beads/beads.jsonl
-{"id":"bv-123","title":"Test","status":"open"}
+{"id":"bv-123","title":"Test","status":"in_progress"}
`)

		events := e.parseDiff(diffData, info, "")

		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}
		if events[0].EventType != EventClaimed {
			t.Errorf("Expected EventClaimed, got %v", events[0].EventType)
		}
	})

	t.Run("status change to closed", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
--- a/.beads/beads.jsonl
+++ b/.beads/beads.jsonl
-{"id":"bv-123","title":"Test","status":"in_progress"}
+{"id":"bv-123","title":"Test","status":"closed"}
`)

		events := e.parseDiff(diffData, info, "")

		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}
		if events[0].EventType != EventClosed {
			t.Errorf("Expected EventClosed, got %v", events[0].EventType)
		}
	})

	t.Run("reopen closed bead", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
--- a/.beads/beads.jsonl
+++ b/.beads/beads.jsonl
-{"id":"bv-123","title":"Test","status":"closed"}
+{"id":"bv-123","title":"Test","status":"open"}
`)

		events := e.parseDiff(diffData, info, "")

		if len(events) != 1 {
			t.Fatalf("Expected 1 event, got %d", len(events))
		}
		if events[0].EventType != EventReopened {
			t.Errorf("Expected EventReopened, got %v", events[0].EventType)
		}
	})

	t.Run("filter by bead ID", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
+{"id":"bv-001","title":"First","status":"open"}
+{"id":"bv-002","title":"Second","status":"open"}
`)

		events := e.parseDiff(diffData, info, "bv-001")

		if len(events) != 1 {
			t.Fatalf("Expected 1 event (filtered), got %d", len(events))
		}
		if events[0].BeadID != "bv-001" {
			t.Errorf("Expected bv-001, got %s", events[0].BeadID)
		}
	})

	t.Run("multiple beads in one commit", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
+{"id":"bv-001","title":"First","status":"open"}
+{"id":"bv-002","title":"Second","status":"open"}
+{"id":"bv-003","title":"Third","status":"open"}
`)

		events := e.parseDiff(diffData, info, "")

		if len(events) != 3 {
			t.Fatalf("Expected 3 events, got %d", len(events))
		}
	})

	t.Run("malformed JSON skipped", func(t *testing.T) {
		diffData := []byte(`diff --git a/.beads/beads.jsonl b/.beads/beads.jsonl
+{"id":"bv-good","title":"Good","status":"open"}
+{malformed json here}
+{"id":"bv-also-good","title":"Also Good","status":"open"}
`)

		events := e.parseDiff(diffData, info, "")

		if len(events) != 2 {
			t.Fatalf("Expected 2 events (skipping malformed), got %d", len(events))
		}
	})
}
