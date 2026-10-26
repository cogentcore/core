// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sshclient

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/exec"
	"cogentcore.org/core/base/sshclient/sshtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testClient returns a client connected to a new test server.
func testClient(t *testing.T) (*Client, *sshtest.Server) {
	s := sshtest.NewServer(t)
	cl := NewClient(NewConfig(&exec.Config{}))
	cl.User.KeyPath = s.KeyPath
	require.NoError(t, cl.Connect(s.Addr))
	t.Cleanup(cl.Close)
	return cl, s
}

// writeFiles writes the given files, as name: content, in the given
// directory, making their directories.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	for name, content := range files {
		fn := filepath.Join(dir, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(fn), 0755))
		require.NoError(t, os.WriteFile(fn, []byte(content), 0644))
	}
}

// assertFiles checks that the given files, as name: content, are in the
// given directory.
func assertFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if assert.NoError(t, err, name) {
			assert.Equal(t, content, string(b), name)
		}
	}
}

func TestCopy(t *testing.T) {
	cl, s := testClient(t)
	assert.Equal(t, s.Root, cl.HomeDir)
	ctx := context.Background()
	local := t.TempDir()
	t.Chdir(local)
	host := func(p string) Location { return Location{Client: cl, Path: p} }
	loc := func(p string) Location { return Location{Path: p} }

	// a remote file to the local directory, as scp @host:a.tsv .
	writeFiles(t, s.Root, map[string]string{"a.tsv": "a", "b.tsv": "b", "c.txt": "c", "sub/d.tsv": "d", "sub/e/f.txt": "f"})
	require.NoError(t, Copy(ctx, loc("."), host("a.tsv")))
	assertFiles(t, local, map[string]string{"a.tsv": "a"})

	// with a new name, and to a remote directory that is relative to the current one
	require.NoError(t, Copy(ctx, loc("renamed.tsv"), host("sub/d.tsv")))
	assertFiles(t, local, map[string]string{"renamed.tsv": "d"})

	// remote wildcards to a new local directory, as scp @host:*.tsv @0:data/
	require.NoError(t, Copy(ctx, loc("data/"), host("*.tsv")))
	assertFiles(t, local, map[string]string{"data/a.tsv": "a", "data/b.tsv": "b"})
	_, err := os.Stat(filepath.Join(local, "data", "c.txt"))
	assert.True(t, os.IsNotExist(err))
	assert.Error(t, Copy(ctx, loc("."), host("*.none")))

	// a remote directory, with all of its contents
	require.NoError(t, Copy(ctx, loc("."), host("sub")))
	assertFiles(t, local, map[string]string{"sub/d.tsv": "d", "sub/e/f.txt": "f"})

	// local files to the remote home directory and an existing directory
	writeFiles(t, local, map[string]string{"up.tsv": "up", "updir/x.txt": "x"})
	require.NoError(t, Copy(ctx, host("."), loc("up.tsv")))
	require.NoError(t, Copy(ctx, host("sub"), loc("updir"), loc("*.tsv")))
	assertFiles(t, s.Root, map[string]string{"up.tsv": "up", "sub/updir/x.txt": "x", "sub/a.tsv": "a", "sub/renamed.tsv": "d"})

	// to an absolute remote path
	require.NoError(t, Copy(ctx, host(filepath.Join(s.Root, "abs.tsv")), loc("up.tsv")))
	assertFiles(t, s.Root, map[string]string{"abs.tsv": "up"})

	// between two connections to remote hosts
	cl2 := NewClient(NewConfig(&exec.Config{}))
	cl2.User.KeyPath = s.KeyPath
	require.NoError(t, cl2.Connect(s.Addr))
	defer cl2.Close()
	require.NoError(t, Copy(ctx, Location{Client: cl2, Path: "copied/"}, host("a.tsv")))
	assertFiles(t, s.Root, map[string]string{"copied/a.tsv": "a"})

	// the old functions
	require.NoError(t, cl.CopyLocalFileToHost(ctx, "up.tsv", "up2.tsv"))
	require.NoError(t, cl.CopyHostToLocalFile(ctx, "up2.tsv", "down2.tsv"))
	assertFiles(t, local, map[string]string{"down2.tsv": "up"})
	require.NoError(t, cl.CopyLocalToHost(ctx, bytes.NewReader([]byte("reader")), 6, "r.txt"))
	var b bytes.Buffer
	require.NoError(t, cl.CopyHostToLocal(ctx, "r.txt", &b))
	assert.Equal(t, "reader", b.String())
}

