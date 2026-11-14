// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sharedfile provides a JSON file of data that is shared by several
// processes, such as different Cogent Core apps, like the sort orders of
// directories and the geometry of windows. The data is read again when
// another process has changed the file, and the file is written with a
// lock file held, after reading the changes of the other processes, so
// that they are not lost, and it is written whole and then renamed, so
// that the other processes never read a partial file.
package sharedfile

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// File is a JSON file of data of type T that is shared by several
// processes: see [File.Reload] and [File.Update]. It is not safe for
// concurrent use by goroutines: the caller protects it with a mutex.
// The zero value is ready to use, with only [File.Path] to set.
type File[T any] struct {

	// Path is the path of the file. If it is empty, the data are only
	// kept in memory. It can be changed, as when the directory of the
	// file changes in tests: the data are then those of the new file.
	Path string

	// Data is the data of the file, as last read from it or updated.
	// It is replaced by a new value when the file is read again, so it
	// must only be changed by [File.Update].
	Data T

	// file is the path of the file that was last read or written.
	file string

	// info is the state of the file when it was last read or written,
	// to read it again when another process changes it.
	info fs.FileInfo
}

// Changed returns whether the file exists and has changed since it was
// last read or written by this File, as another process has written it,
// or whether [File.Path] has changed. Another process is detected by the
// file being a different file (as it is renamed into place when it is
// written), or by a different modification time or size.
func (f *File[T]) Changed() bool {
	if f.Path == "" {
		return false
	}
	info, err := os.Stat(f.Path)
	if err != nil {
		return f.file != f.Path
	}
	return f.changed(info)
}

// changed returns whether the given state of the file differs from the
// recorded one.
func (f *File[T]) changed(info fs.FileInfo) bool {
	if f.file != f.Path || f.info == nil {
		return true
	}
	return !os.SameFile(info, f.info) || !info.ModTime().Equal(f.info.ModTime()) || info.Size() != f.info.Size()
}

// Reload reads the file into [File.Data] if it has [File.Changed], which
// replaces Data by the new value that is read. If the file does not exist,
// Data is kept, except when [File.Path] has changed, when it is reset to
// the zero value. If the file is not valid JSON, Data is what could be
// read from it, and an error is returned; the file is then replaced when
// it is next updated.
func (f *File[T]) Reload() error {
	_, err := f.reload()
	return err
}

// reload is [File.Reload], which also returns whether the error is from
// reading the file (as opposed to decoding it), as readErr.
func (f *File[T]) reload() (readErr bool, err error) {
	if f.Path == "" {
		return false, nil
	}
	// the file is opened before its state is read, so that the state is
	// that of the data read even when another process replaces it
	fh, err := os.Open(f.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			if f.file != f.Path {
				var zero T
				f.Data, f.file, f.info = zero, f.Path, nil
			}
			return false, nil
		}
		return true, err
	}
	defer fh.Close()
	info, err := fh.Stat()
	if err != nil {
		return true, err
	}
	if !f.changed(info) {
		return false, nil
	}
	b, err := io.ReadAll(fh)
	if err != nil {
		return true, err
	}

	// the data are decoded into a new value, so that nothing is kept from
	// before, and the state is recorded even when it is not valid, so it
	// is not read again until it changes
	var d T
	err = json.Unmarshal(b, &d)
	f.Data, f.file, f.info = d, f.Path, info
	return false, err
}

// Update changes [File.Data] with the given change function (which can be
// nil to just write it) and writes it to the file, with the file locked
// (see [Lock]), after reading the changes of other processes with
// [File.Reload], which the change function must then merge with. A file
// that is not valid JSON is replaced. If the file cannot be locked or
// read, the data are only changed in memory, and the error is returned.
// The directory of the file is made if it does not exist.
func (f *File[T]) Update(change func(d *T)) error {
	if change == nil {
		change = func(d *T) {}
	}
	if f.Path == "" {
		change(&f.Data)
		return nil
	}

	// the changes of other processes are read with the file locked, so
	// that no other process writes it until this one has
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		change(&f.Data)
		return err
	}
	unlock, err := Lock(f.Path + ".lck")
	if err != nil {
		change(&f.Data)
		return err
	}
	defer unlock()
	if readErr, err := f.reload(); readErr {
		change(&f.Data)
		return err
	}
	change(&f.Data)

	// the file is written whole, then renamed, so that other processes
	// never read a partial file
	b, err := json.MarshalIndent(f.Data, "", "\t")
	if err != nil {
		return err
	}
	tmp := f.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, f.Path); err != nil {
		os.Remove(tmp)
		return err
	}
	if info, err := os.Stat(f.Path); err == nil {
		f.file, f.info = f.Path, info
	}
	return nil
}

// LockStale is how long a lock file can exist before it is taken as left
// by a process that ended while holding it (see [Lock]).
var LockStale = 2 * time.Second

// lockWait is how long [Lock] waits between tries.
const lockWait = 20 * time.Millisecond

// lockTimeout is how long [Lock] tries for, if not 0 (for tests), instead
// of [LockStale] plus a second.
var lockTimeout time.Duration

// Lock locks the given lock file, by creating it (which only one process
// can do), waiting for another process to remove it, or taking it over
// when it is older than [LockStale]. It gives up with an error after
// waiting for longer than LockStale (plus a second). It returns the
// function to unlock it, by removing it.
func Lock(fn string) (unlock func(), err error) {
	timeout := LockStale + time.Second
	if lockTimeout > 0 {
		timeout = lockTimeout
	}
	deadline := time.Now().Add(timeout)
	for {
		// the time of locking is written for anyone looking at it, but the
		// modification time of the file is what is used
		fh, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			fh.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
			fh.Close()
			return func() { os.Remove(fn) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}

		// a stale lock file is removed, and another try made right away
		if info, err := os.Stat(fn); err == nil && time.Since(info.ModTime()) > LockStale {
			os.Remove(fn)
			continue
		}
		if time.Now().After(deadline) {
			return nil, errors.New("sharedfile: could not lock " + fn)
		}
		time.Sleep(lockWait)
	}
}
