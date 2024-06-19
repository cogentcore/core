// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package slicesx

// Tiered represents a tiered set of slices of the same type.
// For example, this is frequently used to represent stacks of
// functions that can be run at normal priority or before or
// after such.
type Tiered[E any] struct {

	// First contains the elements that will be used first, before
	// those in [Tiered.Normal] and [Tiered.Final].
	First []E

	// Normal contains the elements that will be used at the normal
	// time, after those in [Tiered.First] and before those in [Tiered.Final].
	Normal []E

	// Final contains the elements that will be used last, after
	// those in [Tiered.First] and [Tiered.Normal].
	Final []E
}

// Do calls the given function for each element in the tiered set of slices,
// going through first, then normal, then final.
func (t *Tiered[E]) Do(f func(E)) {
	for _, e := range t.First {
		f(e)
	}
	for _, e := range t.Normal {
		f(e)
	}
	for _, e := range t.Final {
		f(e)
	}
}
