package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigFromRoutes(t *testing.T) {
	// Create a temporary routes.jsonl
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	routesContent := `{"prefix":"bv-","path":"bv/mayor/rig"}
{"prefix":"st-","path":"sfgastown/mayor/rig"}
{"prefix":"hq-","path":"."}
{"prefix":"st","path":"/absolute/path/.beads"}
{"prefix":"hq-cv-","path":"."}
{"prefix":"fc-","path":"frankencord"}
`
	routesPath := filepath.Join(beadsDir, "routes.jsonl")
	if err := os.WriteFile(routesPath, []byte(routesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfigFromRoutes(routesPath)
	if err != nil {
		t.Fatalf("LoadConfigFromRoutes() error = %v", err)
	}

	if config.Name != "gas-town" {
		t.Errorf("config.Name = %q, want %q", config.Name, "gas-town")
	}

	// Should have 4 repos: bv-, st-, hq-, fc-
	// "st" (no dash) should be skipped
	// "hq-cv-" (sub-prefix) should be skipped
	if len(config.Repos) != 4 {
		t.Errorf("len(config.Repos) = %d, want 4", len(config.Repos))
		for _, r := range config.Repos {
			t.Logf("  repo: name=%s prefix=%s path=%s", r.Name, r.Prefix, r.Path)
		}
	}

	// Check that repos are sorted by name
	for i := 1; i < len(config.Repos); i++ {
		if config.Repos[i].Name < config.Repos[i-1].Name {
			t.Errorf("repos not sorted: %q comes after %q", config.Repos[i].Name, config.Repos[i-1].Name)
		}
	}

	// Check specific entries
	repoMap := make(map[string]RepoConfig)
	for _, r := range config.Repos {
		repoMap[r.Prefix] = r
	}

	if r, ok := repoMap["bv-"]; ok {
		if r.Name != "bv" {
			t.Errorf("bv repo name = %q, want %q", r.Name, "bv")
		}
	} else {
		t.Error("missing bv- repo")
	}

	if r, ok := repoMap["hq-"]; ok {
		if r.Name != "hq" {
			t.Errorf("hq repo name = %q, want %q", r.Name, "hq")
		}
	} else {
		t.Error("missing hq- repo")
	}

	if r, ok := repoMap["fc-"]; ok {
		if r.Name != "frankencord" {
			t.Errorf("fc repo name = %q, want %q", r.Name, "frankencord")
		}
	} else {
		t.Error("missing fc- repo")
	}
}

func TestLoadConfigFromRoutesDeduplicatesPath(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// bd- and pa- both point to the same path — only the first should be kept
	routesContent := `{"prefix":"bd-","path":"pi_agent/mayor/rig"}
{"prefix":"pa-","path":"pi_agent/mayor/rig"}
`
	routesPath := filepath.Join(beadsDir, "routes.jsonl")
	if err := os.WriteFile(routesPath, []byte(routesContent), 0o644); err != nil {
		t.Fatal(err)
	}

	config, err := LoadConfigFromRoutes(routesPath)
	if err != nil {
		t.Fatalf("LoadConfigFromRoutes() error = %v", err)
	}

	if len(config.Repos) != 1 {
		t.Errorf("len(config.Repos) = %d, want 1 (should deduplicate by path)", len(config.Repos))
		for _, r := range config.Repos {
			t.Logf("  repo: name=%s prefix=%s path=%s", r.Name, r.Prefix, r.Path)
		}
	}
}

func TestIsRoutesFile(t *testing.T) {
	tmpDir := t.TempDir()

	// JSONL file
	jsonlPath := filepath.Join(tmpDir, "routes.jsonl")
	os.WriteFile(jsonlPath, []byte(`{"prefix":"bv-"}`), 0o644)
	if !IsRoutesFile(jsonlPath) {
		t.Error("expected routes.jsonl to be detected as routes file")
	}

	// YAML file
	yamlPath := filepath.Join(tmpDir, "workspace.yaml")
	os.WriteFile(yamlPath, []byte("name: test\nrepos:\n"), 0o644)
	if IsRoutesFile(yamlPath) {
		t.Error("expected workspace.yaml to NOT be detected as routes file")
	}

	// JSON content without .jsonl extension
	jsonPath := filepath.Join(tmpDir, "routes.txt")
	os.WriteFile(jsonPath, []byte(`{"prefix":"bv-","path":"bv"}`), 0o644)
	if !IsRoutesFile(jsonPath) {
		t.Error("expected JSON-content file to be detected as routes file")
	}
}

func TestExtractRigName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"bv/mayor/rig", "bv"},
		{"frankencord", "frankencord"},
		{".", ""},
		{"", ""},
		{"./bv/mayor/rig", "bv"},
		{"sfgastown/mayor/rig", "sfgastown"},
	}

	for _, tt := range tests {
		got := extractRigName(tt.path)
		if got != tt.want {
			t.Errorf("extractRigName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
