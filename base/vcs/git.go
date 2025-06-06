// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vcs

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Masterminds/vcs"
)

type GitRepo struct {
	vcs.GitRepo
	files        Files
	gettingFiles bool
	sync.Mutex
}

func (gr *GitRepo) Type() Types {
	return Git
}

// Files returns a map of the current files and their status,
// using a cached version of the file list if available.
// nil will be returned immediately if no cache is available.
// The given onUpdated function will be called from a separate
// goroutine when the updated list of the files is available,
// if an update is not already under way. An update is always triggered
// if no files have yet been cached, even if the function is nil.
func (gr *GitRepo) Files(onUpdated func(f Files)) (Files, error) {
	gr.Lock()
	if gr.files != nil {
		f := gr.files
		gr.Unlock()
		if onUpdated != nil {
			go gr.updateFiles(onUpdated)
		}
		return f, nil
	}
	gr.Unlock()
	go gr.updateFiles(onUpdated)
	return nil, nil
}

func (gr *GitRepo) updateFiles(onUpdated func(f Files)) {
	gr.Lock()
	if gr.gettingFiles {
		gr.Unlock()
		return
	}
	gr.gettingFiles = true
	gr.Unlock()

	nf := max(len(gr.files), 64)
	f := make(Files, nf)

	out, err := gr.RunFromDir("git", "ls-files", "-o") // other -- untracked
	if err == nil {
		scan := bufio.NewScanner(bytes.NewReader(out))
		for scan.Scan() {
			fn := filepath.FromSlash(string(scan.Bytes()))
			f[fn] = Untracked
		}
	}

	out, err = gr.RunFromDir("git", "ls-files", "-c") // cached = all in repo
	if err == nil {
		scan := bufio.NewScanner(bytes.NewReader(out))
		for scan.Scan() {
			fn := filepath.FromSlash(string(scan.Bytes()))
			f[fn] = Stored
		}
	}

	out, err = gr.RunFromDir("git", "ls-files", "-m") // modified
	if err == nil {
		scan := bufio.NewScanner(bytes.NewReader(out))
		for scan.Scan() {
			fn := filepath.FromSlash(string(scan.Bytes()))
			f[fn] = Modified
		}
	}

	out, err = gr.RunFromDir("git", "ls-files", "-d") // deleted
	if err == nil {
		scan := bufio.NewScanner(bytes.NewReader(out))
		for scan.Scan() {
			fn := filepath.FromSlash(string(scan.Bytes()))
			f[fn] = Deleted
		}
	}

	out, err = gr.RunFromDir("git", "ls-files", "-u") // unmerged
	if err == nil {
		scan := bufio.NewScanner(bytes.NewReader(out))
		for scan.Scan() {
			fn := filepath.FromSlash(string(scan.Bytes()))
			f[fn] = Conflicted
		}
	}

	out, err = gr.RunFromDir("git", "diff", "--name-only", "--diff-filter=A", "HEAD") // deleted
	if err == nil {
		scan := bufio.NewScanner(bytes.NewReader(out))
		for scan.Scan() {
			fn := filepath.FromSlash(string(scan.Bytes()))
			f[fn] = Added
		}
	}

	gr.Lock()
	gr.files = f
	gr.Unlock()
	if onUpdated != nil {
		onUpdated(f)
	}
	gr.Lock()
	gr.gettingFiles = false
	gr.Unlock()
}

