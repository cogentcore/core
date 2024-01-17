// Copyright (c) 2021 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
)

// OSEvent reports an OS level event
type OSEvent struct {
	Base
}

func NewOSEvent(typ Types) *OSEvent {
	ev := &OSEvent{}
	ev.Typ = typ
	return ev
}

func (ev *OSEvent) String() string {
	return fmt.Sprintf("%v{Time: %v}", ev.Type(), ev.Time().Format("04:05"))
}

// osevent.OpenFilesEvent is for OS open files action to open given files
type OSFiles struct {
	OSEvent

	// Files are a list of files to open
	Files []string
}

func NewOSFiles(typ Types, files []string) *OSFiles {
	ev := &OSFiles{}
	ev.Typ = typ
	ev.Files = files
	return ev
}
