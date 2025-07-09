package logging

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileWriter(t *testing.T) {
	// Create temp directory
	tmpDir, err := ioutil.TempDir("", "katago-mcp-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	// Create file writer
	fw, err := NewFileWriter(logPath, 1, 3, 30, false)
	if err != nil {
		t.Fatal(err)
	}
	defer fw.Close()

	// Write some data
	testData := []byte("Test log message\n")
	n, err := fw.Write(testData)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Verify file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}

	// Read file contents
	content, err := ioutil.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(testData) {
		t.Errorf("Expected content %q, got %q", testData, content)
	}
}

func TestFileWriterRotation(t *testing.T) {
	// Create temp directory
	tmpDir, err := ioutil.TempDir("", "katago-mcp-test-rotation")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	// Create file writer with small max size (1KB)
	fw, err := NewFileWriter(logPath, 0, 3, 30, false) // 0 MB = 1KB for testing
	if err != nil {
		t.Fatal(err)
	}
	fw.maxSize = 1024 // Override to 1KB for testing
	defer fw.Close()

	// Write enough data to trigger rotation
	largeData := make([]byte, 600)
	for i := range largeData {
		largeData[i] = 'A'
	}

	// First write
	_, err = fw.Write(largeData)
	if err != nil {
		t.Fatal(err)
	}

	// Second write should trigger rotation
	_, err = fw.Write(largeData)
	if err != nil {
		t.Fatal(err)
	}

	// Give rotation a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Check for backup file
	files, err := filepath.Glob(filepath.Join(tmpDir, "test.log.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 backup file, found %d", len(files))
	}
}

func TestMultiWriter(t *testing.T) {
	// Create temp file
	tmpFile, err := ioutil.TempFile("", "katago-mcp-multiwriter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Create multi-writer
	mw := NewMultiWriter(os.Stderr, tmpFile)

	// Write data
	testData := []byte("Multi-writer test\n")
	n, err := mw.Write(testData)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Verify file contents
	tmpFile.Seek(0, 0)
	content, err := ioutil.ReadAll(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != string(testData) {
		t.Errorf("Expected content %q, got %q", testData, content)
	}
}
