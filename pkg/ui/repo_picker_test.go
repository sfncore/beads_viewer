package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRepoPickerSelectionAndToggle(t *testing.T) {
	repos := []string{"api", "web", "lib"}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(80, 24)

	// Default is all selected
	if got := len(m.SelectedRepos()); got != 3 {
		t.Fatalf("expected 3 selected repos by default, got %d", got)
	}

	// Toggle first repo off
	m.ToggleSelected()
	if got := len(m.SelectedRepos()); got != 2 {
		t.Fatalf("expected 2 selected after toggle, got %d", got)
	}

	// Select all
	m.SelectAll()
	if got := len(m.SelectedRepos()); got != 3 {
		t.Fatalf("expected 3 selected after SelectAll, got %d", got)
	}
}

func TestRepoPickerDeselectAll(t *testing.T) {
	repos := []string{"api", "web", "lib"}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(80, 24)

	// All selected by default
	if got := len(m.SelectedRepos()); got != 3 {
		t.Fatalf("expected 3 selected by default, got %d", got)
	}

	// Deselect all
	m.DeselectAll()
	if got := len(m.SelectedRepos()); got != 0 {
		t.Fatalf("expected 0 selected after DeselectAll, got %d", got)
	}

	// Re-select all
	m.SelectAll()
	if got := len(m.SelectedRepos()); got != 3 {
		t.Fatalf("expected 3 selected after SelectAll, got %d", got)
	}
}

func TestRepoPickerViewContainsRepos(t *testing.T) {
	repos := []string{"api"}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(60, 20)

	// Default mode: "Repo Filter"
	out := m.View()
	if !strings.Contains(out, "Repo Filter") {
		t.Fatalf("expected 'Repo Filter' title in view, got:\n%s", out)
	}
	if !strings.Contains(out, "api") {
		t.Fatalf("expected repo name in view, got:\n%s", out)
	}
}

func TestRepoPickerDoltModeTitle(t *testing.T) {
	repos := []string{"beads", "gastown"}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(60, 20)
	m.SetDoltMode(true)

	out := m.View()
	if !strings.Contains(out, "Rig Filter") {
		t.Fatalf("expected 'Rig Filter' title in dolt mode, got:\n%s", out)
	}
	if strings.Contains(out, "Repo Filter") {
		t.Fatalf("should not contain 'Repo Filter' in dolt mode, got:\n%s", out)
	}
}

func TestRepoPickerScrolling(t *testing.T) {
	// Create many repos in a small window to force scrolling
	repos := make([]string, 20)
	for i := range repos {
		repos[i] = strings.Repeat("r", 1) + string(rune('a'+i%26))
	}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(60, 16) // Small height forces scrolling

	vh := m.visibleHeight()
	if vh >= 20 {
		t.Fatalf("visibleHeight should be less than 20 repos for height 16, got %d", vh)
	}

	// Initially at top, scroll offset 0
	if m.scrollOffset != 0 {
		t.Fatalf("expected scrollOffset 0, got %d", m.scrollOffset)
	}

	// Move down past visible height
	for i := 0; i < vh+2; i++ {
		m.MoveDown()
	}

	if m.scrollOffset == 0 {
		t.Fatalf("expected scrollOffset > 0 after scrolling down, got 0")
	}
	if m.selectedIndex != vh+2 {
		t.Fatalf("expected selectedIndex %d, got %d", vh+2, m.selectedIndex)
	}

	// View should contain the down arrow indicator
	out := m.View()
	// The scroll-down indicator should be visible if there are items below
	if m.scrollOffset+vh < len(repos) && !strings.Contains(out, "↓") {
		t.Fatalf("expected scroll-down indicator in view")
	}

	// Move back up to top
	for i := 0; i < vh+5; i++ {
		m.MoveUp()
	}

	if m.selectedIndex != 0 {
		t.Fatalf("expected selectedIndex 0 at top, got %d", m.selectedIndex)
	}
	if m.scrollOffset != 0 {
		t.Fatalf("expected scrollOffset 0 at top, got %d", m.scrollOffset)
	}
}

func TestRepoPickerIssueCounts(t *testing.T) {
	repos := []string{"beads", "gastown", "bv"}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(60, 20)
	m.SetIssueCounts(map[string]int{
		"beads":   42,
		"gastown": 18,
		"bv":      127,
	})

	out := m.View()
	if !strings.Contains(out, "42") {
		t.Fatalf("expected issue count 42 in view, got:\n%s", out)
	}
	if !strings.Contains(out, "18") {
		t.Fatalf("expected issue count 18 in view, got:\n%s", out)
	}
	if !strings.Contains(out, "127") {
		t.Fatalf("expected issue count 127 in view, got:\n%s", out)
	}
}

func TestRepoPickerFooterShowsDeselectHint(t *testing.T) {
	repos := []string{"api"}
	m := NewRepoPickerModel(repos, DefaultTheme(lipgloss.NewRenderer(nil)))
	m.SetSize(120, 20) // Wide enough to avoid word wrapping

	out := m.View()
	if !strings.Contains(out, "A:none") {
		t.Fatalf("expected 'A:none' hint in footer, got:\n%s", out)
	}
}
