// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package github is a client of the GitHub REST API for the git data of
// repositories: branches, commits, trees and files, as for storing and
// syncing the files of an app in a repository. It uses only HTTPS
// requests with a token, which work on all platforms, including the web,
// as the API allows requests from browsers, unlike the git protocol.
package github

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultAPI is the URL of the GitHub REST API.
const DefaultAPI = "https://api.github.com"

// Client is a client of the GitHub REST API. Its methods take the
// repository as "owner/name".
type Client struct {

	// Token is the token for the requests, such as a fine-grained personal
	// access token with read and write access to the contents of the
	// repositories, or none for public ones.
	Token string

	// API is the URL of the API, which is [DefaultAPI] if empty,
	// or that of a test server (see package githubtest).
	API string

	// HTTP is the client for the requests, which is
	// [http.DefaultClient] if nil.
	HTTP *http.Client

	// etags are the ETags and bodies of the last responses to GET requests,
	// by path, for conditional requests: a response that is not modified
	// does not count against the rate limit of the API.
	etags map[string]etagged

	mu sync.Mutex
}

// etagged is the ETag and body of the last response to a GET request.
type etagged struct {
	etag string
	body []byte
}

// Error is an error response of the API.
type Error struct {

	// Status is the HTTP status code of the response.
	Status int

	// Message is the message of the response.
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("github: %d %s", e.Status, e.Message)
}

// StatusOf returns the HTTP status code of the given error, if it is an
// [Error], and otherwise 0.
func StatusOf(err error) int {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Status
	}
	return 0
}

// ErrNotFastForward is the error of [Client.UpdateRef] when the branch
// has commits that the new commit does not have, as when it was updated
// since it was read.
var ErrNotFastForward = errors.New("github: the update of the branch is not a fast forward")

// Do sends a request with the given method to the given path of the API,
// with the given body as JSON if it is not nil, and decodes the JSON of
// the response into the given result if it is not nil. GET requests are
// conditional on the ETag of the last response to the same path, which
// is used again if it is not modified.
func (c *Client) Do(method, path string, body, result any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	api := c.API
	if api == "" {
		api = DefaultAPI
	}
	req, err := http.NewRequest(method, strings.TrimSuffix(api, "/")+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.mu.Lock()
	prev, hasPrev := c.etags[path]
	c.mu.Unlock()
	if method == http.MethodGet && hasPrev {
		req.Header.Set("If-None-Match", prev.etag)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// a response that is not modified has the body of the last one
	switch {
	case resp.StatusCode == http.StatusNotModified && hasPrev:
		data = prev.body
	case resp.StatusCode >= 300:
		var msg struct{ Message string }
		json.Unmarshal(data, &msg)
		return &Error{Status: resp.StatusCode, Message: msg.Message}
	case method == http.MethodGet && resp.Header.Get("ETag") != "":
		c.mu.Lock()
		if c.etags == nil {
			c.etags = map[string]etagged{}
		}
		c.etags[path] = etagged{etag: resp.Header.Get("ETag"), body: data}
		c.mu.Unlock()
	}
	if result == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, result)
}

// Commit is a commit of a repository.
type Commit struct {

	// SHA is the SHA of the commit.
	SHA string

	// Tree is the SHA of the tree of the files of the commit.
	Tree string

	// Parents are the SHAs of the parents of the commit.
	Parents []string

	// Message is the message of the commit.
	Message string

	// Date is when the commit was made.
	Date time.Time
}

// Head returns the SHA of the last commit of the given branch of the
// given repository. A repository without commits, which the API does not
// support otherwise, has the status 409 (see [StatusOf] and [Client.PutFile]),
// and one without the branch 404.
func (c *Client) Head(repo, branch string) (string, error) {
	var ref struct {
		Object struct{ SHA string }
	}
	if err := c.Do(http.MethodGet, "/repos/"+repo+"/git/ref/heads/"+branch, nil, &ref); err != nil {
		return "", err
	}
	return ref.Object.SHA, nil
}

// Commit returns the commit with the given SHA of the given repository.
func (c *Client) Commit(repo, sha string) (*Commit, error) {
	var cm struct {
		SHA     string
		Message string
		Tree    struct{ SHA string }
		Parents []struct{ SHA string }
		// the time of the commit, after that of its author for a rebase
		Committer struct{ Date time.Time }
	}
	if err := c.Do(http.MethodGet, "/repos/"+repo+"/git/commits/"+sha, nil, &cm); err != nil {
		return nil, err
	}
	res := &Commit{SHA: cm.SHA, Tree: cm.Tree.SHA, Message: cm.Message, Date: cm.Committer.Date}
	for _, p := range cm.Parents {
		res.Parents = append(res.Parents, p.SHA)
	}
	return res, nil
}

// TreeEntry is a file or directory in a tree of a repository.
type TreeEntry struct {

	// Path is the path of the file in the tree, with slashes.
	Path string

	// Type is blob for a file, and tree for a directory.
	Type string

	// SHA is the SHA of the blob of a file (see [BlobSHA]),
	// or of the tree of a directory.
	SHA string
}

// Files returns the files of the tree with the given SHA of the given
// repository, with those in its directories, by path.
func (c *Client) Files(repo, tree string) (map[string]TreeEntry, error) {
	var tr struct {
		Tree      []TreeEntry
		Truncated bool
	}
	if err := c.Do(http.MethodGet, "/repos/"+repo+"/git/trees/"+tree+"?recursive=1", nil, &tr); err != nil {
		return nil, err
	}
	if tr.Truncated {
		return nil, fmt.Errorf("github: the tree %s of %s has too many files", tree, repo)
	}
	files := map[string]TreeEntry{}
	for _, e := range tr.Tree {
		if e.Type == "blob" {
			files[e.Path] = e
		}
	}
	return files, nil
}

// Blob returns the contents of the blob of a file with the given SHA of
// the given repository.
func (c *Client) Blob(repo, sha string) ([]byte, error) {
	var b struct{ Content, Encoding string }
	if err := c.Do(http.MethodGet, "/repos/"+repo+"/git/blobs/"+sha, nil, &b); err != nil {
		return nil, err
	}
	if b.Encoding != "base64" {
		return []byte(b.Content), nil
	}
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(b.Content, "\n", ""))
}

