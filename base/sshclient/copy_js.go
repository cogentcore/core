// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"context"
	"errors"
	"io"
	"io/fs"
)

// Copying files to and from remote hosts uses SFTP, which is not
// available on the web, where SSH connections can not be made.

// errNoSFTP is the error of copying files to and from a remote host on the
// web.
var errNoSFTP = errors.New("ssh: copying files with remote hosts is not available on the web")

// sftpConn is the SFTP client of a connection, which is never made.
type sftpConn = io.Closer

// locationFS returns the file system of the host of the given location,
// which must be the local host.
func locationFS(l Location) (fileSystem, error) {
	if l.Client != nil {
		return nil, errNoSFTP
	}
	return localFS{}, nil
}

// sftpReaderFrom returns false: no file is on a remote host.
func sftpReaderFrom(w io.Writer) (io.ReaderFrom, bool) {
	return nil, false
}

// OpenFile returns an error on the web.
func (cl *Client) OpenFile(name string, flag int) (io.ReadWriteCloser, error) {
	return nil, errNoSFTP
}

// ReadDir returns an error on the web.
func (cl *Client) ReadDir(dir string) ([]fs.FileInfo, error) {
	return nil, errNoSFTP
}

// Stat returns an error on the web.
func (cl *Client) Stat(name string) (fs.FileInfo, error) {
	return nil, errNoSFTP
}

// CopyLocalToHost returns an error on the web.
func (cl *Client) CopyLocalToHost(ctx context.Context, r io.Reader, size int64, hostFilename string) error {
	return errNoSFTP
}

// CopyHostToLocal returns an error on the web.
func (cl *Client) CopyHostToLocal(ctx context.Context, hostFilename string, w io.Writer) error {
	return errNoSFTP
}
