package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultTheme(t *testing.T) {
	renderer := lipgloss.NewRenderer(nil)
	theme := DefaultTheme(renderer)

	if theme.Renderer != renderer {
		t.Error("DefaultTheme renderer mismatch")
	}
	// Check a few known colors are set (not zero value)
	if isColorEmpty(theme.Primary) {
		t.Error("DefaultTheme Primary color is empty")
	}
	if isColorEmpty(theme.Open) {
		t.Error("DefaultTheme Open color is empty")
	}
}

func isColorEmpty(c lipgloss.AdaptiveColor) bool {
	return c.Light == "" && c.Dark == ""
}

func TestGetStatusColor(t *testing.T) {
	renderer := lipgloss.NewRenderer(nil)
	theme := DefaultTheme(renderer)

	tests := []struct {
		status string
		want   lipgloss.AdaptiveColor
	}{
		{"open", theme.Open},
		{"in_progress", theme.InProgress},
		{"blocked", theme.Blocked},
		{"closed", theme.Closed},
		{"unknown", theme.Subtext},
		{"", theme.Subtext},
	}

	for _, tt := range tests {
		got := theme.GetStatusColor(tt.status)
		if got != tt.want {
			t.Errorf("GetStatusColor(%q) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestGetTypeIcon(t *testing.T) {
	renderer := lipgloss.NewRenderer(nil)
	theme := DefaultTheme(renderer)

	tests := []struct {
		typ      string
		wantIcon string
		wantCol  lipgloss.AdaptiveColor
	}{
		{"bug", "🐛", theme.Bug},
		{"feature", "✨", theme.Feature},
		{"task", "📋", theme.Task},
		{"epic", "🏔️", theme.Epic},
		{"chore", "🧹", theme.Chore},
		{"unknown", "•", theme.Subtext},
	}

	for _, tt := range tests {
		icon, col := theme.GetTypeIcon(tt.typ)
		if icon != tt.wantIcon {
			t.Errorf("GetTypeIcon(%q) icon = %q, want %q", tt.typ, icon, tt.wantIcon)
		}
		if col != tt.wantCol {
			t.Errorf("GetTypeIcon(%q) color = %v, want %v", tt.typ, col, tt.wantCol)
		}
	}
}
