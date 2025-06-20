package layout

import (
	"fmt"

	"github.com/gofrs/flock"
)

// fileLock manages file locking for any file operations.
// It provides thread-safe access to files by using OS-level file locks.
//
// Example usage:
//
//	lock := newFileLock(path, "config.json")
//	if err := lock.Lock(); err != nil {
//	    return err
//	}
//	defer lock.Unlock()
//	// ... perform file operations ...
type fileLock struct {
	path Path
	file string
	lock *flock.Flock
}

// newFileLock creates a new fileLock for the given path and file.
// The lock file will be created as {file}.lock in the same directory.
func newFileLock(path Path, file string) *fileLock {
	return &fileLock{
		path: path,
		file: file,
		lock: flock.New(path.path(file + ".lock")),
	}
}

// Lock acquires the lock for file operations.
// This is a blocking call that will wait until the lock is available.
func (l *fileLock) Lock() error {
	// Use blocking lock with a short timeout
	if err := l.lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire lock for %s: %w", l.file, err)
	}
	return nil
}

// TryLock attempts to acquire the lock without blocking.
// Returns an error immediately if the lock is already held.
func (l *fileLock) TryLock() error {
	locked, err := l.lock.TryLock()
	if err != nil {
		return fmt.Errorf("failed to acquire lock for %s: %w", l.file, err)
	}
	if !locked {
		return fmt.Errorf("lock for %s is already held", l.file)
	}
	return nil
}

// Unlock releases the lock.
// It's safe to call this even if the lock is not held.
func (l *fileLock) Unlock() error {
	if !l.lock.Locked() {
		return nil
	}
	if err := l.lock.Unlock(); err != nil {
		return fmt.Errorf("failed to release lock for %s: %w", l.file, err)
	}
	return nil
}