func (gr *GitRepo) charToStat(stat byte) FileStatus {
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

// StatusFast returns file status based on the cached file info,
// which might be slightly stale. Much faster than Status.
// Returns Untracked if no cached files.
func (gr *GitRepo) StatusFast(fname string) FileStatus {
	var ff Files
	gr.Lock()
	ff = gr.files
	gr.Unlock()
	if ff != nil {
		return ff.Status(gr, fname)
	}
	return Untracked
}

// Status returns status of given file; returns Untracked on any error.
func (gr *GitRepo) Status(fname string) (FileStatus, string) {
	out, err := gr.RunFromDir("git", "status", "--porcelain", relPath(gr, fname))
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
	return gr.charToStat(stat), ostr
}

// Add adds the file to the repo
func (gr *GitRepo) Add(fname string) error {
	fname = relPath(gr, fname)
	out, err := gr.RunFromDir("git", "add", fname)
	if err != nil {
		log.Println(string(out))
		return err
	}
	gr.Lock()
	if gr.files != nil {
		gr.files[fname] = Added
	}
	gr.Unlock()
	return nil
}

// Move moves updates the repo with the rename
func (gr *GitRepo) Move(oldpath, newpath string) error {
	out, err := gr.RunFromDir("git", "mv", relPath(gr, oldpath), relPath(gr, newpath))
	if err != nil {
		log.Println(string(out))
		return err
	}
	out, err = gr.RunFromDir("git", "add", relPath(gr, newpath))
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// Delete removes the file from the repo; uses "force" option to ensure deletion
func (gr *GitRepo) Delete(fname string) error {
	out, err := gr.RunFromDir("git", "rm", "-f", relPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		fmt.Printf("%s\n", out)
		return err
	}
	return nil
}

// Delete removes the file from the repo
func (gr *GitRepo) DeleteRemote(fname string) error {
	out, err := gr.RunFromDir("git", "rm", "--cached", relPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// CommitFile commits single file to repo staging
func (gr *GitRepo) CommitFile(fname string, message string) error {
	out, err := gr.RunFromDir("git", "commit", relPath(gr, fname), "-m", message)
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// RevertFile reverts a single file to last commit of master
func (gr *GitRepo) RevertFile(fname string) error {
	out, err := gr.RunFromDir("git", "checkout", relPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

// UpdateVersion sets the version of a package currently checked out via Git.
func (s *GitRepo) UpdateVersion(version string) error {
	out, err := s.RunFromDir("git", "switch", "--detach", version)
	if err != nil {
		return vcs.NewLocalError("Unable to update checked out version", err, string(out))
	}
	return nil
}

// FileContents returns the contents of given file, as a []byte array
// at given revision specifier. -1, -2 etc also work as universal
// ways of specifying prior revisions.
func (gr *GitRepo) FileContents(fname string, rev string) ([]byte, error) {
	if rev == "" {
		out, err := os.ReadFile(fname)
		if err != nil {
			log.Println(err.Error())
		}
		return out, err
	} else if rev[0] == '-' {
		rsp, err := strconv.Atoi(rev)
		if err == nil && rsp < 0 {
			rev = fmt.Sprintf("HEAD~%d:", -rsp)
		}
	} else {
		rev += ":"
	}
	fspec := rev + relPath(gr, fname)
	out, err := gr.RunFromDir("git", "show", fspec)
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
}

// fieldsThroughDelim gets the concatenated byte through to point where
// field ends with given delimiter, starting at given index
func fieldsThroughDelim(flds [][]byte, delim byte, idx int) (int, string) {
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
		ni, date := fieldsThroughDelim(flds, '}', 1)
		ni, author := fieldsThroughDelim(flds, '}', ni)
		ni, email := fieldsThroughDelim(flds, '}', ni)
		msg := string(bytes.Join(flds[ni:], []byte(" ")))
		lg.Add(rev, date, author, email, msg)
	}
	return lg, nil
}

// CommitDesc returns the full textual description of the given commit,
// if rev is empty, defaults to current HEAD, -1, -2 etc also work as universal
// ways of specifying prior revisions.
// Optionally includes diffs for the changes (otherwise just a list of files
// with modification status).
func (gr *GitRepo) CommitDesc(rev string, diffs bool) ([]byte, error) {
	if rev == "" {
		rev = "HEAD"
	} else if rev[0] == '-' {
		rsp, err := strconv.Atoi(rev)
		if err == nil && rsp < 0 {
			rev = fmt.Sprintf("HEAD~%d", -rsp)
		}
	}
	var out []byte
	var err error
	if diffs {
		out, err = gr.RunFromDir("git", "show", rev)
	} else {
		out, err = gr.RunFromDir("git", "show", "--name-status", rev)
	}
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
}

// FilesChanged returns the list of files changed and their statuses,
// between two revisions.
// If revA is empty, defaults to current HEAD; revB defaults to HEAD-1.
// -1, -2 etc also work as universal ways of specifying prior revisions.
// Optionally includes diffs for the changes.
func (gr *GitRepo) FilesChanged(revA, revB string, diffs bool) ([]byte, error) {
	if revA == "" {
		revA = "HEAD"
	} else if revA[0] == '-' {
		rsp, err := strconv.Atoi(revA)
		if err == nil && rsp < 0 {
			revA = fmt.Sprintf("HEAD~%d", -rsp)
		}
	}
	if revB != "" && revB[0] == '-' {
		rsp, err := strconv.Atoi(revB)
		if err == nil && rsp < 0 {
			revB = fmt.Sprintf("HEAD~%d", -rsp)
		}
	}
	var out []byte
	var err error
	if diffs {
		out, err = gr.RunFromDir("git", "diff", "-u", revA, revB)
	} else {
		if revB == "" {
			out, err = gr.RunFromDir("git", "diff", "--name-status", revA)
		} else {
			out, err = gr.RunFromDir("git", "diff", "--name-status", revA, revB)
		}
	}
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
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
