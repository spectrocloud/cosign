package layout

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestIndexLock(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	t.Logf("Testing file locking in directory: %s", tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	t.Logf("Created initial index.json file")

	// Create two locks for index.json
	lock1 := newFileLock(path, "index.json")
	lock2 := newFileLock(path, "index.json")

	t.Logf("Created two file locks for index.json")

	// First lock should succeed (non-blocking)
	if err := lock1.TryLock(); err != nil {
		t.Fatalf("First lock failed: %v", err)
	}
	t.Logf("First lock acquired successfully")

	// Second lock should fail immediately
	if err := lock2.TryLock(); err == nil {
		t.Fatal("Second lock should have failed")
	}
	t.Logf("Second lock correctly failed (as expected)")

	// Unlock first lock
	if err := lock1.Unlock(); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	t.Logf("First lock released successfully")

	// Now second lock should succeed
	if err := lock2.TryLock(); err != nil {
		t.Fatalf("Second lock failed after unlock: %v", err)
	}
	t.Logf("Second lock acquired successfully after first lock was released")

	// Cleanup
	if err := lock2.Unlock(); err != nil {
		t.Fatalf("Final unlock failed: %v", err)
	}
	t.Logf("Test completed successfully - all locks working as expected")
}

func TestWriteFileWithLocking(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		wantLock bool
	}{
		{
			name:     "root index.json",
			filePath: "index.json",
			wantLock: true,
		},
		{
			name:     "nested index.json",
			filePath: "path/to/index.json",
			wantLock: true,
		},
		{
			name:     "blobs index.json",
			filePath: "blobs/sha256/index.json",
			wantLock: true,
		},
		{
			name:     "other file",
			filePath: "other.json",
			wantLock: true,
		},
		{
			name:     "nested other file",
			filePath: "path/to/other.json",
			wantLock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			path := Path(tmp)

			// Create the directory structure if needed
			dir := filepath.Dir(tt.filePath)
			if dir != "." {
				if err := os.MkdirAll(filepath.Join(tmp, dir), os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}

			// Try to write the file
			err := path.WriteFile(tt.filePath, []byte("test"), os.ModePerm)
			if err != nil {
				t.Fatalf("WriteFile failed: %v", err)
			}

			// Check if lock file was created
			lockPath := filepath.Join(tmp, tt.filePath+".lock")
			_, err = os.Stat(lockPath)
			lockExists := err == nil

			if lockExists != tt.wantLock {
				t.Errorf("Lock file existence = %v, want %v", lockExists, tt.wantLock)
			}

			// Verify the file was written
			content, err := os.ReadFile(filepath.Join(tmp, tt.filePath))
			if err != nil {
				t.Fatalf("Failed to read written file: %v", err)
			}
			if string(content) != "test" {
				t.Errorf("File content = %s, want test", string(content))
			}
		})
	}
}

func TestConcurrentWrites(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	numWriters := 10
	successCount := 0
	var mu sync.Mutex // Protect successCount

	// Launch multiple goroutines that will try to write to index.json simultaneously
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Try to write to index.json
			content := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, i)
			t.Logf("Writer %d attempting to write", i)
			err := path.WriteFile("index.json", []byte(content), os.ModePerm)
			if err != nil {
				t.Errorf("Writer %d got error: %v", i, err)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Expect all writes to succeed
	if successCount != numWriters {
		t.Errorf("Expected all writes to succeed, got %d successes", successCount)
	}

	// Verify the final content of index.json
	content, err := os.ReadFile(filepath.Join(tmp, "index.json"))
	if err != nil {
		t.Fatal(err)
	}

	// The content should be one of the possible values
	validContent := false
	for i := 0; i < numWriters; i++ {
		expected := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, i)
		if string(content) == expected {
			t.Logf("Writer %d wrote content: %s", i, expected)
			validContent = true
			break
		}
	}
	if !validContent {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestMultipleConcurrentWrites(t *testing.T) {
	tmp := t.TempDir()
	path1 := Path(tmp)
	path2 := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	numWriters := 10
	successCount := 0
	var mu sync.Mutex // Protect successCount

	// Launch multiple goroutines that will try to write to index.json simultaneously
	// using two different Path instances
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Try to write to index.json using first path
			content := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, i)
			err := path1.WriteFile("index.json", []byte(content), os.ModePerm)
			if err != nil {
				t.Errorf("Writer %d got error with path1: %v", i, err)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)

		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			// Try to write to index.json using second path
			content := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, i)
			err := path2.WriteFile("index.json", []byte(content), os.ModePerm)
			if err != nil {
				t.Errorf("Writer %d got error with path2: %v", i, err)
			} else {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Expect all writes to succeed
	if successCount != numWriters*2 {
		t.Errorf("Expected all writes to succeed, got %d successes", successCount)
	}

	// Verify the final content of index.json
	content, err := os.ReadFile(filepath.Join(tmp, "index.json"))
	if err != nil {
		t.Fatal(err)
	}

	// The content should be one of the possible values
	validContent := false
	for i := 0; i < numWriters; i++ {
		expected := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, i)
		if string(content) == expected {
			validContent = true
			break
		}
	}
	if !validContent {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestLockPanic(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// First goroutine that will panic while holding the lock
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic did not occur")
			}
		}()

		// This will panic after acquiring the lock
		err := path.WriteFile("index.json", []byte(`{"manifests":[{"id":1}]}`), os.ModePerm)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		panic("intentional panic while holding lock")
	}()

	// Wait a bit for the first goroutine to acquire the lock
	time.Sleep(100 * time.Millisecond)

	// Second goroutine that should still be able to acquire the lock
	// even after the first goroutine panicked
	err := path.WriteFile("index.json", []byte(`{"manifests":[{"id":2}]}`), os.ModePerm)
	if err != nil {
		t.Errorf("Failed to acquire lock after panic: %v", err)
	}

	// Wait for the first goroutine to finish
	<-done
}

