// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Location is a file path on a host, for [Copy]: on the host of the given
// connected Client, or on the local host if the Client is nil. A relative
// path on a host is relative to the current Dir of its Client.
type Location struct {
	Client *Client
	Path   string
}

func (l Location) String() string {
	if l.Client == nil {
		return l.Path
	}
	return l.Client.Host + ":" + l.Path
}

// Copy copies the files of the given source locations to the given
// destination location, as cp -r does, using the SFTP protocol on the
// connections of the remote hosts. A source can have the wildcards of
// [path.Match], such as *.tsv, and a directory is copied with all of its
// contents. The sources are copied into the destination if it is a
// directory: if it exists, ends with a /, is . or .., or if there are
// several sources (a directory that does not exist is made); otherwise,
// the destination is the new name of the single source file or directory.
// The modes of the files are kept. The copy stops when the context is
// done; use context.Background() for no limit.
func Copy(ctx context.Context, dst Location, srcs ...Location) error {
	return CopyProgress(ctx, nil, dst, srcs...)
}

// Progress is the progress of a [CopyProgress].
type Progress struct {

	// File is the path of the source file being copied.
	File string

	// Done and Size are the bytes of the file copied so far, and its size.
	Done, Size int64

	// TotalDone and Total are the bytes of all of the files copied so far,
	// and their total size.
	TotalDone, Total int64

	// FileNum is the 1-based number of the file being copied, of NFiles.
	FileNum, NFiles int

	// Finished is whether all of the files are copied.
	Finished bool
}

// CopyProgress copies files, as [Copy], calling the given function, if
// non-nil, with the progress of the copy: when a file starts, as its
// bytes are copied, and at the end, when Finished is true. It is called
// often, so a function that shows the progress should show it at
// intervals. It is not called concurrently, but it can be called on
// another goroutine than that of CopyProgress.
func CopyProgress(ctx context.Context, progress func(p Progress), dst Location, srcs ...Location) error {
	dfs, err := locationFS(dst)
	if err != nil {
		return err
	}
	type source struct {
		fs   fileSystem
		path string
	}
	var files []source
	for _, src := range srcs {
		sfs, err := locationFS(src)
		if err != nil {
			return err
		}
		p := sfs.abs(src.Path)
		if !hasMeta(p) {
			files = append(files, source{sfs, p})
			continue
		}
		matches, err := sfs.Glob(p)
		if err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
		if len(matches) == 0 {
			return fmt.Errorf("%s: no files match", src)
		}
		for _, m := range matches {
			files = append(files, source{sfs, m})
		}
	}
	if len(files) == 0 {
		return errors.New("copy: no files to copy")
	}

	cp := &copier{ctx: ctx, progress: progress}
	if progress != nil { // the total size, for the progress
		for _, f := range files {
			if err := cp.measure(f.fs, f.path); err != nil {
				return err
			}
		}
	}

	dpath := dfs.abs(dst.Path)
	intoDir := len(files) > 1 || strings.HasSuffix(dst.Path, "/") || dst.Path == "." || dst.Path == ".." || dst.Path == ""
	if info, err := dfs.Stat(dpath); err == nil && info.IsDir() {
		intoDir = true
	} else if intoDir {
		if err := dfs.MkdirAll(dpath, 0777); err != nil {
			return fmt.Errorf("%s: %w", dst, err)
		}
	}
	for _, f := range files {
		to := dpath
		if intoDir {
			to = dfs.join(dpath, f.fs.base(f.path))
		}
		if err := cp.copyPath(dfs, to, f.fs, f.path); err != nil {
			return err
		}
	}
	cp.report(true)
	return nil
}

// copier copies files for [CopyProgress].
type copier struct {
	ctx      context.Context
	progress func(p Progress)
	p        Progress
}

// report calls the progress function, if any, with the progress.
func (cp *copier) report(finished bool) {
	if cp.progress != nil {
		cp.p.Finished = finished
		cp.progress(cp.p)
	}
}

// add adds the given number of bytes copied to the progress.
func (cp *copier) add(n int64) {
	if n > 0 {
		cp.p.Done += n
		cp.p.TotalDone += n
		cp.report(false)
	}
}

