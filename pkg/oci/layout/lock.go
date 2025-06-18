// Copyright 2024 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
