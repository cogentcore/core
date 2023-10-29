// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events/key"
)

// TODO(kai/menu): remove these old functions once we fully switch over to new structure

// ActionUpdateFunc is a function that updates method enabled / disabled status.
// The first argument is the object on which the method is defined (receiver).
type ActionUpdateFunc func(it any, act *gi.Button)

// SubMenuFunc is a function that returns a string slice of submenu items
// used in MethodView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubMenuFunc func(it any, vp *gi.Scene) []string

// SubSubMenuFunc is a function that returns a slice of string slices
// to create submenu items each having their own submenus.
// used in MethodView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubSubMenuFunc func(it any, vp *gi.Scene) [][]string

// ShortcutFunc is a function that returns a key.Chord string for a shortcut
// used in MethodView shortcut-func option
// first argument is the object on which the method is defined (receiver)
type ShortcutFunc func(it any, act *gi.Button) key.Chord

// LabelFunc is a function that returns a string to set a label
// first argument is the object on which the method is defined (receiver)
type LabelFunc func(it any, act *gi.Button) string
