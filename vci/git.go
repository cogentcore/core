// Copyright (c) 2019, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"

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
		fn := filepath.FromSlash(string(scan.Bytes()))
		f[fn] = Untracked
	}

	out, err = gr.RunFromDir("git", "ls-files", "-c") // cached = all in repo
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := filepath.FromSlash(string(scan.Bytes()))
		f[fn] = Stored
	}

	out, err = gr.RunFromDir("git", "ls-files", "-m") // modified
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := filepath.FromSlash(string(scan.Bytes()))
		f[fn] = Modified
	}

	out, err = gr.RunFromDir("git", "ls-files", "-d") // deleted
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := filepath.FromSlash(string(scan.Bytes()))
		f[fn] = Deleted
	}

	out, err = gr.RunFromDir("git", "ls-files", "-u") // unmerged
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := filepath.FromSlash(string(scan.Bytes()))
		f[fn] = Conflicted
	}

	out, err = gr.RunFromDir("git", "diff", "--name-only", "--diff-filter=A", "HEAD") // deleted
	if err != nil {
		return nil, err
	}
	scan = bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		fn := filepath.FromSlash(string(scan.Bytes()))
		f[fn] = Added
	}

	return f, nil
}

func (gr *GitRepo) CharToStat(stat byte) FileStatus {
	switch stat {
	case 'M':
		return Modified
	case 'A':
		return Added
	case 'D':
		return Deleted
	case 'U':
		return Conflicted
	case '?', '!':
		return Untracked
	}
	return Untracked
}

// Status returns status of given file -- returns Untracked on any error
func (gr *GitRepo) Status(fname string) (FileStatus, string) {
	out, err := gr.RunFromDir("git", "status", "--porcelain", RelPath(gr, fname))
	if err != nil {
		return Untracked, err.Error()
	}
	ostr := string(out)
	if ostr == "" {
		return Stored, ""
	}
	sf := strings.Fields(ostr)
	if len(sf) < 2 {
		return Stored, ostr
	}
	stat := sf[0][0]
	return gr.CharToStat(stat), ostr
}

// Add adds the file to the repo
func (gr *GitRepo) Add(fname string) error {
	out, err := gr.RunFromDir("git", "add", RelPath(gr, fname))
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

// Delete removes the file from the repo -- uses "force" option to ensure deletion
func (gr *GitRepo) Delete(fname string) error {
	out, err := gr.RunFromDir("git", "rm", "-f", RelPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		fmt.Printf("%s\n", out)
		return err
	}
	return nil
}

// Delete removes the file from the repo
func (gr *GitRepo) DeleteRemote(fname string) error {
	out, err := gr.RunFromDir("git", "rm", "--cached", RelPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// CommitFile commits single file to repo staging
func (gr *GitRepo) CommitFile(fname string, message string) error {
	out, err := gr.RunFromDir("git", "commit", RelPath(gr, fname), "-m", message)
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// RevertFile reverts a single file to last commit of master
func (gr *GitRepo) RevertFile(fname string) error {
	out, err := gr.RunFromDir("git", "checkout", RelPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// FileContents returns the contents of given file, as a []byte array
// at given revision specifier (if empty, defaults to current HEAD).
// -1, -2 etc also work as universal ways of specifying prior revisions.
func (gr *GitRepo) FileContents(fname string, rev string) ([]byte, error) {
	if rev == "" {
		rev = "HEAD:"
	} else if rev[0] == '-' {
		rsp, err := strconv.Atoi(rev)
		if err == nil && rsp < 0 {
			rev = fmt.Sprintf("HEAD~%d:", -rsp)
		}
	} else {
		rev += ":"
	}
	fspec := rev + RelPath(gr, fname)
	out, err := gr.RunFromDir("git", "show", fspec)
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
}

// FieldsThroughDelim gets the concatenated byte through to point where
// field ends with given delimiter, starting at given index
func FieldsThroughDelim(flds [][]byte, delim byte, idx int) (int, string) {
	ln := len(flds)
	for i := idx; i < ln; i++ {
		fld := flds[i]
		fsz := len(fld)
		if fld[fsz-1] == delim {
			str := string(bytes.Join(flds[idx:i+1], []byte(" ")))
			return i + 1, str[:len(str)-1]
		}
	}
	return ln, string(bytes.Join(flds[idx:ln], []byte(" ")))
}

// Log returns the log history of commits for given filename
// (or all files if empty).  If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc
func (gr *GitRepo) Log(fname string, since string) (Log, error) {
	args := []string{"log", "--all"}
	if since != "" {
		args = append(args, `--since="`+since+`"`)
	}
	args = append(args, `--pretty=format:%h %ad} %an} %ae} %s`)
	if fname != "" {
		args = append(args, fname)
	}
	out, err := gr.RunFromDir("git", args...)
	if err != nil {
		return nil, err
	}
	var lg Log
	scan := bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		ln := scan.Bytes()
		flds := bytes.Fields(ln)
		if len(flds) < 4 {
			continue
		}
		rev := string(flds[0])
		ni, date := FieldsThroughDelim(flds, '}', 1)
		ni, author := FieldsThroughDelim(flds, '}', ni)
		ni, email := FieldsThroughDelim(flds, '}', ni)
		msg := string(bytes.Join(flds[ni:], []byte(" ")))
		lg.Add(rev, date, author, email, msg)
	}
	return lg, nil
}

// Blame returns an annotated report about the file, showing which revision last
// modified each line.
func (gr *GitRepo) Blame(fname string) ([]byte, error) {
	out, err := gr.RunFromDir("git", "blame", fname)
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
}
