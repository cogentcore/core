// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// responses are responses of the API by path, in the form of those of GitHub,
// with lowercase names.
var responses = map[string]string{
	"/repos/me/lists/git/ref/heads/main": `{"ref":"refs/heads/main","node_id":"REF_1","url":"https://api.github.com/repos/me/lists/git/refs/heads/main","object":{"sha":"c2","type":"commit","url":"https://api.github.com/repos/me/lists/git/commits/c2"}}`,
	"/repos/me/lists/git/commits/c2":     `{"sha":"c2","node_id":"C_2","message":"Edit Chores","tree":{"sha":"t2","url":"https://x"},"parents":[{"sha":"c1","url":"https://x","html_url":"https://x"}],"author":{"name":"Me","email":"me@example.com","date":"2026-11-03T10:00:00Z"},"committer":{"name":"Me","email":"me@example.com","date":"2026-11-03T10:05:00Z"},"verification":{"verified":false}}`,
	"/repos/me/lists/git/trees/t2":       `{"sha":"t2","url":"https://x","tree":[{"path":"tree.json","mode":"100644","type":"blob","sha":"b1","size":2,"url":"https://x"},{"path":"lists","mode":"040000","type":"tree","sha":"t3","url":"https://x"}],"truncated":false}`,
	"/repos/me/lists/git/blobs/b1":       `{"sha":"b1","node_id":"B_1","size":3,"url":"https://x","content":"e30K\n","encoding":"base64"}`,
}

// fakeTransport responds with the [responses], without a network.
type fakeTransport struct{}

func (fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body, ok := responses[req.URL.Path]
	status := http.StatusOK
	if !ok {
		status, body = http.StatusNotFound, `{"message":"Not Found","documentation_url":"https://x","status":"404"}`
	}
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

// TestResponses tests decoding the responses of the API, which have
// lowercase names, also in JavaScript (see TestGoScript).
func TestResponses(t *testing.T) {
	c := &Client{API: "https://api.example.com", HTTP: &http.Client{Transport: fakeTransport{}}}
	head, err := c.Head("me/lists", "main")
	if err != nil || head != "c2" {
		t.Fatalf("Head: %q, %v", head, err)
	}
	cm, err := c.Commit("me/lists", head)
	if err != nil {
		t.Fatal(err)
	}
	if cm.Tree != "t2" || cm.Message != "Edit Chores" || len(cm.Parents) != 1 || cm.Parents[0] != "c1" {
		t.Errorf("Commit: %+v", cm)
	}
	if want := time.Date(2026, 11, 3, 10, 5, 0, 0, time.UTC); !cm.Date.Equal(want) {
		t.Errorf("Commit date: %v", cm.Date)
	}
	files, err := c.Files("me/lists", cm.Tree)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files["tree.json"].SHA != "b1" {
		t.Errorf("Files: %+v", files)
	}
	b, err := c.Blob("me/lists", "b1")
	if err != nil || string(b) != "{}\n" {
		t.Errorf("Blob: %q, %v", b, err)
	}
	_, err = c.Head("me/lists", "other")
	if e, ok := err.(*Error); !ok || e.Status != http.StatusNotFound || e.Message != "Not Found" {
		t.Errorf("Head of a missing branch: %v", err)
	}
}
