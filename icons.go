// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package icons provides Go constant names for Material Design Symbols as SVG files.
package icons

import (
	"embed"
	"io/fs"
	"strings"

	_ "github.com/iancoleman/strcase" // needed so that it gets included in the mod (the generator uses it)
	"goki.dev/glop/dirs"
)

//go:generate go run gen.go

// Icons contains all of the default embedded svg icons
//
//go:embed svg/*.svg
var Icons embed.FS

const (
	// None is an icon that indicates to not use an icon.
	// It completely prevents the rendering of an icon,
	// whereas [Blank] renders a blank icon.
	None Icon = "none"

	// Blank is a blank icon that can be used as a
	// placeholder when no other icon is appropriate.
	// It still renders an icon, just a blank one,
	// whereas [None] indicates to not render one at all.
	Blank Icon = "blank"
)

// Icon contains the name of an icon
type Icon string

// Fill returns the icon as a filled icon.
// It returns the icon unchanged if it is already filled.
func (i Icon) Fill() Icon {
	if i.IsFilled() {
		return i
	}
	return i + "-fill"
}

// Unfill returns the icon as an unfilled icon.
// It returns the icon unchanged if it is already unfilled.
// Icons are unfilled by default, so you only
// need to call this to reverse a prior [Icon.Fill] call
func (i Icon) Unfill() Icon {
	return Icon(strings.TrimSuffix(string(i), "-fill"))
}

// IsFilled returns whether the icon
// is a filled icon.
func (i Icon) IsFilled() bool {
	return strings.HasSuffix(string(i), "-fill")
}

// IsNil returns whether the icon name is empty,
// [None], or "nil"; those indicate not to use an icon.
func (i Icon) IsNil() bool {
	return i == "" || i == None || i == "nil"
}

// Filename returns the filename of the icon in [Icons]
func (i Icon) Filename() string {
	// fs always uses forward slashes
	return "svg/" + string(i) + ".svg"
}

// IsValid returns whether the icon name corresponds to
// a valid existing icon.
func (i Icon) IsValid() bool {
	if i.IsNil() {
		return false
	}
	ex, _ := dirs.FileExistsFS(Icons, i.Filename())
	return ex
}

// All returns a list of all the icons
func All() []string {
	files, err := fs.ReadDir(Icons, "svg")
	if err != nil {
		return nil
	}
	fns := make([]string, 0, len(files))
	for _, fi := range files {
		fns = append(fns, fi.Name())
	}
	return fns
}
