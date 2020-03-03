// Copyright (c) 2018, The GoGi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

// Commit is one VCS commit entry, as returned in a Log
type Commit struct {
	Rev     string `desc:"revision number / hash code / unique id"`
	Date    string `desc:"date (author's time) when comitted"`
	Author  string `desc:"author's name"`
	Email   string `desc:"author's email"`
	Message string `desc:"message / subject line for commit"`
}

// Log is the listing of commits
type Log []*Commit

func (lg *Log) Add(rev, date, author, email, message string) *Commit {
	cm := &Commit{Rev: rev, Date: date, Author: author, Email: email, Message: message}
	*lg = append(*lg, cm)
	return cm
}
