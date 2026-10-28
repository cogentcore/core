// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package noref implements [fmt.Stringer] without referencing it,
// to test that typegen finds the interfaces in the dependencies.
package noref

import "fmt"

// Pet is a pet with a name.
type Pet struct {

	// Name is the name of the pet.
	Name string
}

func (p *Pet) String() string { return fmt.Sprint("pet ", p.Name) }
