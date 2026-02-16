# bv Keymaps

> Complete keyboard reference for the bv TUI.
>
> Source of truth: [`pkg/ui/model.go`](../pkg/ui/model.go) (key dispatch),
> [`pkg/ui/shortcuts_sidebar.go`](../pkg/ui/shortcuts_sidebar.go) (sidebar),
> [`pkg/ui/context_help.go`](../pkg/ui/context_help.go) (in-app help).

---

## Global (all views)

| Key | Action |
|-----|--------|
| `?` / `F1` | Toggle help overlay |
| `` ` `` | Toggle interactive tutorial |
| `;` / `F2` | Toggle shortcuts sidebar |
| `Ctrl+R` / `F5` | Force refresh data |
| `Ctrl+C` | Quit immediately |
| `q` | Close current view / quit |
| `Esc` | Close modal / back / clear filters / quit confirm |
| `Ctrl+J` / `Ctrl+K` | Scroll shortcuts sidebar (when visible) |

## View Switching (from list / non-filtering state)

| Key | Target |
|-----|--------|
| `b` | Board view |
| `g` | Graph view |
| `a` | Actionable view |
| `h` | History view |
| `i` | Insights panel |
| `E` | Tree view (hierarchical) |
| `f` | Flow matrix (cross-label deps) |
| `[` / `F3` | Label dashboard |
| `]` / `F4` | Attention view |
| `p` | Toggle priority hints |
| `!` | Toggle alerts panel |
| `'` | Recipe picker |
| `w` | Repo picker (workspace mode) |
| `l` | Label picker |

## List View

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move up / down (bubbles/list) |
| `Enter` | Open issue details |
| `home` | Jump to first issue |
| `G` / `End` | Jump to last issue |
| `Ctrl+D` | Page down |
| `Ctrl+U` | Page up |

### Filtering

| Key | Action |
|-----|--------|
| `o` | Open issues only |
| `c` | Closed issues only |
| `r` | Ready (no blockers) |
| `a` | All (clear filter) |
| `/` | Fuzzy search (bubbles/list built-in) |
| `Ctrl+S` | Toggle semantic (AI) search |
| `H` | Toggle hybrid search mode |
| `Alt+H` | Cycle hybrid search presets |

### Sorting

| Key | Action |
|-----|--------|
| `s` | Cycle sort mode (Default → Created ↑ → Created ↓ → Priority → Updated) |
| `S` | Apply triage recipe (sort by triage score) |

### Actions

| Key | Action |
|-----|--------|
| `t` | Time-travel: enter revision prompt |
| `T` | Quick time-travel (HEAD~5) |
| `y` | Copy issue ID to clipboard |
| `C` | Copy full issue to clipboard |
| `O` | Open beads.jsonl in `$EDITOR` |
| `x` | Export to Markdown file |
| `V` | Cass session preview modal |
| `U` | Self-update modal |

## Split View

| Key | Action |
|-----|--------|
| `Tab` | Switch focus between list and detail panes |
| `<` | Shrink list pane |
| `>` | Expand list pane |

## Detail View

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll content (viewport) |
| `Esc` / `q` | Return to list |

## Board View

### Navigation

| Key | Action |
|-----|--------|
| `h` / `l` / `←` / `→` | Move between columns |
| `j` / `k` / `↑` / `↓` | Move within column |
| `1` - `4` | Jump to column (Open / In Progress / Blocked / Closed) |
| `H` / `L` | Jump to first / last column |
| `gg` | Jump to top of column |
| `G` / `End` | Jump to bottom of column |
| `0` / `$` | First / last item in column |
| `home` | Top of column |
| `Ctrl+D` / `Ctrl+U` | Page down / up |

### Search

| Key | Action |
|-----|--------|
| `/` | Start search |
| `n` / `N` | Next / previous match |
| `Enter` | Finish search (keep results) |
| `Esc` | Cancel search |
| `Backspace` | Delete character |

### Filtering & Display

| Key | Action |
|-----|--------|
| `o` / `c` / `r` | Filter: open / closed / ready |
| `s` | Cycle swimlane mode (Status / Priority / Type) |
| `e` | Toggle empty column visibility |
| `d` | Toggle inline card expansion |

### Actions

| Key | Action |
|-----|--------|
| `Tab` | Toggle detail panel |
| `Ctrl+J` / `Ctrl+K` | Scroll detail panel |
| `y` | Copy issue ID to clipboard |
| `Enter` | Open issue in detail view |

## Graph View

| Key | Action |
|-----|--------|
| `h` / `j` / `k` / `l` | Navigate (vim directions) |
| `H` / `L` | Scroll left / right |
| `Ctrl+D` / `PgDn` | Page down |
| `Ctrl+U` / `PgUp` | Page up |
| `Enter` | Jump to selected issue |

