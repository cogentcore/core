// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package githubtest is a test server of the parts of the GitHub REST API
// that package github uses, with repositories in memory.
package githubtest

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/github"
)

// Server is a test server of the GitHub REST API, as for a [github.Client]
// with [Server.URL] as its API. The repositories are made when they are
// first used, without commits.
type Server struct {
	*httptest.Server

	// Token, if set, is the token that the requests must have.
	Token string

	// Requests is the number of requests to the server, without
	// those that are not modified, as for the rate limit.
	Requests int

	repos map[string]*repo
	mu    sync.Mutex
}

// repo is a repository of the test server.
type repo struct {
	blobs   map[string][]byte
	trees   map[string]map[string]string // the blobs of the files by path
	commits map[string]*commit
	refs    map[string]string // the commits of the branches
}

// commit is a commit of a repository of the test server.
type commit struct {
	tree, message string
	parents       []string
	date          time.Time
}

// NewServer returns a new running test server, to be closed after use.
func NewServer() *Server {
	s := &Server{repos: map[string]*repo{}}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

// Client returns a new client of the server.
func (s *Server) Client() *github.Client {
	return &github.Client{API: s.URL, Token: s.Token}
}

// Files returns the files of the last commit of the given branch of the
// given repository, by path, or nil if it has none.
func (s *Server) Files(repoName, branch string) map[string][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.repo(repoName)
	c := r.commits[r.refs[branch]]
	if c == nil {
		return nil
	}
	files := map[string][]byte{}
	for path, sha := range r.trees[c.tree] {
		files[path] = r.blobs[sha]
	}
	return files
}

// Commits returns the messages of the commits of the given branch of the
// given repository, from the last one, along the first parents.
func (s *Server) Commits(repoName, branch string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.repo(repoName)
	var msgs []string
	for c := r.commits[r.refs[branch]]; c != nil; {
		msgs = append(msgs, c.message)
		if len(c.parents) == 0 {
			break
		}
		c = r.commits[c.parents[0]]
	}
	return msgs
}

// repo returns the repository with the given name, making it if needed.
func (s *Server) repo(name string) *repo {
	r := s.repos[name]
	if r == nil {
		r = &repo{blobs: map[string][]byte{}, trees: map[string]map[string]string{}, commits: map[string]*commit{}, refs: map[string]string{}}
		s.repos[name] = r
	}
	return r
}

// hash returns a SHA for the given object of a repository.
func hash(kind string, v any) string {
	b, _ := json.Marshal(v)
	h := sha1.Sum(append([]byte(kind+" "), b...))
	return hex.EncodeToString(h[:])
}

// addTree adds the tree with the given files to the repository.
func (r *repo) addTree(files map[string]string) string {
	sha := hash("tree", files)
	r.trees[sha] = files
	return sha
}

// addCommit adds the commit to the repository.
func (r *repo) addCommit(c *commit) string {
	sha := hash("commit", []any{c.tree, c.message, c.parents, c.date})
	r.commits[sha] = c
	return sha
}

// addBlob adds the blob with the given contents to the repository.
func (r *repo) addBlob(content []byte) string {
	sha := github.BlobSHA(content)
	r.blobs[sha] = content
	return sha
}

// handle handles a request of the API.
func (s *Server) handle(w http.ResponseWriter, req *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Token != "" && req.Header.Get("Authorization") != "Bearer "+s.Token {
		writeError(w, http.StatusUnauthorized, "Bad credentials")
		return
	}
	// /repos/owner/name/rest...
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/repos/"), "/", 3)
	if len(parts) < 3 {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	r := s.repo(parts[0] + "/" + parts[1])
	rest := parts[2]
	var body map[string]any
	if req.Body != nil {
		json.NewDecoder(req.Body).Decode(&body)
	}
	if !strings.HasPrefix(rest, "contents/") && len(r.refs) == 0 {
		writeError(w, http.StatusConflict, "Git Repository is empty.")
		return
	}
	switch {
	case req.Method == http.MethodGet && strings.HasPrefix(rest, "git/ref/heads/"):
		sha, ok := r.refs[strings.TrimPrefix(rest, "git/ref/heads/")]
		if !ok {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		s.write(w, req, map[string]any{"object": map[string]any{"sha": sha, "type": "commit"}})
	case req.Method == http.MethodGet && strings.HasPrefix(rest, "git/commits/"):
		c := r.commits[strings.TrimPrefix(rest, "git/commits/")]
		if c == nil {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		var parents []map[string]any
		for _, p := range c.parents {
			parents = append(parents, map[string]any{"sha": p})
		}
		s.write(w, req, map[string]any{"sha": strings.TrimPrefix(rest, "git/commits/"), "message": c.message,
			"tree": map[string]any{"sha": c.tree}, "parents": parents, "committer": map[string]any{"date": c.date}})
	case req.Method == http.MethodGet && strings.HasPrefix(rest, "git/trees/"):
		files, ok := r.trees[strings.TrimPrefix(rest, "git/trees/")]
		if !ok {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		var entries []map[string]any
		for path, sha := range files {
			entries = append(entries, map[string]any{"path": path, "type": "blob", "sha": sha})
		}
		s.write(w, req, map[string]any{"tree": entries})
	case req.Method == http.MethodGet && strings.HasPrefix(rest, "git/blobs/"):
		b, ok := r.blobs[strings.TrimPrefix(rest, "git/blobs/")]
		if !ok {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		s.write(w, req, map[string]any{"content": base64.StdEncoding.EncodeToString(b), "encoding": "base64"})
	case req.Method == http.MethodPost && rest == "git/trees":
		s.createTree(w, req, r, body)
	case req.Method == http.MethodPost && rest == "git/commits":
		c := &commit{tree: fmt.Sprint(body["tree"]), message: fmt.Sprint(body["message"]), date: time.Now()}
		if _, ok := r.trees[c.tree]; !ok {
			writeError(w, http.StatusUnprocessableEntity, "Tree SHA does not exist")
			return
		}
		for _, p := range body["parents"].([]any) {
			c.parents = append(c.parents, fmt.Sprint(p))
		}
		s.write(w, req, map[string]any{"sha": r.addCommit(c)})
	case req.Method == http.MethodPatch && strings.HasPrefix(rest, "git/refs/heads/"):
		branch := strings.TrimPrefix(rest, "git/refs/heads/")
		sha := fmt.Sprint(body["sha"])
		if r.commits[sha] == nil {
			writeError(w, http.StatusUnprocessableEntity, "Object does not exist")
			return
		}
		if force, _ := body["force"].(bool); !force && !r.hasAncestor(sha, r.refs[branch]) {
			writeError(w, http.StatusUnprocessableEntity, "Update is not a fast forward")
			return
		}
		r.refs[branch] = sha
		s.write(w, req, map[string]any{"object": map[string]any{"sha": sha}})
	case req.Method == http.MethodPut && strings.HasPrefix(rest, "contents/"):
		s.putFile(w, r, strings.TrimPrefix(rest, "contents/"), body)
	default:
		writeError(w, http.StatusNotFound, "Not Found")
	}
}

// createTree handles a request to create a tree.
func (s *Server) createTree(w http.ResponseWriter, req *http.Request, r *repo, body map[string]any) {
	files := map[string]string{}
	if base, _ := body["base_tree"].(string); base != "" {
		bt, ok := r.trees[base]
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "base_tree is not a valid tree")
			return
		}
		files = maps.Clone(bt)
	}
	for _, e := range body["tree"].([]any) {
		e := e.(map[string]any)
		path := fmt.Sprint(e["path"])
		if content, ok := e["content"].(string); ok {
			files[path] = r.addBlob([]byte(content))
		} else if sha, ok := e["sha"].(string); ok {
			files[path] = sha
		} else {
			delete(files, path)
		}
	}
	s.write(w, req, map[string]any{"sha": r.addTree(files)})
}

// putFile handles a request to commit a file with the contents API.
func (s *Server) putFile(w http.ResponseWriter, r *repo, path string, body map[string]any) {
	branch, _ := body["branch"].(string)
	if branch == "" {
		branch = "main"
	}
	content, err := base64.StdEncoding.DecodeString(fmt.Sprint(body["content"]))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "content is not valid Base64")
		return
	}
	files := map[string]string{}
	var parents []string
	if head, ok := r.refs[branch]; ok {
		files = maps.Clone(r.trees[r.commits[head].tree])
		parents = []string{head}
	} else if len(r.refs) > 0 {
		writeError(w, http.StatusNotFound, "Branch not found")
		return
	}
	files[path] = r.addBlob(content)
	c := &commit{tree: r.addTree(files), message: fmt.Sprint(body["message"]), parents: parents, date: time.Now()}
	r.refs[branch] = r.addCommit(c)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"commit": map[string]any{"sha": r.refs[branch]}})
	s.Requests++
}

// hasAncestor returns whether the given commit has the given ancestor,
// or is it, or the ancestor is none.
func (r *repo) hasAncestor(sha, ancestor string) bool {
	if ancestor == "" || sha == ancestor {
		return true
	}
	c := r.commits[sha]
	if c == nil {
		return false
	}
	for _, p := range c.parents {
		if r.hasAncestor(p, ancestor) {
			return true
		}
	}
	return false
}

// write writes the given response, with an ETag for a GET request,
// or not modified for one with the same ETag.
func (s *Server) write(w http.ResponseWriter, req *http.Request, v any) {
	b, _ := json.Marshal(v)
	if req.Method == http.MethodGet {
		h := sha1.Sum(b)
		etag := `"` + hex.EncodeToString(h[:]) + `"`
		w.Header().Set("ETag", etag)
		if req.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	s.Requests++
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// writeError writes an error response with the given status and message.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{"message": message})
}
