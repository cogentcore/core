// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cogentcore.org/core/base/github"
	"cogentcore.org/core/base/github/githubtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlobSHA(t *testing.T) {
	// as by git hash-object
	assert.Equal(t, "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391", github.BlobSHA(nil))
	assert.Equal(t, "ce013625030ba8dba906f756967f9e9ca394464a", github.BlobSHA([]byte("hello\n")))
}

// TestCommitFiles tests the first commit of an empty repository, committing
// files, reading them back, and that an update that is not a fast forward
// fails.
func TestCommitFiles(t *testing.T) {
	s := githubtest.NewServer()
	defer s.Close()
	s.Token = "secret"
	c := s.Client()
	const repo = "me/lists"

	_, err := c.Head(repo, "main")
	assert.Equal(t, http.StatusConflict, github.StatusOf(err), "empty")
	require.NoError(t, c.PutFile(repo, "main", "README.md", "first", []byte("# Lists\n")))
	first, err := c.Head(repo, "main")
	require.NoError(t, err)

	second, err := c.CommitFiles(repo, "main", first, "add", map[string][]byte{"a.json": []byte("{}\n"), "d/b.json": []byte("[]\n")})
	require.NoError(t, err)
	head, err := c.Head(repo, "main")
	require.NoError(t, err)
	assert.Equal(t, second, head)
	cm, err := c.Commit(repo, head)
	require.NoError(t, err)
	assert.Equal(t, []string{first}, cm.Parents)
	assert.Equal(t, "add", cm.Message)
	files, err := c.Files(repo, cm.Tree)
	require.NoError(t, err)
	assert.Len(t, files, 3)
	assert.Equal(t, github.BlobSHA([]byte("[]\n")), files["d/b.json"].SHA)
	b, err := c.Blob(repo, files["d/b.json"].SHA)
	require.NoError(t, err)
	assert.Equal(t, "[]\n", string(b))

	// a file deleted, after the first commit, which is not a fast forward
	_, err = c.CommitFiles(repo, "main", first, "delete", map[string][]byte{"a.json": nil})
	assert.True(t, errors.Is(err, github.ErrNotFastForward), err)
	_, err = c.CommitFiles(repo, "main", second, "delete", map[string][]byte{"a.json": nil})
	require.NoError(t, err)
	assert.Equal(t, map[string][]byte{"README.md": []byte("# Lists\n"), "d/b.json": []byte("[]\n")}, s.Files(repo, "main"))
	assert.Equal(t, []string{"delete", "add", "first"}, s.Commits(repo, "main"))

	// a head that is not modified is not a request for the rate limit
	last, err := c.Head(repo, "main")
	require.NoError(t, err)
	n := s.Requests
	head, err = c.Head(repo, "main")
	require.NoError(t, err)
	assert.Equal(t, last, head)
	assert.Equal(t, n, s.Requests)

	_, err = (&github.Client{API: s.URL}).Head(repo, "main")
	assert.Equal(t, http.StatusUnauthorized, github.StatusOf(err))
}

// TestEmptySHA tests that a response without a SHA is an error,
// and that there is no request for an empty SHA.
func TestEmptySHA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := &github.Client{API: srv.URL}
	_, err := c.Head("me/lists", "main")
	assert.Error(t, err)
	_, err = c.Commit("me/lists", "")
	assert.Error(t, err)
	_, err = c.CommitFiles("me/lists", "main", "", "message", nil)
	assert.Error(t, err)
}
