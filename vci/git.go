// Copyright (c) 2019, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"bufio"
	"bytes"
	"fmt"
	"log"

	"github.com/Masterminds/vcs"
)

type GitRepo struct {
	vcs.GitRepo
}

func (gr *GitRepo) Files() (Files, error) {
	f := make(Files, 1000)

	out, err := gr.RunFromDir("git", "ls-files", "-o") // other -- untracked
	if err != nil {
		return nil, err
	}
	scan := bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := string(scan.Bytes())
		f[fn] = Untracked
	}

	out, err = gr.RunFromDir("git", "ls-files", "-c") // cached = all in repo
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := string(scan.Bytes())
		f[fn] = Stored
	}

	out, err = gr.RunFromDir("git", "ls-files", "-m") // modified
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := string(scan.Bytes())
		f[fn] = Modified
	}

	out, err = gr.RunFromDir("git", "ls-files", "-d") // deleted
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := string(scan.Bytes())
		f[fn] = Deleted
	}

	out, err = gr.RunFromDir("git", "ls-files", "-u") // unmerged
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := string(scan.Bytes())
		f[fn] = Conflicted
	}

	out, err = gr.RunFromDir("git", "diff", "--name-only", "--diff-filter=A", "HEAD") // deleted
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := string(scan.Bytes())
		f[fn] = Added
	}

	return f, nil
}

// Add adds the file to the repo
func (gr *GitRepo) Add(filename string) error {
	out, err := gr.RunFromDir("git", "add", filename)
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// Move moves updates the repo with the rename
func (gr *GitRepo) Move(oldpath, newpath string) error {
	out, err := gr.RunFromDir("git", "mv", oldpath, newpath)
	if err != nil {
		log.Println(string(out))
		fmt.Printf("%s\n", out)
		return err
	}
	return nil
}

// Delete removes the file from the repo
func (gr *GitRepo) Delete(filename string) error {
	out, err := gr.RunFromDir("git", "rm", filename)
	if err != nil {
		log.Println(string(out))
		fmt.Printf("%s\n", out)
		return err
	}
	return nil
}

// Delete removes the file from the repo
func (gr *GitRepo) DeleteKeepLocal(filename string) error {
	out, err := gr.RunFromDir("git", "rm", "--cached", filename)
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// CommitFile commits single file to repo staging
func (gr *GitRepo) CommitFile(filename string, message string) error {
	out, err := gr.RunFromDir("git", "commit", filename, "-m", message)
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// RevertFile reverts a single file to last commit of master
func (gr *GitRepo) RevertFile(filename string) error {
	out, err := gr.RunFromDir("git", "checkout", filename)
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}
