// Package correlation provides file-to-bead reverse index functionality.
package correlation

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BeadReference links a bead to a file via commits.
type BeadReference struct {
	BeadID     string    `json:"bead_id"`
	Title      string    `json:"title"`
	Status     string    `json:"status"`     // open/in_progress/closed
	CommitSHAs []string  `json:"commit_shas"` // which commits linked this bead to this file
	LastTouch  time.Time `json:"last_touch"`  // most recent commit timestamp
	TotalChanges int     `json:"total_changes"` // sum of insertions + deletions across commits
}

// FileBeadIndex provides O(1) lookup from file path to beads that touched it.
type FileBeadIndex struct {
	// FileToBeads maps normalized file paths to beads that modified them
	FileToBeads map[string][]BeadReference `json:"file_to_beads"`

	// Stats provides aggregate information about the index
	Stats FileIndexStats `json:"stats"`
}

// FileIndexStats contains aggregate statistics about the file index.
type FileIndexStats struct {
	TotalFiles       int `json:"total_files"`       // number of unique files
	TotalBeadLinks   int `json:"total_bead_links"`  // sum of all bead references
	FilesWithMultipleBeads int `json:"files_with_multiple_beads"` // files touched by >1 bead
}

// FileBeadLookupResult is the result of looking up beads for a file.
type FileBeadLookupResult struct {
	FilePath    string          `json:"file_path"`
	OpenBeads   []BeadReference `json:"open_beads"`   // currently open beads
	ClosedBeads []BeadReference `json:"closed_beads"` // recently closed beads
	TotalBeads  int             `json:"total_beads"`
}

// FileLookup provides file-to-bead lookup functionality.
type FileLookup struct {
	index *FileBeadIndex
	beads map[string]BeadHistory // BeadID -> history for status lookups
}

// BuildFileIndex creates a file index from a history report.
// It extracts all file paths from correlated commits and maps them to beads.
func BuildFileIndex(report *HistoryReport) *FileBeadIndex {
	if report == nil {
		return &FileBeadIndex{
			FileToBeads: make(map[string][]BeadReference),
		}
	}

	// fileBeadMap: file -> beadID -> reference (for deduplication)
	fileBeadMap := make(map[string]map[string]*BeadReference)

	for beadID, history := range report.Histories {
		for _, commit := range history.Commits {
			for _, file := range commit.Files {
				// Normalize path (remove leading ./ and normalize separators)
				normalizedPath := normalizePath(file.Path)

				if fileBeadMap[normalizedPath] == nil {
					fileBeadMap[normalizedPath] = make(map[string]*BeadReference)
				}

				ref := fileBeadMap[normalizedPath][beadID]
				if ref == nil {
					ref = &BeadReference{
						BeadID:     beadID,
						Title:      history.Title,
						Status:     history.Status,
						CommitSHAs: []string{},
						LastTouch:  commit.Timestamp,
					}
					fileBeadMap[normalizedPath][beadID] = ref
				}

				// Add commit SHA if not already present
				found := false
				for _, sha := range ref.CommitSHAs {
					if sha == commit.ShortSHA {
						found = true
						break
					}
				}
				if !found {
					ref.CommitSHAs = append(ref.CommitSHAs, commit.ShortSHA)
				}

				// Update last touch time if this commit is more recent
				if commit.Timestamp.After(ref.LastTouch) {
					ref.LastTouch = commit.Timestamp
				}

				// Accumulate changes
				ref.TotalChanges += file.Insertions + file.Deletions
			}
		}
	}

	// Convert to final structure
	result := &FileBeadIndex{
		FileToBeads: make(map[string][]BeadReference),
	}

	totalLinks := 0
	multipleBeadsCount := 0

	for filePath, beadMap := range fileBeadMap {
		refs := make([]BeadReference, 0, len(beadMap))
		for _, ref := range beadMap {
			refs = append(refs, *ref)
		}

		// Sort by last touch time (most recent first)
		sort.Slice(refs, func(i, j int) bool {
			return refs[i].LastTouch.After(refs[j].LastTouch)
		})

		result.FileToBeads[filePath] = refs
		totalLinks += len(refs)
		if len(refs) > 1 {
			multipleBeadsCount++
		}
	}

	result.Stats = FileIndexStats{
		TotalFiles:             len(result.FileToBeads),
		TotalBeadLinks:         totalLinks,
		FilesWithMultipleBeads: multipleBeadsCount,
	}

	return result
}

