// Copyright (c) 2019, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Masterminds/vcs"
	"github.com/goki/ki/dirs"
)

type SvnRepo struct {
	vcs.SvnRepo
}

func (gr *SvnRepo) CharToStat(stat byte) FileStatus {
	switch stat {
	case 'M', 'R':
		return Modified
	case 'A':
		return Added
	case 'D', '!':
		return Deleted
	case 'C':
		return Conflicted
	case '?', 'I':
		return Untracked
	case '*':
		return Updated
	default:
		return Stored
	}
	return Untracked
}

func (gr *SvnRepo) Files() (Files, error) {
	f := make(Files, 1000)

	lpath := gr.LocalPath()
	allfs, err := dirs.AllFiles(lpath) // much faster than svn list --recursive
	for _, fn := range allfs {
		rpath, _ := filepath.Rel(lpath, fn)
		f[rpath] = Stored
	}

	out, err := gr.RunFromDir("svn", "status", "-u")
	if err != nil {
		return nil, err
	}
	scan := bufio.NewScanner(bytes.NewReader(out))
	for scan.Scan() {
		ln := string(scan.Bytes())
		flds := strings.Fields(ln)
		if len(flds) < 2 {
			continue // shouldn't happend
		}
		stat := flds[0][0]
		fn := flds[len(flds)-1]
		f[fn] = gr.CharToStat(stat)
	}
	return f, nil
}

// Status returns status of given file -- returns Untracked on any error
func (gr *SvnRepo) Status(fname string) (FileStatus, string) {
	out, err := gr.RunFromDir("svn", "status", RelPath(gr, fname))
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
func (gr *SvnRepo) Add(fname string) error {
	oscmd := exec.Command("svn", "add", RelPath(gr, fname))
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// Move moves updates the repo with the rename
func (gr *SvnRepo) Move(oldpath, newpath string) error {
	oscmd := exec.Command("svn", "mv", oldpath, newpath)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// Delete removes the file from the repo -- uses "force" option to ensure deletion
func (gr *SvnRepo) Delete(fname string) error {
	oscmd := exec.Command("svn", "rm", "-f", RelPath(gr, fname))
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// DeleteRemote removes the file from the repo, but keeps local copy
func (gr *SvnRepo) DeleteRemote(fname string) error {
	oscmd := exec.Command("svn", "delete", "--keep-local", RelPath(gr, fname))
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// CommitFile commits single file to repo staging
func (gr *SvnRepo) CommitFile(fname string, message string) error {
	oscmd := exec.Command("svn", "commit", RelPath(gr, fname), "-m", message)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// RevertFile reverts a single file to last commit of master
func (gr *SvnRepo) RevertFile(fname string) error {
	oscmd := exec.Command("svn", "revert", RelPath(gr, fname))
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// FileContents returns the contents of given file, as a []byte array
// at given revision specifier (if empty, defaults to current HEAD).
// -1, -2 etc also work as universal ways of specifying prior revisions.
func (gr *SvnRepo) FileContents(fname string, rev string) ([]byte, error) {
	if rev == "" {
		rev = "HEAD"
		// } else if rev[0] == '-' { // no support at this point..
		// 	rsp, err := strconv.Atoi(rev)
		// 	if err == nil && rsp < 0 {
		// 		rev = fmt.Sprintf("HEAD~%d:", rsp)
		// 	}
	}
	out, err := gr.RunFromDir("svn", "-r", "rev", "cat", RelPath(gr, fname))
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
}

// Log returns the log history of commits for given filename
// (or all files if empty).  If since is non-empty, it should be
// a date-like expression that the VCS will understand, such as
// 1/1/2020, yesterday, last year, etc
func (gr *SvnRepo) Log(fname string, since string) (Log, error) {
	// todo: parse -- requires parsing over multiple lines..
	return nil, nil
}

// CommitDesc returns the full textual description of the given commit,
// if rev is empty, defaults to current HEAD, -1, -2 etc also work as universal
// ways of specifying prior revisions.
// Optionally includes diffs for the changes (otherwise just a list of files
// with modification status).
func (gr *SvnRepo) CommitDesc(rev string, diffs bool) ([]byte, error) {
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
	return out, err
}

// Blame returns an annotated report about the file, showing which revision last
// modified each line.
func (gr *SvnRepo) Blame(fname string) ([]byte, error) {
	out, err := gr.RunFromDir("svn", "blame", fname)
	if err != nil {
		log.Println(string(out))
		return nil, err
	}
	return out, nil
}
