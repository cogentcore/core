// Copyright (c) 2021 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package osevent defines operating system level events, not associated
// with a particular window.
package osevent

//go:generate enumgen

import (
	"fmt"

	"goki.dev/goosi"
)

// osevent.Event reports an OS level event
type Event struct {
	goosi.EventBase

	// Action taken on the osevent -- what has changed.
	// Osevent state fields have current values.
	Action Actions
}

// Actions is the action taken on the osevent by the user.
type Actions int32 //enums:enum

const (
	// OpenFiles means the user indicated that the app should open file(s) stored in Files
	OpenFiles Actions = iota
)

func NewEvent() *Event {
	ev := &Event{}
	ev.Typ = goosi.OSEvent
	return ev
}

func (ev *Event) OnWinFocus() bool { // os events generally not focus-specific
	return false
}

func (ev *Event) OnFocus() bool {
	return false
}

func (ev *Event) String() string {
	return fmt.Sprintf("Type: %v Action: %v  Time: %v", ev.Type(), ev.Action, ev.Time())
}

// osevent.OpenFilesEvent is for OS open files action to open given files
type OpenFilesEvent struct {
	Event

	// Files are a list of files to open
	Files []string
}

func NewOpenFilesEvent(files []string) *OpenFilesEvent {
	ev := &OpenFilesEvent{}
	ev.Typ = goosi.OSOpenFilesEvent
	ev.Action = OpenFiles
	ev.Files = files
	return ev
}
