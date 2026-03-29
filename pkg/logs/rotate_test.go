package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --------------------------------------------------------------------------
// TestRotatingFileWriter_Write_Basic
// Verify that a small write succeeds and returns correct n.
// --------------------------------------------------------------------------

func TestRotatingFileWriter_Write_Basic(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter: %v", err)
	}
	defer w.Close()

	data := []byte("hello world\n")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned n=%d, want %d", n, len(data))
	}
}

// --------------------------------------------------------------------------
// TestRotatingFileWriter_RotatesOnSize
// MaxSize=50 bytes. Write 31 bytes twice → second write triggers rotation
// so the directory must contain at least 2 files.
// --------------------------------------------------------------------------

func TestRotatingFileWriter_RotatesOnSize(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{
		MaxSize:    50,
		MaxBackups: 10,
		MaxAge:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter: %v", err)
	}
	defer w.Close()

	payload := strings.Repeat("x", 31)
	if _, err := w.Write([]byte(payload)); err != nil {
		t.Fatalf("first Write: %v", err)
	}
	if _, err := w.Write([]byte(payload)); err != nil {
		t.Fatalf("second Write: %v", err)
	}

	w.waitCleanup()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("expected at least 2 files after rotation, got %d", len(entries))
	}
}

// --------------------------------------------------------------------------
// TestRotatingFileWriter_MaxBackups
// MaxSize=20, MaxBackups=2. Write 10 times (each 10 bytes) → many rotations.
// After waitCleanup the directory must contain at most 3 files
// (1 current + 2 backups).
// --------------------------------------------------------------------------

func TestRotatingFileWriter_MaxBackups(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "app"), ".log", RotateConfig{
		MaxSize:    20,
		MaxBackups: 2,
		MaxAge:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter: %v", err)
	}
	defer w.Close()

	payload := strings.Repeat("y", 10)
	for i := 0; i < 10; i++ {
		if _, err := w.Write([]byte(payload)); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
		// Small sleep so timestamp-based filenames do not collide.
		time.Sleep(time.Millisecond)
	}

	w.waitCleanup()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) > 3 {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Errorf("expected at most 3 files, got %d: %v", len(entries), names)
	}
}

// --------------------------------------------------------------------------
// TestRotatingFileWriter_ExtFollowsFormat
// Use .json extension; all files in the directory must end with ".json".
// --------------------------------------------------------------------------

func TestRotatingFileWriter_ExtFollowsFormat(t *testing.T) {
	dir := t.TempDir()
	w, err := NewRotatingFileWriter(filepath.Join(dir, "data"), ".json", RotateConfig{
		MaxSize:    30,
		MaxBackups: 10,
		MaxAge:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter: %v", err)
	}
	defer w.Close()

	payload := strings.Repeat("z", 20)
	for i := 0; i < 3; i++ {
		if _, err := w.Write([]byte(payload)); err != nil {
			t.Fatalf("Write %d: %v", i, err)
		}
		time.Sleep(time.Millisecond)
	}

	w.waitCleanup()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			t.Errorf("unexpected file without .json extension: %s", e.Name())
		}
	}
}

// --------------------------------------------------------------------------
// TestRotatingFileWriter_CleanupOnlyOnRotation
// MaxSize=1024 (large enough to never trigger rotation).
// Plant fake old files in the directory, write a small payload.
// Cleanup must NOT run → old fake files must still exist.
// --------------------------------------------------------------------------

func TestRotatingFileWriter_CleanupOnlyOnRotation(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "app")

	// Plant two fake "old" backup files.
	fakeFiles := []string{
		base + ".20200101-000000.log",
		base + ".20200102-000000.log",
	}
	for _, p := range fakeFiles {
		if err := os.WriteFile(p, []byte("old data"), 0o644); err != nil {
			t.Fatalf("WriteFile fake: %v", err)
		}
	}

	w, err := NewRotatingFileWriter(base, ".log", RotateConfig{
		MaxSize:    1024,
		MaxBackups: 10,
		MaxAge:     24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("NewRotatingFileWriter: %v", err)
	}
	defer w.Close()

	// Write something small — must NOT trigger rotation.
	if _, err := w.Write([]byte("tiny payload\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}

	w.waitCleanup()

	// Fake old files must still be present.
	for _, p := range fakeFiles {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			t.Errorf("fake old file was unexpectedly removed: %s", p)
		}
	}
}
