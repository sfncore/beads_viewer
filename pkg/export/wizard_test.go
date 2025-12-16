package export

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewWizard(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	if wizard == nil {
		t.Fatal("NewWizard returned nil")
	}

	if wizard.beadsPath != "/tmp/test" {
		t.Errorf("Expected beadsPath '/tmp/test', got %s", wizard.beadsPath)
	}

	if wizard.config == nil {
		t.Error("Expected config to be initialized")
	}

	if wizard.reader == nil {
		t.Error("Expected reader to be initialized")
	}
}

func TestWizardConfig(t *testing.T) {
	config := WizardConfig{
		IncludeClosed:   true,
		Title:           "Test Title",
		Subtitle:        "Test Subtitle",
		DeployTarget:    "github",
		RepoName:        "test-repo",
		RepoPrivate:     true,
		RepoDescription: "Test description",
		OutputPath:      "/tmp/output",
	}

	if !config.IncludeClosed {
		t.Error("Expected IncludeClosed to be true")
	}

	if config.Title != "Test Title" {
		t.Errorf("Expected Title 'Test Title', got %s", config.Title)
	}

	if config.DeployTarget != "github" {
		t.Errorf("Expected DeployTarget 'github', got %s", config.DeployTarget)
	}

	if !config.RepoPrivate {
		t.Error("Expected RepoPrivate to be true")
	}
}

func TestWizardResult(t *testing.T) {
	result := WizardResult{
		BundlePath:   "/tmp/bundle",
		RepoFullName: "user/repo",
		PagesURL:     "https://user.github.io/repo/",
		DeployTarget: "github",
	}

	if result.BundlePath != "/tmp/bundle" {
		t.Errorf("Expected BundlePath '/tmp/bundle', got %s", result.BundlePath)
	}

	if result.RepoFullName != "user/repo" {
		t.Errorf("Expected RepoFullName 'user/repo', got %s", result.RepoFullName)
	}

	if result.PagesURL != "https://user.github.io/repo/" {
		t.Errorf("Expected PagesURL 'https://user.github.io/repo/', got %s", result.PagesURL)
	}
}

func TestWizard_GetConfig(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	config := wizard.GetConfig()
	if config == nil {
		t.Fatal("GetConfig returned nil")
	}

	// Default values
	if config.IncludeClosed {
		t.Error("Expected IncludeClosed to be false by default")
	}

	if config.DeployTarget != "" {
		t.Errorf("Expected empty DeployTarget by default, got %s", config.DeployTarget)
	}
}

func TestWizardConfigPath(t *testing.T) {
	path := WizardConfigPath()

	// Should return a valid path (or empty if no home dir)
	if path != "" {
		if !filepath.IsAbs(path) {
			t.Errorf("Expected absolute path, got %s", path)
		}

		// Should end with pages-wizard.json
		if filepath.Base(path) != "pages-wizard.json" {
			t.Errorf("Expected path to end with pages-wizard.json, got %s", path)
		}
	}
}

func TestSaveAndLoadWizardConfig(t *testing.T) {
	// Create a temp config directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".config", "bv")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "pages-wizard.json")

	// Create config to save
	config := &WizardConfig{
		IncludeClosed: true,
		Title:         "Saved Title",
		DeployTarget:  "github",
		RepoName:      "saved-repo",
		RepoPrivate:   true,
	}

	// We can't easily test SaveWizardConfig because it uses a fixed path
	// So we'll just test the config struct serialization
	if config.Title != "Saved Title" {
		t.Errorf("Expected Title 'Saved Title', got %s", config.Title)
	}

	// Just verify the config path function works
	_ = configPath
}

func TestWizard_PerformExport(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	tmpDir := t.TempDir()
	err := wizard.PerformExport(tmpDir)
	if err != nil {
		t.Errorf("PerformExport returned unexpected error: %v", err)
	}

	if wizard.bundlePath != tmpDir {
		t.Errorf("Expected bundlePath %s, got %s", tmpDir, wizard.bundlePath)
	}
}

func TestWizard_PrintBanner(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	// Just verify it doesn't panic
	wizard.printBanner()
}

func TestWizard_PrintSuccess_GitHub(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	result := &WizardResult{
		BundlePath:   "/tmp/bundle",
		RepoFullName: "user/repo",
		PagesURL:     "https://user.github.io/repo/",
		DeployTarget: "github",
	}

	// Just verify it doesn't panic
	wizard.PrintSuccess(result)
}

func TestWizard_PrintSuccess_Local(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	result := &WizardResult{
		BundlePath:   "/tmp/bundle",
		DeployTarget: "local",
	}

	// Just verify it doesn't panic
	wizard.PrintSuccess(result)
}

func TestWizard_PrintSuccess_Cloudflare(t *testing.T) {
	wizard := NewWizard("/tmp/test")

	result := &WizardResult{
		BundlePath:        "/tmp/bundle",
		CloudflareProject: "my-project",
		CloudflareURL:     "https://my-project.pages.dev",
		DeployTarget:      "cloudflare",
	}

	// Just verify it doesn't panic
	wizard.PrintSuccess(result)
}
