// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sharedfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// set returns a change function that sets the given key to the given value
// in a map, making it if it is nil.
func set(k string, v int) func(d *map[string]int) {
	return func(d *map[string]int) {
		if *d == nil {
			*d = map[string]int{}
		}
		(*d)[k] = v
	}
}

// TestReload tests that the data written by another process (another File)
// are read again only when the file changes.
func TestReload(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "sub", "x.json")
	var a, b File[map[string]int]
	a.Path, b.Path = fn, fn
	require.NoError(t, a.Reload())
	assert.Nil(t, a.Data, "no file yet")
	assert.False(t, a.Changed())

	// the other process writes it, and it is read again here
	require.NoError(t, b.Update(set("b", 1)))
	assert.False(t, b.Changed(), "written by itself")
	assert.True(t, a.Changed())
	require.NoError(t, a.Reload())
	assert.Equal(t, map[string]int{"b": 1}, a.Data)
	assert.False(t, a.Changed())

	// the same size and modification time, as a different file
	require.NoError(t, b.Update(set("b", 2)))
	info, err := os.Stat(fn)
	require.NoError(t, err)
	require.NoError(t, os.Chtimes(fn, info.ModTime(), a.info.ModTime()))
	assert.True(t, a.Changed(), "renamed into place")
	require.NoError(t, a.Reload())
	assert.Equal(t, map[string]int{"b": 2}, a.Data)

	// an update here keeps the changes of the other process
	require.NoError(t, b.Update(set("c", 3)))
	require.NoError(t, a.Update(set("a", 4)))
	assert.Equal(t, map[string]int{"a": 4, "b": 2, "c": 3}, a.Data)
	require.NoError(t, b.Reload())
	assert.Equal(t, a.Data, b.Data)
	_, err = os.Stat(fn + ".lck")
	assert.True(t, os.IsNotExist(err), "unlocked")
	_, err = os.Stat(fn + ".tmp")
	assert.True(t, os.IsNotExist(err), "renamed")

	// a deleted file keeps the data, and a new path resets them
	require.NoError(t, os.Remove(fn))
	require.NoError(t, a.Reload())
	assert.Len(t, a.Data, 3)
	a.Path = filepath.Join(t.TempDir(), "y.json")
	assert.True(t, a.Changed())
	require.NoError(t, a.Reload())
	assert.Nil(t, a.Data)
}

// TestMemory tests that the data are only kept in memory with no path.
func TestMemory(t *testing.T) {
	var f File[map[string]int]
	assert.False(t, f.Changed())
	require.NoError(t, f.Reload())
	require.NoError(t, f.Update(set("a", 1)))
	assert.Equal(t, map[string]int{"a": 1}, f.Data)
}

// TestInvalid tests that a file that is not valid JSON gives an error when
// it is read, and is replaced when it is updated.
func TestInvalid(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "x.json")
	require.NoError(t, os.WriteFile(fn, []byte("{not json"), 0o644))
	f := File[map[string]int]{Path: fn}
	assert.Error(t, f.Reload())
	assert.False(t, f.Changed(), "not read again until it changes")
	require.NoError(t, f.Update(set("a", 1)))
	var g File[map[string]int]
	g.Path = fn
	require.NoError(t, g.Reload())
	assert.Equal(t, map[string]int{"a": 1}, g.Data)
}

// TestConcurrent tests that many processes (goroutines with their own
// File) updating the file at once each keep the changes of the others.
func TestConcurrent(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "x.json")
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			f := File[map[string]int]{Path: fn}
			for j := range 10 {
				assert.NoError(t, f.Update(set(fmt.Sprintf("p%d-%d", i, j), j)))
			}
		})
	}
	wg.Wait()
	f := File[map[string]int]{Path: fn}
	require.NoError(t, f.Reload())
	assert.Len(t, f.Data, 8*10)
}

// TestLockStale tests that a lock file left by a process that ended while
// holding it is taken over, and that a lock held by another process is
// waited for.
func TestLockStale(t *testing.T) {
	fn := filepath.Join(t.TempDir(), "x.lck")
	require.NoError(t, os.WriteFile(fn, nil, 0o644))
	old := time.Now().Add(-time.Minute)
	require.NoError(t, os.Chtimes(fn, old, old))
	unlock, err := Lock(fn)
	require.NoError(t, err)

	// another process waits for it to be unlocked
	locked := make(chan time.Time)
	go func() {
		unlock, err := Lock(fn)
		assert.NoError(t, err)
		locked <- time.Now()
		unlock()
	}()
	time.Sleep(100 * time.Millisecond)
	unlocked := time.Now()
	unlock()
	assert.True(t, (<-locked).After(unlocked), "waited")
	time.Sleep(10 * time.Millisecond)
	_, err = os.Stat(fn)
	assert.True(t, os.IsNotExist(err))
}

// TestLockTimeout tests that a lock that is held, and not stale, gives an
// error after a while.
func TestLockTimeout(t *testing.T) {
	defer func(s time.Duration) { LockStale, lockTimeout = s, 0 }(LockStale)
	LockStale, lockTimeout = time.Hour, 100*time.Millisecond
	fn := filepath.Join(t.TempDir(), "x.lck")
	unlock, err := Lock(fn)
	require.NoError(t, err)
	defer unlock()
	_, err = Lock(fn)
	assert.Error(t, err)
}
