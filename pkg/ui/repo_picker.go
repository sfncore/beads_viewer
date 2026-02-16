package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RepoPickerModel represents the repository filter picker overlay (workspace mode).
type RepoPickerModel struct {
	repos         []string
	selectedIndex int
	scrollOffset  int
	selected      map[string]bool // repo -> selected
	issueCounts   map[string]int  // repo -> issue count
	isDoltMode    bool            // controls "Rig" vs "Repo" label
	width         int
	height        int
	theme         Theme
}

// NewRepoPickerModel creates a new repo picker. By default, all repos are selected.
func NewRepoPickerModel(repos []string, theme Theme) RepoPickerModel {
	m := RepoPickerModel{
		repos:         append([]string(nil), repos...),
		selectedIndex: 0,
		selected:      make(map[string]bool, len(repos)),
		theme:         theme,
	}
	for _, r := range m.repos {
		m.selected[r] = true
	}
	return m
}

// SetSize updates the picker dimensions.
func (m *RepoPickerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// SetIssueCounts sets the issue count for each repo.
func (m *RepoPickerModel) SetIssueCounts(counts map[string]int) {
	m.issueCounts = counts
}

// SetDoltMode sets whether the picker shows "Rig Filter" (true) or "Repo Filter" (false).
func (m *RepoPickerModel) SetDoltMode(dolt bool) {
	m.isDoltMode = dolt
}

// SetActiveRepos initializes selection from the currently active repo filter (nil = all).
func (m *RepoPickerModel) SetActiveRepos(active map[string]bool) {
	if len(m.repos) == 0 {
		m.selected = map[string]bool{}
		return
	}

	m.selected = make(map[string]bool, len(m.repos))
	if active == nil {
		for _, r := range m.repos {
			m.selected[r] = true
		}
		return
	}

	for _, r := range m.repos {
		if active[r] {
			m.selected[r] = true
		}
	}
}

// visibleHeight returns the number of repo lines that fit in the box.
// Accounts for title (2 lines: title + blank), footer (2 lines: blank + footer),
// border padding (2 top + 2 bottom), and border itself (2).
func (m *RepoPickerModel) visibleHeight() int {
	// Box chrome: border (2) + padding (2 top + 2 bottom) = 6
	// Content chrome: title line + blank line + blank line + footer line = 4
	overhead := 10
	vh := m.height - overhead
	if vh < 3 {
		vh = 3
	}
	return vh
}

// MoveUp moves selection up.
func (m *RepoPickerModel) MoveUp() {
	if m.selectedIndex > 0 {
		m.selectedIndex--
	}
	if m.selectedIndex < m.scrollOffset {
		m.scrollOffset = m.selectedIndex
	}
}

// MoveDown moves selection down.
func (m *RepoPickerModel) MoveDown() {
	if m.selectedIndex < len(m.repos)-1 {
		m.selectedIndex++
	}
	vh := m.visibleHeight()
	if m.selectedIndex >= m.scrollOffset+vh {
		m.scrollOffset = m.selectedIndex - vh + 1
	}
}

// ToggleSelected toggles the selected state of the current repo.
func (m *RepoPickerModel) ToggleSelected() {
	if len(m.repos) == 0 || m.selectedIndex < 0 || m.selectedIndex >= len(m.repos) {
		return
	}
	r := m.repos[m.selectedIndex]
	m.selected[r] = !m.selected[r]
}

// SelectAll selects all repos.
func (m *RepoPickerModel) SelectAll() {
	for _, r := range m.repos {
		m.selected[r] = true
	}
}

// DeselectAll deselects all repos.
func (m *RepoPickerModel) DeselectAll() {
	for _, r := range m.repos {
		m.selected[r] = false
	}
}

// SelectedRepos returns the selected repos as a map (repo -> true).
func (m RepoPickerModel) SelectedRepos() map[string]bool {
	out := make(map[string]bool)
	for _, r := range m.repos {
		if m.selected[r] {
			out[r] = true
		}
	}
	return out
}

// View renders the repo picker overlay.
func (m *RepoPickerModel) View() string {
	if m.width == 0 {
		m.width = 60
	}
	if m.height == 0 {
		m.height = 20
	}

	t := m.theme

	// Calculate box dimensions
	boxWidth := 50
	if m.width < 60 {
		boxWidth = m.width - 10
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	// Content width inside box (minus border 2 + padding 4)
	innerWidth := boxWidth - 6
	if innerWidth < 20 {
		innerWidth = 20
	}

	var lines []string

	// Title: "Rig Filter" in dolt mode, "Repo Filter" otherwise
	titleStyle := t.Renderer.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		MarginBottom(1)
	title := "Repo Filter"
	if m.isDoltMode {
		title = "Rig Filter"
	}
	lines = append(lines, titleStyle.Render(title))
	lines = append(lines, "")

	if len(m.repos) == 0 {
		emptyStyle := t.Renderer.NewStyle().Foreground(t.Secondary).Italic(true)
		lines = append(lines, emptyStyle.Render("No repos available."))
	} else {
		vh := m.visibleHeight()
		total := len(m.repos)

		// Clamp scrollOffset
		if m.scrollOffset > total-vh {
			m.scrollOffset = total - vh
		}
		if m.scrollOffset < 0 {
			m.scrollOffset = 0
		}

		canScrollUp := m.scrollOffset > 0
		canScrollDown := m.scrollOffset+vh < total

		// Find max repo name length for alignment
		maxNameLen := 0
		for _, repo := range m.repos {
			if len(repo) > maxNameLen {
				maxNameLen = len(repo)
			}
		}

		// Scroll-up indicator
		if canScrollUp {
			arrowStyle := t.Renderer.NewStyle().Foreground(t.Secondary)
			lines = append(lines, arrowStyle.Render("  ↑"))
		}

		// Visible window
		end := m.scrollOffset + vh
		if end > total {
			end = total
		}
		for i := m.scrollOffset; i < end; i++ {
			repo := m.repos[i]
			isCursor := i == m.selectedIndex
			isSelected := m.selected[repo]

			nameStyle := t.Renderer.NewStyle().Foreground(t.Base.GetForeground())
			if isCursor {
				nameStyle = nameStyle.Foreground(t.Primary).Bold(true)
			}

			prefix := "  "
			if isCursor {
				prefix = "▸ "
			}
			check := "[ ]"
			if isSelected {
				check = "[x]"
			}

			// Build line with right-aligned count
			name := repo
			line := prefix + check + " " + name

			if m.issueCounts != nil {
				count := m.issueCounts[repo]
				countStr := fmt.Sprintf("%d", count)
				countStyle := t.Renderer.NewStyle().Foreground(t.Secondary)

				// Pad between name and count
				// prefix(2) + check(3) + space(1) + name + padding + count
				usedWidth := 2 + 3 + 1 + len(name)
				padding := innerWidth - usedWidth - len(countStr)
				if padding < 1 {
					padding = 1
				}

				line = prefix + check + " " + name + strings.Repeat(" ", padding) + countStyle.Render(countStr)
				// Re-render with nameStyle only on the name portion
				lines = append(lines, nameStyle.Render(prefix+check+" "+name)+strings.Repeat(" ", padding)+countStyle.Render(countStr))
				continue
			}

			lines = append(lines, nameStyle.Render(line))
		}

		// Scroll-down indicator
		if canScrollDown {
			arrowStyle := t.Renderer.NewStyle().Foreground(t.Secondary)
			lines = append(lines, arrowStyle.Render("  ↓"))
		}
	}

	lines = append(lines, "")

	// Footer
	footerStyle := t.Renderer.NewStyle().
		Foreground(t.Secondary).
		Italic(true)

	footerText := "j/k:move  space:toggle  a:all  A:none  enter:apply  esc:cancel"

	// Add position indicator when scrollable
	if len(m.repos) > m.visibleHeight() {
		footerText = fmt.Sprintf("%d/%d  %s", m.selectedIndex+1, len(m.repos), footerText)
	}

	lines = append(lines, footerStyle.Render(footerText))

	content := strings.Join(lines, "\n")

	boxStyle := t.Renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Primary).
		Padding(1, 2).
		Width(boxWidth)
	box := boxStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}
