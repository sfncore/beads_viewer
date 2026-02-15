package workspace

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// routeEntry represents a single line in routes.jsonl
type routeEntry struct {
	Prefix string `json:"prefix"`
	Path   string `json:"path"`
}

// LoadConfigFromRoutes builds a workspace Config from a beads routes.jsonl file.
// This enables bv to auto-discover rigs in a Gas Town workspace by reading the
// routing table that bd already maintains.
//
// The rig name is extracted from the first path component (e.g., "bv/mayor/rig" → "bv").
// Only entries with a trailing "-" prefix are included (e.g., "bv-" but not "bv").
// Sub-prefixes (containing more than one "-") are skipped to avoid duplicates.
func LoadConfigFromRoutes(routesPath string) (*Config, error) {
	f, err := os.Open(routesPath)
	if err != nil {
		return nil, fmt.Errorf("opening routes file: %w", err)
	}
	defer f.Close()

	basePath := filepath.Dir(routesPath)
	// routes.jsonl is typically at ~/gt/.beads/routes.jsonl, so base is ~/gt/.beads/
	// Paths in routes.jsonl are relative to the Gas Town root (parent of .beads/)
	gtRoot := filepath.Dir(basePath)

	var entries []routeEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry routeEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading routes file: %w", err)
	}

	// Build repo configs from route entries.
	// Only include entries where prefix ends with "-" (standard rig prefixes).
	// Skip bare prefixes (no dash) and sub-prefixes (multiple dashes like "hq-cv-").
	seenPrefix := make(map[string]bool)
	seenPath := make(map[string]bool)
	var repos []RepoConfig

	for _, entry := range entries {
		// Must end with "-"
		if !strings.HasSuffix(entry.Prefix, "-") {
			continue
		}

		// Skip sub-prefixes (more than one "-")
		if strings.Count(entry.Prefix, "-") > 1 {
			continue
		}

		// Resolve relative paths against gtRoot
		repoPath := entry.Path
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(gtRoot, repoPath)
		}

		// Extract rig name from first path component
		rigName := extractRigName(entry.Path)
		if rigName == "" {
			rigName = strings.TrimSuffix(entry.Prefix, "-")
		}

		// Deduplicate by prefix and by resolved path.
		// routes.jsonl can have multiple prefixes pointing to the same path
		// (e.g., bd- and pa- both pointing to pi_agent/mayor/rig).
		// Prefer the entry whose prefix matches the rig name.
		if seenPrefix[entry.Prefix] {
			continue
		}
		resolvedPath := filepath.Clean(repoPath)
		if seenPath[resolvedPath] {
			continue
		}
		seenPrefix[entry.Prefix] = true
		seenPath[resolvedPath] = true

		repos = append(repos, RepoConfig{
			Name:         rigName,
			Path:         repoPath,
			Prefix:       entry.Prefix,
			DoltDatabase: rigName,
		})
	}

	// Sort by name for stable output
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Name < repos[j].Name
	})

	config := &Config{
		Name:  "gas-town",
		Repos: repos,
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("generated config invalid: %w", err)
	}

	return config, nil
}

// extractRigName extracts the rig name from a routes.jsonl path.
// e.g., "bv/mayor/rig" → "bv", "frankencord" → "frankencord", "." → ""
func extractRigName(path string) string {
	path = strings.TrimPrefix(path, "./")
	if path == "" || path == "." {
		return ""
	}
	parts := strings.SplitN(path, "/", 2)
	return parts[0]
}

// IsRoutesFile checks if a file path looks like a routes.jsonl file
// (as opposed to a workspace.yaml file).
func IsRoutesFile(path string) bool {
	if strings.HasSuffix(path, ".jsonl") {
		return true
	}
	// Check first line for JSON
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		return strings.HasPrefix(line, "{")
	}
	return false
}
