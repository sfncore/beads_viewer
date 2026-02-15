package ui

import (
	"fmt"
	"strings"

	"github.com/Dicklesworthstone/beads_viewer/pkg/triage"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TriageDiffModal displays triage proposals and diffs for review.
type TriageDiffModal struct {
	branch    *triage.TriageBranch
	diffs     []triage.DiffEntry
	cursor    int  // Currently highlighted proposal
	scrollOff int  // Scroll offset for long lists
	accepted  bool // User accepted (merge)
	rejected  bool // User rejected (discard)
	theme     Theme
	width     int
	height    int
}

// TriageDiffLoadedMsg is sent when triage data finishes loading.
type TriageDiffLoadedMsg struct {
	Branch *triage.TriageBranch
	Diffs  []triage.DiffEntry
	Err    error
}

// NewTriageDiffModal creates a modal from triage results.
func NewTriageDiffModal(branch *triage.TriageBranch, diffs []triage.DiffEntry, theme Theme) TriageDiffModal {
	return TriageDiffModal{
		branch: branch,
		diffs:  diffs,
		theme:  theme,
		width:  80,
		height: 30,
	}
}

// Update handles keyboard input for the modal.
func (m TriageDiffModal) Update(msg tea.Msg) (TriageDiffModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.scrollOff {
					m.scrollOff = m.cursor
				}
			}
		case "down", "j":
			max := len(m.branch.Proposals) - 1
			if m.cursor < max {
				m.cursor++
				visible := m.visibleRows()
				if m.cursor >= m.scrollOff+visible {
					m.scrollOff = m.cursor - visible + 1
				}
			}
		case "y", "Y":
			m.accepted = true
		case "n", "N":
			m.rejected = true
		}
	}
	return m, nil
}

// IsAccepted returns true if the user chose to merge.
func (m TriageDiffModal) IsAccepted() bool { return m.accepted }

// IsRejected returns true if the user chose to discard.
func (m TriageDiffModal) IsRejected() bool { return m.rejected }

// View renders the modal content.
func (m TriageDiffModal) View() string {
	r := m.theme.Renderer

	modalStyle := r.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary).
		Padding(1, 2).
		Width(m.width)

	headerStyle := r.NewStyle().Bold(true).Foreground(m.theme.Primary)
	subtextStyle := r.NewStyle().Foreground(m.theme.Subtext).Italic(true)
	dimStyle := r.NewStyle().Foreground(m.theme.Muted)
	highlightStyle := r.NewStyle().Bold(true).Foreground(m.theme.Highlight)

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render(fmt.Sprintf("Triage Review: %s", m.branch.Database)))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Branch: %s", m.branch.BranchName)))
	b.WriteString("\n\n")

	// Summary stats
	report := m.branch.Report
	b.WriteString(fmt.Sprintf("Issues: %d total, %d open   ", report.IssueCount, report.OpenCount))
	b.WriteString(highlightStyle.Render(fmt.Sprintf("Proposals: %d", report.ProposalCount)))
	b.WriteString("\n")

	// Count by type
	counts := make(map[triage.ChangeType]int)
	for _, p := range m.branch.Proposals {
		counts[p.ChangeType]++
	}
	var typeParts []string
	if n := counts[triage.ChangePriority]; n > 0 {
		typeParts = append(typeParts, fmt.Sprintf("%d priority", n))
	}
	if n := counts[triage.ChangeStatus]; n > 0 {
		typeParts = append(typeParts, fmt.Sprintf("%d status", n))
	}
	if n := counts[triage.ChangeLabel]; n > 0 {
		typeParts = append(typeParts, fmt.Sprintf("%d labels", n))
	}
	if len(typeParts) > 0 {
		b.WriteString(dimStyle.Render(strings.Join(typeParts, ", ")))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Proposal list
	if len(m.branch.Proposals) == 0 {
		b.WriteString(dimStyle.Render("No proposals - all issues look good."))
		b.WriteString("\n")
	} else {
		visible := m.visibleRows()
		end := m.scrollOff + visible
		if end > len(m.branch.Proposals) {
			end = len(m.branch.Proposals)
		}

		for i := m.scrollOff; i < end; i++ {
			p := m.branch.Proposals[i]
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}

			// Change type badge
			badge := m.changeTypeBadge(r, p.ChangeType)

			// Value change
			var change string
			if p.OldValue != "" && p.NewValue != "" {
				change = fmt.Sprintf("%s -> %s", p.OldValue, p.NewValue)
			} else if p.NewValue != "" {
				change = fmt.Sprintf("+%s", p.NewValue)
			} else {
				change = fmt.Sprintf("-%s", p.OldValue)
			}

			line := fmt.Sprintf("%s%s %-12s %s", prefix, badge, p.IssueID, change)
			if i == m.cursor {
				b.WriteString(highlightStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")

			// Show reason for selected item
			if i == m.cursor {
				b.WriteString(subtextStyle.Render(fmt.Sprintf("    %.3f  %s", p.Score, p.Reason)))
				b.WriteString("\n")
			}
		}

		if end < len(m.branch.Proposals) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  ... %d more (scroll down)", len(m.branch.Proposals)-end)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(subtextStyle.Render("[j/k] Navigate  [Y] Merge  [N] Discard  [Esc] Close"))

	return modalStyle.Render(b.String())
}

// changeTypeBadge returns a styled short badge for change type.
func (m TriageDiffModal) changeTypeBadge(r *lipgloss.Renderer, ct triage.ChangeType) string {
	var label string
	var fg lipgloss.AdaptiveColor
	switch ct {
	case triage.ChangePriority:
		label = "PRI"
		fg = m.theme.Open
	case triage.ChangeStatus:
		label = "STS"
		fg = m.theme.InProgress
	case triage.ChangeLabel:
		label = "LBL"
		fg = m.theme.Feature
	case triage.ChangeLabelDel:
		label = "LBL"
		fg = m.theme.Blocked
	case triage.ChangeDependency:
		label = "DEP"
		fg = m.theme.Deferred
	default:
		label = "???"
		fg = m.theme.Muted
	}
	return r.NewStyle().Foreground(fg).Bold(true).Render(fmt.Sprintf("[%s] ", label))
}

// visibleRows returns how many proposal rows fit in the modal.
func (m TriageDiffModal) visibleRows() int {
	// Account for header (6 lines), footer (2 lines), padding, border
	avail := m.height - 14
	if avail < 5 {
		avail = 5
	}
	return avail
}

// SetSize updates the modal dimensions.
func (m *TriageDiffModal) SetSize(width, height int) {
	maxWidth := width - 10
	if maxWidth < 60 {
		maxWidth = 60
	}
	if maxWidth > 90 {
		maxWidth = 90
	}
	m.width = maxWidth
	m.height = height
}

// CenterModal returns the modal centered in the terminal.
func (m TriageDiffModal) CenterModal(termWidth, termHeight int) string {
	modal := m.View()

	modalWidth := lipgloss.Width(modal)
	modalHeight := lipgloss.Height(modal)

	padTop := (termHeight - modalHeight) / 2
	padLeft := (termWidth - modalWidth) / 2

	if padTop < 0 {
		padTop = 0
	}
	if padLeft < 0 {
		padLeft = 0
	}

	r := m.theme.Renderer
	return r.NewStyle().
		MarginTop(padTop).
		MarginLeft(padLeft).
		Render(modal)
}
