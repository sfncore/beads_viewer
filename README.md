# Beads Viewer (bv)

**Project Intelligence for Beads.**

`bv` is a slick, polished Terminal User Interface (TUI) for the [Beads](https://github.com/steveyegge/beads) issue tracker. It transforms your flat list of issues into a visual, interactive workspace with advanced graph theory analytics and insights.

## Features

### 🖥️ Visual Workspace
*   **Adaptive Split-View Dashboard**: On wide screens (>100 cols), `bv` automatically transitions to a master-detail view, putting your issue list side-by-side with rich context.
*   **Kanban Board**: Toggle a 4-column Kanban board (Open, In Progress, Blocked, Closed) with `b` to visualize flow.
*   **Ultra-Wide Density**: On large monitors (>140 cols), lists expand to show label tags, comment counts, and relative ages.
*   **Visual Polish**: A vibrant "Dracula" theme with emoji status icons (🐛, ✨, 🏔️) and priority badges (🔥, ⚡).

### 🧠 Deep Analytics
*   **Graph Theory Engine**: `bv` builds a Directed Acyclic Graph (DAG) of your project's dependencies to uncover structural insights.
*   **Impact Scores**: Automatically identifies "Keystone" tasks—those blocking the deepest chains of downstream work—and visualizes them with Sparklines (`███▌`) and Heatmaps (🌋/🏔️) in Ultra-Wide mode.
*   **Insights Dashboard**: Press `i` to open a dedicated panel showing:
    *   **Bottlenecks**: Tasks with high Betweenness Centrality.
    *   **Keystones**: High-impact tasks critical to the critical path.
    *   **Network Health**: Cycle detection and cluster density metrics.

### ⚡ Workflow & Integration
*   **Instant Filtering**: Filter by status with single keystrokes: `o` (Open), `r` (Ready/Unblocked), `c` (Closed), `a` (All).
*   **Markdown Export**: Generate comprehensive status reports with `bv --export-md report.md`. Includes embedded **Mermaid.js** dependency graphs that render visually on GitHub/GitLab.
*   **Smart Search**: Fuzzy search across Titles, IDs, Assignees, and Labels.
*   **Self-Updating**: Automatically checks for and notifies you of new releases.

## Installation

### Quick Install
```bash
curl -fsSL https://raw.githubusercontent.com/Dicklesworthstone/beads_viewer/master/install.sh | bash
```

### Build from Source
```bash
go install github.com/Dicklesworthstone/beads_viewer/cmd/bv@latest
```

## Usage

Navigate to any project initialized with `bd init` and run:

```bash
bv
```

### Controls

| Key | Context | Action |
| :--- | :--- | :--- |
| `b` | Global | Toggle **Kanban Board** |
| `i` | Global | Toggle **Insights Dashboard** |
| `Tab` | Split View | Switch focus between List and Details |
| `h`/`j`/`k`/`l`| Global | Navigate (Vim style) |
| `Enter` | List | Open/Focus details |
| `o` / `r` / `c` | Global | Filter by Status |
| `/` | List | Start Search |
| `q` | Global | Quit |

## CI/CD

This project uses GitHub Actions to run full unit and end-to-end tests on every push and automatically builds optimized binaries for Linux, macOS, and Windows on every release tag.

## License

MIT