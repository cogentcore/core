// Copyright (c) 2019, The Gide Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

import (
	"bufio"
	"bytes"
	"log"
	"os/exec"
	"strings"

	"github.com/Masterminds/vcs"
	"github.com/goki/ki/dirs"
)

type SvnRepo struct {
	vcs.SvnRepo
}

// M            11491   COPYING
// ?                    build_dbg

func (gr *SvnRepo) Files() (Files, error) {
	f := make(Files, 1000)

	allfs, err := dirs.AllFiles(gr.LocalPath()) // much faster than svn list --recursive
	for _, fn := range allfs {
		f[fn] = Stored
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
		switch stat {
		case 'M', 'R':
			f[fn] = Modified
		case 'A':
			f[fn] = Added
		case 'D', '!':
			f[fn] = Deleted
		case 'C':
			f[fn] = Conflicted
		case '?', 'I':
			f[fn] = Untracked
		case '*':
			f[fn] = Updated
		default:
			f[fn] = Stored
		}
	}
	return f, nil
}

// Add adds the file to the repo
func (gr *SvnRepo) Add(filename string) error {
	oscmd := exec.Command("svn", "add", filename)
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
func (gr *SvnRepo) Delete(filename string) error {
	oscmd := exec.Command("svn", "rm", filename)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// Delete removes the file from the repo
func (gr *SvnRepo) DeleteKeepLocal(filename string) error {
	oscmd := exec.Command("svn", "delete", "--keep-local", filename)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// CommitFile commits single file to repo staging
func (gr *SvnRepo) CommitFile(filename string, message string) error {
	oscmd := exec.Command("svn", "commit", filename, "-m", message)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}

// RevertFile reverts a single file to last commit of master
func (gr *SvnRepo) RevertFile(filename string) error {
	oscmd := exec.Command("svn", "revert", filename)
	stdoutStderr, err := oscmd.CombinedOutput()
	if err != nil {
		log.Println(string(stdoutStderr))
		return err
	}
	return nil
}
