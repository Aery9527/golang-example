// Package logs provides file-based logging utilities with rotation support.
package logs

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxSize    = 100 * 1024 * 1024 // 100 MB
	defaultMaxBackups = 10
	defaultMaxAgeDays = 7
)

// RotateConfig holds configuration for the RotatingFileWriter.
type RotateConfig struct {
	// MaxSize is the maximum size in bytes before rotation. Default: 100 MB.
	MaxSize int64
	// MaxBackups is the maximum number of rotated backup files to retain. Default: 10.
	MaxBackups int
	// MaxAge is the maximum age of backup files before deletion. Default: 7 days.
	MaxAge time.Duration
	// Compress enables gzip compression of rotated backup files.
	Compress bool
}

func (c *RotateConfig) applyDefaults() {
	if c.MaxSize <= 0 {
		c.MaxSize = defaultMaxSize
	}
	if c.MaxBackups <= 0 {
		c.MaxBackups = defaultMaxBackups
	}
	if c.MaxAge <= 0 {
		c.MaxAge = defaultMaxAgeDays * 24 * time.Hour
	}
}

// RotatingFileWriter is a concurrent-safe io.WriteCloser that rotates the
// underlying file when it exceeds MaxSize. Cleanup (MaxBackups, MaxAge,
// optional gzip compression) runs in a background goroutine on each rotation.
type RotatingFileWriter struct {
	basePath string
	ext      string
	cfg      RotateConfig

	mu      sync.Mutex
	file    *os.File
	size    int64

	wg sync.WaitGroup
}

// NewRotatingFileWriter creates and opens a RotatingFileWriter.
// basePath is the full path without extension (e.g. "/var/log/app").
// ext is the file extension including the dot (e.g. ".log").
func NewRotatingFileWriter(basePath, ext string, cfg RotateConfig) (*RotatingFileWriter, error) {
	cfg.applyDefaults()

	w := &RotatingFileWriter{
		basePath: basePath,
		ext:      ext,
		cfg:      cfg,
	}

	if err := w.openFile(); err != nil {
		return nil, fmt.Errorf("logs: open initial log file: %w", err)
	}

	return w, nil
}

// Write implements io.Writer.
// It acquires the mutex, rotates the file when necessary, then appends p.
func (w *RotatingFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.size+int64(len(p)) > w.cfg.MaxSize {
		if err := w.rotate(); err != nil {
			return 0, fmt.Errorf("logs: rotate: %w", err)
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	if err != nil {
		return n, fmt.Errorf("logs: write: %w", err)
	}

	return n, nil
}

// Close implements io.Closer. Flushes and closes the current log file.
// It does NOT wait for background cleanup goroutines; call waitCleanup for that.
func (w *RotatingFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.closeFile()
}

// waitCleanup blocks until all background cleanup goroutines have finished.
// Intended for use in tests.
func (w *RotatingFileWriter) waitCleanup() {
	w.wg.Wait()
}

// --------------------------------------------------------------------------
// internal helpers — must be called with w.mu held (except from New).
// --------------------------------------------------------------------------

// currentPath returns the active (non-rotated) log file path.
func (w *RotatingFileWriter) currentPath() string {
	return w.basePath + w.ext
}

// openFile creates the directory tree if necessary, then opens (or creates)
// the current log file in append mode and records its current size.
func (w *RotatingFileWriter) openFile() error {
	dir := filepath.Dir(w.basePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	f, err := os.OpenFile(w.currentPath(), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", w.currentPath(), err)
	}

	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("stat %s: %w", w.currentPath(), err)
	}

	w.file = f
	w.size = info.Size()
	return nil
}

// closeFile flushes and closes the current file handle.
func (w *RotatingFileWriter) closeFile() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.size = 0
	return err
}

// rotate closes the current file, renames it with a timestamp suffix, opens a
// new file, and spawns a background goroutine to run cleanup.
func (w *RotatingFileWriter) rotate() error {
	if err := w.closeFile(); err != nil {
		return fmt.Errorf("close before rotate: %w", err)
	}

	ts := time.Now().Format("20060102-150405")
	rotatedPath := fmt.Sprintf("%s.%s%s", w.basePath, ts, w.ext)

	if err := os.Rename(w.currentPath(), rotatedPath); err != nil {
		// If the current file doesn't exist yet there is nothing to rename.
		if !os.IsNotExist(err) {
			return fmt.Errorf("rename to %s: %w", rotatedPath, err)
		}
	}

	if err := w.openFile(); err != nil {
		return fmt.Errorf("open after rotate: %w", err)
	}

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.cleanup()
	}()

	return nil
}

// cleanup removes excess backups, deletes aged-out files, and optionally
// compresses surviving backups. It reads the filesystem directly — no mutex
// needed because it only touches rotated (non-current) files.
func (w *RotatingFileWriter) cleanup() {
	pattern := w.basePath + ".*" + w.ext
	if w.cfg.Compress {
		// Also match already-compressed backups.
		pattern = w.basePath + ".*" + w.ext + ".gz"
	}

	// Collect all rotated backup paths (uncompressed).
	uncompressedPattern := w.basePath + ".*" + w.ext
	matches, err := filepath.Glob(uncompressedPattern)
	if err != nil {
		return
	}
	// Also collect compressed backups.
	compressedPattern := w.basePath + ".*" + w.ext + ".gz"
	compressedMatches, _ := filepath.Glob(compressedPattern)

	// Exclude the current (non-rotated) log file from the uncompressed list.
	current := w.currentPath()
	var backups []string
	for _, m := range matches {
		if m != current {
			backups = append(backups, m)
		}
	}
	backups = append(backups, compressedMatches...)

	// Sort ascending by name (timestamp embedded in name gives chronological order).
	sort.Strings(backups)

	// Remove files that exceed MaxAge.
	cutoff := time.Now().Add(-w.cfg.MaxAge)
	var surviving []string
	for _, path := range backups {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(path)
		} else {
			surviving = append(surviving, path)
		}
	}

	// Enforce MaxBackups: remove oldest files over the limit.
	if w.cfg.MaxBackups > 0 && len(surviving) > w.cfg.MaxBackups {
		excess := surviving[:len(surviving)-w.cfg.MaxBackups]
		for _, path := range excess {
			_ = os.Remove(path)
		}
		surviving = surviving[len(surviving)-w.cfg.MaxBackups:]
	}

	// Optionally compress surviving uncompressed backups.
	if w.cfg.Compress {
		for _, path := range surviving {
			if strings.HasSuffix(path, ".gz") {
				continue
			}
			_ = compressFile(path)
		}
	}

	// Suppress unused variable warning when Compress is false.
	_ = pattern
}

// compressFile gzip-compresses src into src+".gz", then removes src.
func compressFile(src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	dst := src + ".gz"
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	if _, err := io.Copy(gz, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if err := gz.Close(); err != nil {
		_ = os.Remove(dst)
		return err
	}

	return os.Remove(src)
}
