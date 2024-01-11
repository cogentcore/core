// Copyright (c) 2018, The GoGi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vci

// Commit is one VCS commit entry, as returned in a Log
type Commit struct {

	// revision number / hash code / unique id
	Rev string

	// date (author's time) when comitted
	Date string

	// author's name
	Author string

	// author's email
	Email string

	// message / subject line for commit
	Message string `width:"100"`
}

// Log is the listing of commits
type Log []*Commit

func (lg *Log) Add(rev, date, author, email, message string) *Commit {
	cm := &Commit{Rev: rev, Date: date, Author: author, Email: email, Message: message}
	*lg = append(*lg, cm)
	return cm
}
