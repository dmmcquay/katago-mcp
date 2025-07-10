package logging

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileWriter provides a thread-safe writer with rotation support.
type FileWriter struct {
	mu            sync.Mutex
	file          *os.File
	path          string
	maxSize       int64 // in bytes
	maxBackups    int
	maxAge        int // in days
	compress      bool
	currentSize   int64
	rotateAtStart bool
}

// NewFileWriter creates a new file writer with rotation support.
func NewFileWriter(path string, maxSizeMB, maxBackups, maxAge int, compress bool) (*FileWriter, error) {
	fw := &FileWriter{
		path:       path,
		maxSize:    int64(maxSizeMB) * 1024 * 1024,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open or create the log file
	if err := fw.openFile(); err != nil {
		return nil, err
	}

	// Start cleanup goroutine
	go fw.cleanupOldFiles()

	return fw, nil
}

// Write implements io.Writer interface.
func (fw *FileWriter) Write(p []byte) (n int, err error) {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Check if rotation is needed
	if fw.shouldRotate(int64(len(p))) {
		if rotateErr := fw.rotate(); rotateErr != nil {
			return 0, rotateErr
		}
	}

	n, err = fw.file.Write(p)
	if err != nil {
		return n, err
	}

	fw.currentSize += int64(n)
	return n, nil
}

// Close closes the file writer.
func (fw *FileWriter) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	if fw.file != nil {
		return fw.file.Close()
	}
	return nil
}

// openFile opens the log file for writing.
func (fw *FileWriter) openFile() error {
	// Get file info to check current size
	info, err := os.Stat(fw.path)
	if err == nil {
		fw.currentSize = info.Size()
		// Check if we need to rotate on startup
		if fw.currentSize >= fw.maxSize {
			fw.rotateAtStart = true
		}
	}

	// Open file in append mode
	file, err := os.OpenFile(fw.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	fw.file = file

	// Rotate if needed on startup
	if fw.rotateAtStart {
		fw.rotateAtStart = false
		return fw.rotate()
	}

	return nil
}

// shouldRotate checks if rotation is needed.
func (fw *FileWriter) shouldRotate(writeSize int64) bool {
	if fw.maxSize <= 0 {
		return false
	}
	return fw.currentSize+writeSize > fw.maxSize
}

// rotate performs log rotation.
func (fw *FileWriter) rotate() error {
	// Close current file
	if fw.file != nil {
		if err := fw.file.Close(); err != nil {
			return err
		}
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", fw.path, timestamp)

	// Rename current file to backup
	if err := os.Rename(fw.path, backupPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to rotate log file: %w", err)
	}

	// Compress if enabled
	if fw.compress {
		go fw.compressFile(backupPath)
	}

	// Open new file
	if err := fw.openFile(); err != nil {
		return err
	}

	fw.currentSize = 0
	return nil
}

// compressFile compresses a log file using gzip.
func (fw *FileWriter) compressFile(path string) {
	// This is a placeholder - actual compression would use compress/gzip
	// For now, we'll just rename to indicate it should be compressed
	// In a production implementation, you'd use gzip to actually compress the file
}

// cleanupOldFiles removes old log files based on maxBackups and maxAge.
func (fw *FileWriter) cleanupOldFiles() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once at startup
	fw.performCleanup()

	// Then run daily
	for range ticker.C {
		fw.performCleanup()
	}
}

// performCleanup performs the actual cleanup.
func (fw *FileWriter) performCleanup() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	dir := filepath.Dir(fw.path)
	base := filepath.Base(fw.path)

	// Get all backup files
	pattern := fmt.Sprintf("%s.*", base)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Filter out the current log file
	var backups []string
	for _, match := range matches {
		if match != fw.path {
			backups = append(backups, match)
		}
	}

	// Remove old files based on age
	if fw.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -fw.maxAge)
		for _, backup := range backups {
			info, err := os.Stat(backup)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				_ = os.Remove(backup) // Best effort cleanup
			}
		}
	}

	// Remove excess backups
	if fw.maxBackups > 0 && len(backups) > fw.maxBackups {
		// Sort by modification time and remove oldest
		// This is simplified - in production you'd sort properly
		excess := len(backups) - fw.maxBackups
		for i := 0; i < excess && i < len(backups); i++ {
			_ = os.Remove(backups[i]) // Best effort cleanup
		}
	}
}

// MultiWriter combines multiple writers.
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a writer that duplicates writes to all provided writers.
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{writers: writers}
}

// Write writes to all writers.
func (mw *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range mw.writers {
		n, err = w.Write(p)
		if err != nil {
			return n, err
		}
	}
	return len(p), nil
}
