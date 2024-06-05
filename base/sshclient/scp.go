// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bramvdbogaerde/go-scp"
)

// CopyLocalFileToHost copies given local file to given host file
// on the already-connected remote host, using the 'scp' protocol.
// See ScpPath in Config for path to scp on remote host.
// If the host filename is not absolute (i.e, doesn't start with /)
// then the current Dir path on the Client is prepended to the target path.
// Use context.Background() for a basic context if none otherwise in use.
func (cl *Client) CopyLocalFileToHost(ctx context.Context, localFilename, hostFilename string) error {
	f, err := os.Open(localFilename)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	return cl.CopyLocalToHost(ctx, f, stat.Size(), hostFilename)
}

// CopyLocalToHost copies given io.Reader source data to given filename
// on the already-connected remote host, using the 'scp' protocol.
// See ScpPath in Config for path to scp on remote host.
// The size must be given in advance for the scp protocol.
// If the host filename is not absolute (i.e, doesn't start with /)
// then the current Dir path on the Client is prepended to the target path.
// Use context.Background() for a basic context if none otherwise in use.
func (cl *Client) CopyLocalToHost(ctx context.Context, r io.Reader, size int64, hostFilename string) error {
	if err := cl.mustScpClient(); err != nil {
		return err
	}
	if !filepath.IsAbs(hostFilename) {
		hostFilename = filepath.Join(cl.Dir, hostFilename)
	}
	return cl.scpClient.CopyPassThru(ctx, r, hostFilename, "0666", size, nil)
}

// CopyHostToLocalFile copies given filename on the already-connected remote host,
// to the local file using the 'scp' protocol.
// See ScpPath in Config for path to scp on remote host.
// If the host filename is not absolute (i.e, doesn't start with /)
// then the current Dir path on the Client is prepended to the target path.
// Use context.Background() for a basic context if none otherwise in use.
func (cl *Client) CopyHostToLocalFile(ctx context.Context, hostFilename, localFilename string) error {
	f, err := os.Create(localFilename)
	if err != nil {
		return err
	}
	defer f.Close()
	return cl.CopyHostToLocal(ctx, hostFilename, f)
}

// CopyHostToLocal copies given filename on the already-connected remote host,
// to the local io.Writer using the 'scp' protocol.
// See ScpPath in Config for path to scp on remote host.
// If the host filename is not absolute (i.e, doesn't start with /)
// then the current Dir path on the Client is prepended to the target path.
// Use context.Background() for a basic context if none otherwise in use.
func (cl *Client) CopyHostToLocal(ctx context.Context, hostFilename string, w io.Writer) error {
	if err := cl.mustScpClient(); err != nil {
		return err
	}
	if !filepath.IsAbs(hostFilename) {
		hostFilename = filepath.Join(cl.Dir, hostFilename)
	}
	return cl.scpClient.CopyFromRemotePassThru(ctx, w, hostFilename, nil)
}

func (cl *Client) mustScpClient() error {
	if cl.scpClient != nil {
		return nil
	}
	scl, err := scp.NewClientBySSH(cl.Client)
	if err != nil {
		slog.Error(err.Error())
	} else {
		cl.scpClient = &scl
	}
	return err
}