## Tree View

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move up / down |
| `Enter` / `Space` | Toggle expand node |
| `h` / `←` | Collapse or jump to parent |
| `l` / `→` | Expand or move to child |
| `g` / `G` | Jump to top / bottom |
| `o` / `O` | Expand all / collapse all |
| `Ctrl+D` / `PgDn` | Page down |
| `Ctrl+U` / `PgUp` | Page up |
| `Tab` | Toggle detail panel (syncs selection) |
| `E` / `Esc` | Return to list view |

## Actionable View

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move up / down |
| `Enter` | Jump to selected issue in list |

## History View

### Navigation

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate primary pane |
| `J` / `K` | Navigate secondary pane |
| `Tab` | Cycle focus (list → detail → file tree) |
| `Enter` | Jump to selected bead in list |

### Modes

| Key | Action |
|-----|--------|
| `v` | Toggle Bead / Git mode |
| `/` | Start search |
| `c` | Cycle confidence filter (bead mode) |
| `f` / `F` | Toggle file tree panel |

### File Tree (when focused)

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate files |
| `Enter` / `l` | Expand dir / select file |
| `h` | Collapse directory |
| `Esc` | Clear file filter / unfocus tree |
| `Tab` | Switch focus away |

### Actions

| Key | Action |
|-----|--------|
| `y` | Copy commit SHA |
| `o` | Open commit in browser |
| `g` | Graph view for selected bead |
| `h` / `Esc` | Exit history view |

## Insights Panel

| Key | Action |
|-----|--------|
| `h` / `l` / `←` / `→` | Switch panel |
| `Tab` | Next panel |
| `j` / `k` | Select item |
| `Ctrl+J` / `Ctrl+K` | Scroll detail section |
| `e` | Toggle explanations |
| `x` | Toggle calculation details |
| `m` | Toggle heatmap view |
| `Enter` | Jump to selected issue |
| `Esc` | Return to list |

## Flow Matrix

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move cursor |
| `g` / `home` | Jump to top |
| `G` / `end` | Jump to bottom |
| `Tab` | Toggle panel |
| `Enter` | Open drilldown / jump to issue |
| `f` / `q` / `Esc` | Close (exits drilldown first if open) |

## Sprint View

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Next / previous sprint |
| `P` / `Esc` | Exit sprint view |

## Label Dashboard

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Move cursor |
| `home` | Jump to top |
| `G` / `end` | Jump to bottom |
| `Enter` | Filter list by label and return to list |
| `h` | Open label health detail modal |
| `d` | Open drilldown overlay |
| `Esc` | Return to list |

### Label Health Detail Modal

| Key | Action |
|-----|--------|
| `Esc` / `q` / `Enter` / `h` | Close modal |
| `d` | Open drilldown for this label |

### Label Drilldown

| Key | Action |
|-----|--------|
| `Enter` | Apply label filter and close |
| `g` | Graph analysis sub-view |
| `Esc` / `q` / `d` | Close drilldown |

## Label Picker

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` / `Ctrl+N` / `Ctrl+P` | Navigate |
| `Enter` | Apply label filter |
| `Esc` | Cancel |
| _(other keys)_ | Fuzzy search input |

## Recipe Picker

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate |
| `Enter` | Apply recipe |
| `Esc` | Cancel |

## Repo Picker (workspace mode)

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate |
| `Space` | Toggle selection |
| `a` | Select all |
| `Enter` | Apply filter |
| `Esc` / `q` | Cancel |

## Alerts Panel

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate alerts |
| `Enter` | Jump to issue |
| `d` | Dismiss selected alert |
| `Esc` / `q` / `!` | Close panel |

## Help Overlay

| Key | Action |
|-----|--------|
| `j` / `k` | Scroll |
| `Ctrl+D` / `Ctrl+U` | Scroll 10 lines |
| `g` / `home` | Jump to top |
| `G` / `end` | Jump to bottom |
| `Space` | Open interactive tutorial |
| `q` / `Esc` / `?` / `F1` | Close help |
| _(any other key)_ | Dismiss help |

## Time-Travel Input

| Key | Action |
|-----|--------|
| `Enter` | Submit revision (default: HEAD~5) |
| `Esc` | Cancel |

## Modals

### Quit Confirmation

| Key | Action |
|-----|--------|
| `Esc` / `y` / `Y` | Confirm quit |
| _(any other key)_ | Cancel |

### Cass Session Modal

| Key | Action |
|-----|--------|
| `V` / `Esc` / `Enter` / `q` | Close |

### Self-Update Modal

| Key | Action |
|-----|--------|
| `Esc` / `q` | Close (if not in progress) |
| `Enter` | Confirm / close when complete |
| `n` / `N` | Quick cancel |

### Triage Diff Modal

| Key | Action |
|-----|--------|
| `Y` / `y` | Accept |
| `N` / `n` | Reject |
| `Esc` / `q` | Close without accepting |

## Mouse

| Input | Action |
|-------|--------|
| Wheel Up | Scroll up in current view |
| Wheel Down | Scroll down in current view |
