// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package wintoast shows and schedules the toast notifications of Windows,
// with the WinRT API of Windows.UI.Notifications, through its COM
// interfaces, without cgo. It is only implemented on Windows.
package wintoast

import "time"

// Toast is a toast notification.
type Toast struct {

	// ID identifies the toast, which replaces a toast with the same ID.
	ID string

	// Title is the title of the toast, in bold.
	Title string

	// Body is the text of the toast, below the title.
	Body string

	// Time is when a scheduled toast is shown.
	Time time.Time
}