// BlobSHA returns the SHA of a blob of a file with the given contents,
// as git computes it, for comparing files with those of a tree
// without getting them.
func BlobSHA(content []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", len(content))
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// CreateTree creates a tree of the given repository, with the files of
// the tree with the given SHA, changed by the given files by path: a file
// is added or replaced with its contents, or deleted if they are nil.
// It returns the SHA of the new tree.
func (c *Client) CreateTree(repo, base string, files map[string][]byte) (string, error) {
	entries := make([]map[string]any, 0, len(files))
	for path, content := range files {
		e := map[string]any{"path": path, "mode": "100644", "type": "blob"}
		if content == nil {
			e["sha"] = nil
		} else {
			e["content"] = string(content)
		}
		entries = append(entries, e)
	}
	var res struct{ SHA string }
	err := c.Do(http.MethodPost, "/repos/"+repo+"/git/trees", map[string]any{"base_tree": base, "tree": entries}, &res)
	return res.SHA, err
}

// CreateCommit creates a commit of the given repository with the given
// message, tree and parents, returning its SHA. It does not update a
// branch (see [Client.UpdateRef]).
func (c *Client) CreateCommit(repo, message, tree string, parents ...string) (string, error) {
	var res struct{ SHA string }
	err := c.Do(http.MethodPost, "/repos/"+repo+"/git/commits", map[string]any{"message": message, "tree": tree, "parents": parents}, &res)
	return res.SHA, err
}

// UpdateRef updates the given branch of the given repository to the commit
// with the given SHA. Unless force, it must be a fast forward, with the
// last commit of the branch in its history, or the error is
// [ErrNotFastForward].
func (c *Client) UpdateRef(repo, branch, sha string, force bool) error {
	err := c.Do(http.MethodPatch, "/repos/"+repo+"/git/refs/heads/"+branch, map[string]any{"sha": sha, "force": force}, nil)
	if StatusOf(err) == http.StatusUnprocessableEntity {
		return fmt.Errorf("%w: %w", ErrNotFastForward, err)
	}
	return err
}

// CommitFiles commits the given files by path to the given branch of the
// given repository, after the given commit, the last one of the branch,
// as in [Client.CreateTree], returning the SHA of the new commit. It
// returns [ErrNotFastForward] if the branch has commits after the given
// one, which then must be merged.
func (c *Client) CommitFiles(repo, branch, parent, message string, files map[string][]byte) (string, error) {
	pc, err := c.Commit(repo, parent)
	if err != nil {
		return "", err
	}
	tree, err := c.CreateTree(repo, pc.Tree, files)
	if err != nil {
		return "", err
	}
	sha, err := c.CreateCommit(repo, message, tree, parent)
	if err != nil {
		return "", err
	}
	return sha, c.UpdateRef(repo, branch, sha, false)
}

// PutFile commits a file with the given path and contents to the given
// branch of the given repository, with the contents API, which also works
// for a repository without commits, for its first commit.
func (c *Client) PutFile(repo, branch, path, message string, content []byte) error {
	body := map[string]any{"message": message, "branch": branch, "content": base64.StdEncoding.EncodeToString(content)}
	return c.Do(http.MethodPut, "/repos/"+repo+"/contents/"+path, body, nil)
}