func TestNestedLocks(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Test nested writes to the same file
	err := path.WriteFile("index.json", []byte(`{"manifests":[{"id":1}]}`), os.ModePerm)
	if err != nil {
		t.Fatalf("First write failed: %v", err)
	}

	// Try to write again while holding the lock
	err = path.WriteFile("index.json", []byte(`{"manifests":[{"id":2}]}`), os.ModePerm)
	if err != nil {
		t.Fatalf("Second write failed: %v", err)
	}

	// Verify the final content
	content, err := os.ReadFile(filepath.Join(tmp, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"manifests":[{"id":2}]}`
	if string(content) != expected {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestConcurrentReadsAndWrites(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	numWriters := 5
	numReaders := 10
	successCount := 0
	var mu sync.Mutex

	// Launch readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			content, err := os.ReadFile(filepath.Join(tmp, "index.json"))
			if err != nil {
				t.Errorf("Reader %d failed to read: %v", readerID, err)
				return
			}
			t.Logf("Reader %d read content: %s", readerID, string(content))
		}(i)
	}

	// Launch writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			content := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, writerID)
			err := path.WriteFile("index.json", []byte(content), os.ModePerm)
			if err != nil {
				t.Errorf("Writer %d failed to write: %v", writerID, err)
				return
			}
			mu.Lock()
			successCount++
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// All writes should succeed
	if successCount != numWriters {
		t.Errorf("Expected %d successful writes, got %d", numWriters, successCount)
	}
}

func TestLockErrors(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Test writing to a non-existent directory
	err := path.WriteFile("nonexistent/index.json", []byte(`{"manifests":[]}`), os.ModePerm)
	if err == nil {
		t.Error("Expected error when writing to non-existent directory")
	}

	// Test writing to a read-only directory
	readOnlyDir := filepath.Join(tmp, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0444); err != nil {
		t.Fatal(err)
	}
	readOnlyPath := Path(readOnlyDir)
	err = readOnlyPath.WriteFile("index.json", []byte(`{"manifests":[]}`), os.ModePerm)
	if err == nil {
		t.Error("Expected error when writing to read-only directory")
	}
}

func TestLockTimeout(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// First goroutine that will hold the lock for a while
	done := make(chan struct{})
	go func() {
		defer close(done)
		// Write and hold the lock
		err := path.WriteFile("index.json", []byte(`{"manifests":[{"id":1}]}`), os.ModePerm)
		if err != nil {
			t.Errorf("First write failed: %v", err)
		}
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
	}()

	// Wait a bit for the first goroutine to acquire the lock
	time.Sleep(10 * time.Millisecond)

	// Second goroutine that should wait for the lock
	err := path.WriteFile("index.json", []byte(`{"manifests":[{"id":2}]}`), os.ModePerm)
	if err != nil {
		t.Errorf("Second write failed: %v", err)
	}

	// Wait for the first goroutine to finish
	<-done

	// Verify the final content
	content, err := os.ReadFile(filepath.Join(tmp, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"manifests":[{"id":2}]}`
	if string(content) != expected {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestLockCleanup(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial index.json
	if err := os.WriteFile(filepath.Join(tmp, "index.json"), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	// Write to the file (this acquires and releases the lock)
	err := path.WriteFile("index.json", []byte(`{"manifests":[{"id":1}]}`), os.ModePerm)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Now try to acquire the lock again (should succeed)
	lock := newFileLock(path, "index.json")
	if err := lock.TryLock(); err != nil {
		t.Fatalf("Failed to reacquire lock after write: %v", err)
	}
	if err := lock.Unlock(); err != nil {
		t.Fatalf("Failed to unlock after reacquire: %v", err)
	}
}

func TestConcurrentDifferentFiles(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial files
	files := []string{"index.json", "other.json", "nested/index.json"}
	for _, f := range files {
		dir := filepath.Dir(f)
		if dir != "." {
			if err := os.MkdirAll(filepath.Join(tmp, dir), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(tmp, f), []byte(`{"manifests":[]}`), os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	numWriters := 5

	// Launch writers for each file
	for _, file := range files {
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(file string, writerID int) {
				defer wg.Done()
				content := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, writerID)
				err := path.WriteFile(file, []byte(content), os.ModePerm)
				if err != nil {
					t.Errorf("Writer %d failed to write to %s: %v", writerID, file, err)
				}
			}(file, i)
		}
	}

	wg.Wait()

	// Verify all files were written
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(tmp, file))
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
			continue
		}
		// Content should be one of the possible values
		validContent := false
		for i := 0; i < numWriters; i++ {
			expected := fmt.Sprintf(`{"manifests":[{"id":%d}]}`, i)
			if string(content) == expected {
				validContent = true
				break
			}
		}
		if !validContent {
			t.Errorf("Unexpected content in %s: %s", file, string(content))
		}
	}
}

func TestConcurrentDifferentFileTypes(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Create initial files
	files := []string{
		"index.json",
		"config.json",
		"manifest.json",
		"data.txt",
		"nested/config.yaml",
	}

	for _, f := range files {
		dir := filepath.Dir(f)
		if dir != "." {
			if err := os.MkdirAll(filepath.Join(tmp, dir), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(tmp, f), []byte(`{"initial":true}`), os.ModePerm); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	numWriters := 3

	// Launch writers for each file type
	for _, file := range files {
		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(file string, writerID int) {
				defer wg.Done()
				content := fmt.Sprintf(`{"writer":%d,"file":"%s"}`, writerID, file)
				err := path.WriteFile(file, []byte(content), os.ModePerm)
				if err != nil {
					t.Errorf("Writer %d failed to write to %s: %v", writerID, file, err)
				}
			}(file, i)
		}
	}

	wg.Wait()

	// Verify all files were written
	for _, file := range files {
		content, err := os.ReadFile(filepath.Join(tmp, file))
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
			continue
		}
		// Content should be one of the possible values
		validContent := false
		for i := 0; i < numWriters; i++ {
			expected := fmt.Sprintf(`{"writer":%d,"file":"%s"}`, i, file)
			if string(content) == expected {
				validContent = true
				break
			}
		}
		if !validContent {
			t.Errorf("Unexpected content in %s: %s", file, string(content))
		}

		// Verify lock file exists
		lockPath := filepath.Join(tmp, file+".lock")
		if _, err := os.Stat(lockPath); err != nil {
			t.Errorf("Lock file not found for %s: %v", file, err)
		}
	}
}

func TestGenericFileLock(t *testing.T) {
	tmp := t.TempDir()
	path := Path(tmp)

	// Test files to lock
	testFiles := []string{
		"config.json",
		"manifest.json",
		"data.txt",
		"nested/file.json",
		"deeply/nested/config.yaml",
	}

	for _, fileName := range testFiles {
		t.Run(fileName, func(t *testing.T) {
			// Create directory if needed
			dir := filepath.Dir(fileName)
			if dir != "." {
				if err := os.MkdirAll(filepath.Join(tmp, dir), os.ModePerm); err != nil {
					t.Fatal(err)
				}
			}

			// Create two locks for the same file
			lock1 := newFileLock(path, fileName)
			lock2 := newFileLock(path, fileName)

			// First lock should succeed
			if err := lock1.TryLock(); err != nil {
				t.Fatalf("First lock failed for %s: %v", fileName, err)
			}

			// Second lock should fail immediately
			if err := lock2.TryLock(); err == nil {
				t.Fatalf("Second lock should have failed for %s", fileName)
			}

			// Unlock first lock
			if err := lock1.Unlock(); err != nil {
				t.Fatalf("Unlock failed for %s: %v", fileName, err)
			}

			// Now second lock should succeed
			if err := lock2.TryLock(); err != nil {
				t.Fatalf("Second lock failed after unlock for %s: %v", fileName, err)
			}

			// Cleanup
			if err := lock2.Unlock(); err != nil {
				t.Fatalf("Final unlock failed for %s: %v", fileName, err)
			}

			// Verify lock file was created
			lockPath := filepath.Join(tmp, fileName+".lock")
			if _, err := os.Stat(lockPath); err != nil {
				t.Errorf("Lock file not found for %s: %v", fileName, err)
			}
		})
	}
}
