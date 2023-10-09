// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/goosi/events/key"
	"goki.dev/icons"
)

// TODO(kai): remove ActOpts

// ActOpts provides named and partial parameters for the AddButton method
type ActOpts struct {
	Name        string
	Label       string
	Icon        icons.Icon
	Tooltip     string
	Shortcut    key.Chord
	ShortcutKey KeyFuns
	Data        any
	UpdateFunc  func(bt *Button)
}
