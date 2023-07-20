// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package icons

import (
	"embed"
	"strings"
)

//go:generate go run gen.go

// TheIcons contains all of the default embedded svg icons
//
//go:embed outlined/*.svg
var Icons embed.FS

// An Icon contains the path of an icon
type Icon string

// Fill returns the icon as a filled icon.
// It returns the icon unchanged if it is already filled.
func (i Icon) Fill() Icon {
	if strings.HasSuffix(string(i), "-fill") {
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
