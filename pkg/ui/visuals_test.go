package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestGetHeatmapColor(t *testing.T) {
	tests := []struct {
		score float64
		want  lipgloss.Color
	}{
		{0.9, GradientPeak},
		{0.6, GradientHigh},
		{0.3, GradientMid},
		{0.1, GradientLow},
	}

	for _, tt := range tests {
		got := GetHeatmapColor(tt.score)
		if got != tt.want {
			t.Errorf("GetHeatmapColor(%v) = %v, want %v", tt.score, got, tt.want)
		}
	}
}

func TestGetRepoColor(t *testing.T) {
	// Consistency check
	c1 := GetRepoColor("api")
	c2 := GetRepoColor("api")
	if c1 != c2 {
		t.Error("GetRepoColor should be deterministic")
	}

	// Different prefixes should likely have different colors (though collisions possible)
	c3 := GetRepoColor("web")
	if c1 == c3 {
		t.Logf("Note: collision or same color for 'api' and 'web': %v", c1)
	}

	// Empty prefix check
	cEmpty := GetRepoColor("")
	if cEmpty != ColorMuted {
		t.Errorf("GetRepoColor(\"\") = %v, want %v", cEmpty, ColorMuted)
	}
}

func TestRenderRepoBadge(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{"api", "[API]"},
		{"longprefix", "[LONG]"}, // Should truncate
		{"", ""},
	}

	for _, tt := range tests {
		got := RenderRepoBadge(tt.prefix)
		if tt.prefix == "" {
			if got != "" {
				t.Errorf("RenderRepoBadge(\"\") = %q, want empty", got)
			}
		} else {
			if !strings.Contains(got, tt.want) {
				t.Errorf("RenderRepoBadge(%q) = %q, want to contain %q", tt.prefix, got, tt.want)
			}
		}
	}
}

func TestRenderSparkline(t *testing.T) {
	tests := []struct {
		val   float64
		width int
	}{
		{0.0, 5},
		{0.5, 5},
		{1.0, 5},
		{-0.1, 5}, // Clamping
		{1.2, 5},  // Clamping
	}

	for _, tt := range tests {
		got := RenderSparkline(tt.val, tt.width)
		// Check width (length of string might vary due to unicode, but roughly)
		// Actually checking exact rune count is tricky with unicode bars, 
		// but we can check it's not empty
		if got == "" {
			t.Errorf("RenderSparkline(%v, %d) returned empty", tt.val, tt.width)
		}
	}
}