// NewFileLookup creates a file lookup from a history report.
func NewFileLookup(report *HistoryReport) *FileLookup {
	if report == nil {
		return &FileLookup{
			index: BuildFileIndex(nil),
			beads: make(map[string]BeadHistory),
		}
	}
	return &FileLookup{
		index: BuildFileIndex(report),
		beads: report.Histories,
	}
}

// LookupByFile finds all beads that have touched a given file.
// The path can be exact or a prefix (for directory lookups).
func (fl *FileLookup) LookupByFile(path string) *FileBeadLookupResult {
	normalizedPath := normalizePath(path)

	result := &FileBeadLookupResult{
		FilePath:    path,
		OpenBeads:   []BeadReference{},
		ClosedBeads: []BeadReference{},
	}

	// Try exact match first
	if refs, ok := fl.index.FileToBeads[normalizedPath]; ok {
		for _, ref := range refs {
			// Get current status from beads map (may have changed)
			if history, ok := fl.beads[ref.BeadID]; ok {
				ref.Status = history.Status
				ref.Title = history.Title
			}

			if ref.Status == "closed" {
				result.ClosedBeads = append(result.ClosedBeads, ref)
			} else {
				result.OpenBeads = append(result.OpenBeads, ref)
			}
		}
		result.TotalBeads = len(refs)
		return result
	}

	// Try prefix match for directory lookups
	// Note: normalizePath converts all backslashes to forward slashes, so we only need to check "/"
	for filePath, refs := range fl.index.FileToBeads {
		if strings.HasPrefix(filePath, normalizedPath+"/") {
			for _, ref := range refs {
				// Get current status
				if history, ok := fl.beads[ref.BeadID]; ok {
					ref.Status = history.Status
					ref.Title = history.Title
				}

				// Avoid duplicates across files in directory
				if ref.Status == "closed" {
					if !containsBeadRef(result.ClosedBeads, ref.BeadID) {
						result.ClosedBeads = append(result.ClosedBeads, ref)
					}
				} else {
					if !containsBeadRef(result.OpenBeads, ref.BeadID) {
						result.OpenBeads = append(result.OpenBeads, ref)
					}
				}
			}
		}
	}

	result.TotalBeads = len(result.OpenBeads) + len(result.ClosedBeads)
	return result
}

// LookupByFileGlob finds beads for files matching a glob pattern.
func (fl *FileLookup) LookupByFileGlob(pattern string) *FileBeadLookupResult {
	result := &FileBeadLookupResult{
		FilePath:    pattern,
		OpenBeads:   []BeadReference{},
		ClosedBeads: []BeadReference{},
	}

	// Track seen beads to avoid duplicates
	seenOpen := make(map[string]bool)
	seenClosed := make(map[string]bool)

	for filePath, refs := range fl.index.FileToBeads {
		matched, err := filepath.Match(pattern, filePath)
		if err != nil || !matched {
			continue
		}

		for _, ref := range refs {
			// Get current status
			if history, ok := fl.beads[ref.BeadID]; ok {
				ref.Status = history.Status
				ref.Title = history.Title
			}

			if ref.Status == "closed" {
				if !seenClosed[ref.BeadID] {
					result.ClosedBeads = append(result.ClosedBeads, ref)
					seenClosed[ref.BeadID] = true
				}
			} else {
				if !seenOpen[ref.BeadID] {
					result.OpenBeads = append(result.OpenBeads, ref)
					seenOpen[ref.BeadID] = true
				}
			}
		}
	}

	result.TotalBeads = len(result.OpenBeads) + len(result.ClosedBeads)
	return result
}

// GetAllFiles returns all files in the index, sorted by path.
func (fl *FileLookup) GetAllFiles() []string {
	files := make([]string, 0, len(fl.index.FileToBeads))
	for path := range fl.index.FileToBeads {
		files = append(files, path)
	}
	sort.Strings(files)
	return files
}

