// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js

package sshclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/pkg/sftp"
)

// sftpConn is the SFTP client of a connection (see [Client.SFTP]).
type sftpConn = *sftp.Client

// locationFS returns the file system of the host of the given location.
func locationFS(l Location) (fileSystem, error) {
	if l.Client == nil {
		return localFS{}, nil
	}
	sc, err := l.Client.SFTP()
	if err != nil {
		return nil, err
	}
	return &remoteFS{sc: sc, host: l.Client.Host, dir: l.Client.Dir}, nil
}

// sftpReaderFrom returns the given writer as an [io.ReaderFrom] if it is a
// file on a remote host, which writes concurrently from a reader that has
// a Size method.
func sftpReaderFrom(w io.Writer) (io.ReaderFrom, bool) {
	sf, ok := w.(*sftp.File)
	return sf, ok
}

// remoteFS is the file system of a remote host, through SFTP.
type remoteFS struct {
	sc   *sftp.Client
	host string
	dir  string // the current directory
}

// The errors of SFTP do not have the path, which these add, with the host,
// as an [fs.PathError].

func (r *remoteFS) Stat(name string) (fs.FileInfo, error) {
	info, err := r.sc.Stat(name)
	return info, r.pathError("stat", name, err)
}

func (r *remoteFS) Open(name string) (io.ReadCloser, error) {
	f, err := r.sc.Open(name)
	if err != nil {
		return nil, r.pathError("open", name, err)
	}
	return f, nil
}

func (r *remoteFS) Create(name string) (io.WriteCloser, error) {
	f, err := r.sc.Create(name)
	if err != nil {
		return nil, r.pathError("create", name, err)
	}
	return f, nil
}

func (r *remoteFS) MkdirAll(name string, perm fs.FileMode) error {
	return r.pathError("mkdir", name, r.sc.MkdirAll(name))
}

func (r *remoteFS) ReadDir(name string) ([]fs.FileInfo, error) {
	infos, err := r.sc.ReadDir(name)
	return infos, r.pathError("readdir", name, err)
}

func (r *remoteFS) Glob(pattern string) ([]string, error) {
	m, err := r.sc.Glob(pattern)
	return m, r.pathError("glob", pattern, err)
}

func (r *remoteFS) Chmod(name string, mode fs.FileMode) error {
	return r.pathError("chmod", name, r.sc.Chmod(name, mode))
}

// pathError returns the given error of the given operation on the given
// path as an [fs.PathError] with the host in the path, or nil if nil.
func (r *remoteFS) pathError(op, name string, err error) error {
	if err == nil {
		return nil
	}
	return &fs.PathError{Op: op, Path: r.host + ":" + name, Err: err}
}
func (r *remoteFS) join(elem ...string) string { return path.Join(elem...) }
func (r *remoteFS) base(p string) string       { return path.Base(p) }

func (r *remoteFS) abs(p string) string {
	if path.IsAbs(p) {
		return path.Clean(p)
	}
	return path.Join(r.dir, p)
}

// SFTP returns the SFTP client of the connection to the host, which is
// made when first needed, for copying files to and from the host.
func (cl *Client) SFTP() (*sftp.Client, error) {
	if cl.sftpClient != nil {
		return cl.sftpClient, nil
	}
	if cl.Client == nil {
		return nil, errors.New("ssh: not connected")
	}
	sc, err := sftp.NewClient(cl.Client, sftp.UseConcurrentReads(true), sftp.UseConcurrentWrites(true))
	if err != nil {
		return nil, fmt.Errorf("ssh: starting SFTP on %s: %w", cl.Host, err)
	}
	cl.sftpClient = sc
	return sc, nil
}

// OpenFile opens the given file on the remote host, with the given flags
// of os.OpenFile, such as os.O_CREATE|os.O_WRONLY|os.O_TRUNC to write it,
// or os.O_APPEND to add to it, through SFTP. A relative path is relative to the current Dir of the
// Client.
func (cl *Client) OpenFile(name string, flag int) (io.ReadWriteCloser, error) {
	sc, err := cl.SFTP()
	if err != nil {
		return nil, err
	}
	rfs := &remoteFS{sc: sc, host: cl.Host, dir: cl.Dir}
	name = rfs.abs(name)
	f, err := sc.OpenFile(name, flag)
	if err != nil {
		return nil, rfs.pathError("open", name, err)
	}
	// SFTP servers can ignore the append flag, as OpenSSH does, so the
	// writes start at the end.
	if flag&os.O_APPEND != 0 {
		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			f.Close()
			return nil, rfs.pathError("seek", name, err)
		}
	}
	return f, nil
}

// CopyLocalToHost copies the data of the given reader to the given file on
// the remote host. The size is not needed. A relative host path is
// relative to the current Dir of the Client.
func (cl *Client) CopyLocalToHost(ctx context.Context, r io.Reader, size int64, hostFilename string) error {
	sc, err := cl.SFTP()
	if err != nil {
		return err
	}
	rfs := &remoteFS{sc: sc, host: cl.Host, dir: cl.Dir}
	w, err := sc.Create(rfs.abs(hostFilename))
	if err != nil {
		return err
	}
	stop := context.AfterFunc(ctx, func() { w.Close() })
	defer stop()
	_, err = io.Copy(w, r)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

// CopyHostToLocal copies the given file on the remote host to the given
// writer. A relative host path is relative to the current Dir of the Client.
func (cl *Client) CopyHostToLocal(ctx context.Context, hostFilename string, w io.Writer) error {
	sc, err := cl.SFTP()
	if err != nil {
		return err
	}
	rfs := &remoteFS{sc: sc, host: cl.Host, dir: cl.Dir}
	r, err := sc.Open(rfs.abs(hostFilename))
	if err != nil {
		return err
	}
	defer r.Close()
	stop := context.AfterFunc(ctx, func() { r.Close() })
	defer stop()
	_, err = io.Copy(w, r)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}