// TestCopyLarge copies a large file both ways, which timed out after a
// minute with the scp protocol of go-scp.
func TestCopyLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("copies a large file")
	}
	cl, s := testClient(t)
	ctx := context.Background()
	local := t.TempDir()
	data := make([]byte, 64<<20)
	rand.Read(data)
	fn := filepath.Join(local, "large.bin")
	require.NoError(t, os.WriteFile(fn, data, 0644))
	require.NoError(t, cl.CopyLocalFileToHost(ctx, fn, "large.bin"))
	back := filepath.Join(local, "back.bin")
	require.NoError(t, cl.CopyHostToLocalFile(ctx, "large.bin", back))
	got, err := os.ReadFile(back)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(data, got))
	remote, err := os.ReadFile(filepath.Join(s.Root, "large.bin"))
	require.NoError(t, err)
	assert.True(t, bytes.Equal(data, remote))

	// a done context stops the copy
	cctx, cancel := context.WithCancel(ctx)
	cancel()
	assert.Error(t, cl.CopyHostToLocalFile(cctx, "large.bin", filepath.Join(local, "cancelled.bin")))
}

// TestConnectKeyFallback checks that a standard key file is used if the
// key file does not exist, as id_ed25519 is the default of ssh-keygen.
func TestConnectKeyFallback(t *testing.T) {
	s := sshtest.NewServer(t)
	require.NoError(t, os.Rename(filepath.Join(s.KeyPath, "id_rsa"), filepath.Join(s.KeyPath, "id_ed25519")))
	cl := NewClient(NewConfig(&exec.Config{}))
	cl.User.KeyPath = s.KeyPath
	require.NoError(t, cl.Connect(s.Addr))
	cl.Close()
}

// TestCopyProgress checks the progress of copies to and from a host.
func TestCopyProgress(t *testing.T) {
	cl, s := testClient(t)
	ctx := context.Background()
	local := t.TempDir()
	sizes := map[string]int{"a.bin": 3 << 20, "b.bin": 1 << 20, "c.bin": 0}
	for name, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)
		require.NoError(t, os.WriteFile(filepath.Join(local, name), data, 0644))
	}
	total := int64(4 << 20)
	check := func(dst Location, srcs ...Location) {
		var ps []Progress
		require.NoError(t, CopyProgress(ctx, func(p Progress) { ps = append(ps, p) }, dst, srcs...))
		require.NotEmpty(t, ps)
		last := ps[len(ps)-1]
		assert.True(t, last.Finished)
		assert.Equal(t, total, last.Total)
		assert.Equal(t, total, last.TotalDone)
		assert.Equal(t, 3, last.NFiles)
		assert.Equal(t, 3, last.FileNum)
		for i := 1; i < len(ps); i++ {
			assert.GreaterOrEqual(t, ps[i].TotalDone, ps[i-1].TotalDone)
			assert.LessOrEqual(t, ps[i].Done, ps[i].Size)
		}
	}
	check(Location{Client: cl, Path: "up/"}, Location{Path: filepath.Join(local, "*.bin")})
	check(Location{Path: filepath.Join(local, "down")}, Location{Client: cl, Path: "up"})
	for name, size := range sizes {
		info, err := os.Stat(filepath.Join(s.Root, "up", name))
		require.NoError(t, err)
		assert.Equal(t, int64(size), info.Size())
		info, err = os.Stat(filepath.Join(local, "down", name))
		require.NoError(t, err)
		assert.Equal(t, int64(size), info.Size())
	}
}