// GetStats returns statistics about the file index.
func (fl *FileLookup) GetStats() FileIndexStats {
	return fl.index.Stats
}

// GetHotspots returns files touched by the most beads (potential conflict zones).
func (fl *FileLookup) GetHotspots(limit int) []FileHotspot {
	type fileBeadCount struct {
		path  string
		count int
		refs  []BeadReference
	}

	var counts []fileBeadCount
	for path, refs := range fl.index.FileToBeads {
		counts = append(counts, fileBeadCount{
			path:  path,
			count: len(refs),
			refs:  refs,
		})
	}

	// Sort by count descending
	sort.Slice(counts, func(i, j int) bool {
		return counts[i].count > counts[j].count
	})

	// Take top N
	if limit <= 0 || limit > len(counts) {
		limit = len(counts)
	}

	hotspots := make([]FileHotspot, 0, limit)
	for i := 0; i < limit; i++ {
		c := counts[i]

		// Count open vs closed using current status from fl.beads
		openCount := 0
		for _, ref := range c.refs {
			// Get current status from beads map (may have changed since index was built)
			status := ref.Status
			if history, ok := fl.beads[ref.BeadID]; ok {
				status = history.Status
			}
			if status != "closed" {
				openCount++
			}
		}

		hotspots = append(hotspots, FileHotspot{
			FilePath:    c.path,
			TotalBeads:  c.count,
			OpenBeads:   openCount,
			ClosedBeads: c.count - openCount,
		})
	}

	return hotspots
}

// FileHotspot represents a file that has been touched by many beads.
type FileHotspot struct {
	FilePath    string `json:"file_path"`
	TotalBeads  int    `json:"total_beads"`
	OpenBeads   int    `json:"open_beads"`
	ClosedBeads int    `json:"closed_beads"`
}

// ImpactResult is the result of analyzing what beads might be affected by file changes.
type ImpactResult struct {
	Files         []string       `json:"files"`
	AffectedBeads []AffectedBead `json:"affected_beads"`
	RiskLevel     string         `json:"risk_level"`
	RiskScore     float64        `json:"risk_score"`
	Warnings      []string       `json:"warnings"`
	Summary       string         `json:"summary"`
}

