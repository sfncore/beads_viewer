package watcher

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDebouncer_CoalescesRapidTriggers(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)

	var callCount atomic.Int32

	// Trigger rapidly 10 times
	for i := 0; i < 10; i++ {
		d.Trigger(func() {
			callCount.Add(1)
		})
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for debounce to complete
	time.Sleep(100 * time.Millisecond)

	if count := callCount.Load(); count != 1 {
		t.Errorf("expected 1 callback invocation, got %d", count)
	}
}

func TestDebouncer_Cancel(t *testing.T) {
	d := NewDebouncer(50 * time.Millisecond)

	var called atomic.Bool

	d.Trigger(func() {
		called.Store(true)
	})

	// Cancel before debounce completes
	d.Cancel()

	time.Sleep(100 * time.Millisecond)

	if called.Load() {
		t.Error("callback should not have been invoked after cancel")
	}
}

func TestDebouncer_DefaultDuration(t *testing.T) {
	d := NewDebouncer(0)
	if d.Duration() != DefaultDebounceDuration {
		t.Errorf("expected default duration %v, got %v", DefaultDebounceDuration, d.Duration())
	}
}

func TestWatcher_DetectsFileChange(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		changeMu sync.Mutex
		changed  bool
	)

	w, err := NewWatcher(tmpFile,
		WithDebounceDuration(50*time.Millisecond),
		WithOnChange(func() {
			changeMu.Lock()
			changed = true
			changeMu.Unlock()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Give watcher time to initialize
	time.Sleep(100 * time.Millisecond)

	// Modify file
	if err := os.WriteFile(tmpFile, []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for change detection
	time.Sleep(300 * time.Millisecond)

	changeMu.Lock()
	wasChanged := changed
	changeMu.Unlock()

	if !wasChanged {
		t.Error("expected change to be detected")
	}
}

func TestWatcher_PollingFallback(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		changeMu sync.Mutex
		changed  bool
	)

	w, err := NewWatcher(tmpFile,
		WithDebounceDuration(50*time.Millisecond),
		WithPollInterval(100*time.Millisecond),
		WithForcePoll(true),
		WithOnChange(func() {
			changeMu.Lock()
			changed = true
			changeMu.Unlock()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	if !w.IsPolling() {
		t.Error("expected watcher to be in polling mode")
	}

	// Give polling time to start
	time.Sleep(50 * time.Millisecond)

	// Modify file
	if err := os.WriteFile(tmpFile, []byte("modified via polling"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for polling to detect change
	time.Sleep(300 * time.Millisecond)

	changeMu.Lock()
	wasChanged := changed
	changeMu.Unlock()

	if !wasChanged {
		t.Error("expected change to be detected via polling")
	}
}

func TestWatcher_ChangedChannel(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(tmpFile,
		WithDebounceDuration(50*time.Millisecond),
		WithPollInterval(100*time.Millisecond),
		WithForcePoll(true),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Modify file
	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(tmpFile, []byte("new content"), 0644)
	}()

	// Wait for change via channel
	select {
	case <-w.Changed():
		// Success
	case <-time.After(500 * time.Millisecond):
		t.Error("timeout waiting for change notification")
	}
}

func TestWatcher_FileRemoved(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		errMu     sync.Mutex
		gotError  error
	)

	w, err := NewWatcher(tmpFile,
		WithDebounceDuration(50*time.Millisecond),
		WithPollInterval(100*time.Millisecond),
		WithForcePoll(true),
		WithOnError(func(err error) {
			errMu.Lock()
			gotError = err
			errMu.Unlock()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}
	defer w.Stop()

	// Give watcher time to start
	time.Sleep(50 * time.Millisecond)

	// Remove file
	if err := os.Remove(tmpFile); err != nil {
		t.Fatal(err)
	}

	// Wait for error detection
	time.Sleep(300 * time.Millisecond)

	errMu.Lock()
	receivedError := gotError
	errMu.Unlock()

	if receivedError != ErrFileRemoved {
		t.Errorf("expected ErrFileRemoved, got %v", receivedError)
	}
}

func TestWatcher_StartStop(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if w.IsStarted() {
		t.Error("watcher should not be started initially")
	}

	if err := w.Start(); err != nil {
		t.Fatal(err)
	}

	if !w.IsStarted() {
		t.Error("watcher should be started after Start()")
	}

	// Double start should error
	if err := w.Start(); err != ErrAlreadyStarted {
		t.Errorf("expected ErrAlreadyStarted, got %v", err)
	}

	w.Stop()

	if w.IsStarted() {
		t.Error("watcher should not be started after Stop()")
	}

	// Double stop should be safe
	w.Stop()
}

func TestWatcher_Path(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")

	if err := os.WriteFile(tmpFile, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}

	w, err := NewWatcher(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	absPath, _ := filepath.Abs(tmpFile)
	if w.Path() != absPath {
		t.Errorf("expected path %s, got %s", absPath, w.Path())
	}
}