// measure adds the size and number of the files of the given path of the
// given file system to the total of the progress.
func (cp *copier) measure(sfs fileSystem, src string) error {
	info, err := sfs.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		cp.p.Total += info.Size()
		cp.p.NFiles++
		return nil
	}
	entries, err := sfs.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := cp.measure(sfs, sfs.join(src, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// copyPath copies the file or directory at the given path of the source
// file system to the given path of the destination file system.
func (cp *copier) copyPath(dfs fileSystem, dst string, sfs fileSystem, src string) error {
	if err := cp.ctx.Err(); err != nil {
		return err
	}
	info, err := sfs.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return cp.copyFile(dfs, dst, sfs, src, info)
	}
	if err := dfs.MkdirAll(dst, info.Mode().Perm()|0700); err != nil {
		return err
	}
	entries, err := sfs.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := cp.copyPath(dfs, dfs.join(dst, e.Name()), sfs, sfs.join(src, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// copyFile copies the file at the given path of the source file system,
// with the given info, to the given path of the destination file system,
// with the same mode.
func (cp *copier) copyFile(dfs fileSystem, dst string, sfs fileSystem, src string, info fs.FileInfo) error {
	ctx := cp.ctx
	r, err := sfs.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()
	w, err := dfs.Create(dst)
	if err != nil {
		return err
	}
	// Closing the files stops the copy when the context is done.
	stop := context.AfterFunc(ctx, func() {
		r.Close()
		w.Close()
	})
	defer stop()
	cp.p.File, cp.p.Done, cp.p.Size = src, 0, info.Size()
	cp.p.FileNum++
	cp.report(false)
	if rf, ok := sftpReaderFrom(w); ok {
		// The reader has the size, for SFTP to write concurrently.
		_, err = rf.ReadFrom(&countReader{r: r, size: info.Size(), add: cp.add})
	} else {
		// A remote source reads concurrently, to the writer (its WriteTo).
		_, err = io.Copy(&countWriter{w: w, add: cp.add}, r)
	}
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil {
		return fmt.Errorf("copying %s: %w", src, err)
	}
	return dfs.Chmod(dst, info.Mode().Perm())
}

// countReader is a reader that counts the bytes that it reads, for the
// progress of a copy. It has the size of its data, for the concurrent
// writes of [sftp.File.ReadFrom].
type countReader struct {
	r    io.Reader
	size int64
	add  func(n int64)
}

func (c *countReader) Read(b []byte) (int, error) {
	n, err := c.r.Read(b)
	c.add(int64(n))
	return n, err
}

func (c *countReader) Size() int64 { return c.size }

// countWriter is a writer that counts the bytes that it writes, for the
// progress of a copy.
type countWriter struct {
	w   io.Writer
	add func(n int64)
}

func (c *countWriter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.add(int64(n))
	return n, err
}

// hasMeta returns whether the given path has any of the wildcards of
// path.Match.
func hasMeta(p string) bool {
	return strings.ContainsAny(p, `*?[\`)
}

// fileSystem is a file system of a host for [Copy]: that of the local host,
// or of a remote host through SFTP.
type fileSystem interface {
	Stat(name string) (fs.FileInfo, error)
	Open(name string) (io.ReadCloser, error)
	Create(name string) (io.WriteCloser, error)
	MkdirAll(name string, perm fs.FileMode) error
	ReadDir(name string) ([]fs.FileInfo, error)
	Glob(pattern string) ([]string, error)
	Chmod(name string, mode fs.FileMode) error

	// abs returns the given path as an absolute path, relative to the
	// current directory of the host if relative.
	abs(p string) string

	join(elem ...string) string
	base(p string) string
}

// localFS is the file system of the local host.
type localFS struct{}

func (localFS) Stat(name string) (fs.FileInfo, error)        { return os.Stat(name) }
func (localFS) Open(name string) (io.ReadCloser, error)      { return os.Open(name) }
func (localFS) Create(name string) (io.WriteCloser, error)   { return os.Create(name) }
func (localFS) MkdirAll(name string, perm fs.FileMode) error { return os.MkdirAll(name, perm) }
func (localFS) Glob(pattern string) ([]string, error)        { return filepath.Glob(pattern) }
func (localFS) Chmod(name string, mode fs.FileMode) error    { return os.Chmod(name, mode) }
func (localFS) join(elem ...string) string                   { return filepath.Join(elem...) }
func (localFS) base(p string) string                         { return filepath.Base(p) }

func (localFS) ReadDir(name string) ([]fs.FileInfo, error) {
	entries, err := os.ReadDir(name)
	if err != nil {
		return nil, err
	}
	infos := make([]fs.FileInfo, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (localFS) abs(p string) string {
	if p == "" {
		p = "."
	}
	if a, err := filepath.Abs(p); err == nil {
		return a
	}
	return p
}

// CopyLocalFileToHost copies the given local file or directory to the
// given path on the remote host (see [Copy]). A relative host path is
// relative to the current Dir of the Client.
func (cl *Client) CopyLocalFileToHost(ctx context.Context, localFilename, hostFilename string) error {
	return Copy(ctx, Location{Client: cl, Path: hostFilename}, Location{Path: localFilename})
}

// CopyHostToLocalFile copies the given file or directory on the remote
// host to the given local path (see [Copy]). A relative host path is
// relative to the current Dir of the Client.
func (cl *Client) CopyHostToLocalFile(ctx context.Context, hostFilename, localFilename string) error {
	return Copy(ctx, Location{Path: localFilename}, Location{Client: cl, Path: hostFilename})
}