// AffectedBead represents a bead that touches one or more of the analyzed files.
type AffectedBead struct {
	BeadID       string    `json:"bead_id"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	OverlapFiles []string  `json:"overlap_files"`
	OverlapCount int       `json:"overlap_count"`
	LastActivity time.Time `json:"last_activity"`
	Relevance    float64   `json:"relevance"`
	TotalChanges int       `json:"total_changes"`
}

// ImpactAnalysis analyzes what beads might be affected if the given files are modified.
func (fl *FileLookup) ImpactAnalysis(files []string) *ImpactResult {
	result := &ImpactResult{
		Files:         files,
		AffectedBeads: []AffectedBead{},
		RiskLevel:     "low",
		RiskScore:     0.0,
		Warnings:      []string{},
	}

	if len(files) == 0 {
		result.Summary = "No files to analyze"
		return result
	}

	normalizedFiles := make([]string, len(files))
	for i, f := range files {
		normalizedFiles[i] = normalizePath(f)
	}

	beadMap := make(map[string]*AffectedBead)
	now := time.Now()

	for _, filePath := range normalizedFiles {
		lookup := fl.LookupByFile(filePath)

		for _, ref := range lookup.OpenBeads {
			ab := beadMap[ref.BeadID]
			if ab == nil {
				ab = &AffectedBead{
					BeadID:       ref.BeadID,
					Title:        ref.Title,
					Status:       ref.Status,
					OverlapFiles: []string{},
					LastActivity: ref.LastTouch,
				}
				beadMap[ref.BeadID] = ab
			}
			ab.OverlapFiles = append(ab.OverlapFiles, filePath)
			ab.OverlapCount = len(ab.OverlapFiles)
			ab.TotalChanges += ref.TotalChanges
			if ref.LastTouch.After(ab.LastActivity) {
				ab.LastActivity = ref.LastTouch
			}
		}

		for _, ref := range lookup.ClosedBeads {
			if now.Sub(ref.LastTouch) > 7*24*time.Hour {
				continue
			}
			ab := beadMap[ref.BeadID]
			if ab == nil {
				ab = &AffectedBead{
					BeadID:       ref.BeadID,
					Title:        ref.Title,
					Status:       ref.Status,
					OverlapFiles: []string{},
					LastActivity: ref.LastTouch,
				}
				beadMap[ref.BeadID] = ab
			}
			ab.OverlapFiles = append(ab.OverlapFiles, filePath)
			ab.OverlapCount = len(ab.OverlapFiles)
			ab.TotalChanges += ref.TotalChanges
			if ref.LastTouch.After(ab.LastActivity) {
				ab.LastActivity = ref.LastTouch
			}
		}
	}

	openCount := 0
	inProgressCount := 0
	recentClosedCount := 0

	for _, ab := range beadMap {
		daysSince := now.Sub(ab.LastActivity).Hours() / 24
		recencyScore := 1.0 - (daysSince / 7.0)
		if recencyScore < 0 {
			recencyScore = 0
		}
		overlapScore := float64(ab.OverlapCount) / float64(len(files))
		statusMultiplier := 0.5
		if ab.Status == "in_progress" {
			statusMultiplier = 1.0
			inProgressCount++
		} else if ab.Status == "open" {
			statusMultiplier = 0.8
			openCount++
		} else {
			recentClosedCount++
		}
		ab.Relevance = (recencyScore*0.4 + overlapScore*0.4 + statusMultiplier*0.2)
		result.AffectedBeads = append(result.AffectedBeads, *ab)
	}

	sort.Slice(result.AffectedBeads, func(i, j int) bool {
		statusPriority := map[string]int{"in_progress": 0, "open": 1, "closed": 2}
		pi, pj := statusPriority[result.AffectedBeads[i].Status], statusPriority[result.AffectedBeads[j].Status]
		if pi != pj {
			return pi < pj
		}
		return result.AffectedBeads[i].Relevance > result.AffectedBeads[j].Relevance
	})

	result.RiskScore = float64(inProgressCount)*0.4 + float64(openCount)*0.2 + float64(recentClosedCount)*0.05
	if len(files) > 3 {
		result.RiskScore += 0.1
	}
	if result.RiskScore > 1.0 {
		result.RiskScore = 1.0
	}

	switch {
	case result.RiskScore >= 0.7:
		result.RiskLevel = "critical"
	case result.RiskScore >= 0.4:
		result.RiskLevel = "high"
	case result.RiskScore >= 0.2:
		result.RiskLevel = "medium"
	default:
		result.RiskLevel = "low"
	}

	if inProgressCount > 0 {
		result.Warnings = append(result.Warnings, "Active work in progress on these files - coordinate before making changes")
	}
	if openCount > 0 {
		result.Warnings = append(result.Warnings, "Open beads touch these files - review before modifying")
	}

	total := inProgressCount + openCount + recentClosedCount
	if total == 0 {
		result.Summary = "No beads found touching these files - safe to proceed"
	} else {
		parts := []string{}
		if inProgressCount > 0 {
			parts = append(parts, fmt.Sprintf("%d bead(s) in progress", inProgressCount))
		}
		if openCount > 0 {
			parts = append(parts, fmt.Sprintf("%d open bead(s)", openCount))
		}
		if recentClosedCount > 0 {
			parts = append(parts, fmt.Sprintf("%d recently closed bead(s)", recentClosedCount))
		}
		prefix := "Found "
		if inProgressCount > 0 {
			prefix = "⚠️ Conflict risk: "
		}
		result.Summary = prefix + strings.Join(parts, ", ") + " touching these files"
	}

	return result
}

// Helper functions

// normalizePath normalizes a file path for consistent lookup.
func normalizePath(path string) string {
	// Normalize backslashes to forward slashes first (before prefix removal)
	path = strings.ReplaceAll(path, "\\", "/")

	// Remove leading ./ or ./
	path = strings.TrimPrefix(path, "./")

	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	return path
}

// containsBeadRef checks if a slice contains a bead reference with the given ID.
func containsBeadRef(refs []BeadReference, beadID string) bool {
	for _, ref := range refs {
		if ref.BeadID == beadID {
			return true
		}
	}
	return false
}
