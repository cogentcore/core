// Copyright (c) 2019, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"path/filepath"
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

// Delete removes the file from the repo
func (gr *SvnRepo) Delete(fname string) error {
	oscmd := exec.Command("svn", "rm", RelPath(gr, fname))
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
