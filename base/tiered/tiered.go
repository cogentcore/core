// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tiered provides a type for a tiered set of objects.
package tiered

// Tiered represents a tiered set of objects of the same type.
// For example, this is frequently used to represent slices of
// functions that can be run at the first, normal, or final time.
type Tiered[T any] struct {

	// First is the object that will be used first,
	// before [Tiered.Normal] and [Tiered.Final].
	First T

	// Normal is the object that will be used at the normal
	// time, after [Tiered.First] and before [Tiered.Final].
	Normal T

	// Final is the object that will be used last,
	// after [Tiered.First] and [Tiered.Normal].
	Final T
}

// Do calls the given function for each tier,
// going through first, then normal, then final.
func (t *Tiered[T]) Do(f func(*T)) {
	f(&t.First)
	f(&t.Normal)
	f(&t.Final)
}

// DoWith calls the given function with each tier of this tiered
// set and the other given tiered set, going through first, then
// normal, then final.
func (t *Tiered[T]) DoWith(other *Tiered[T], f func(*T, *T)) {
	f(&t.First, &other.First)
	f(&t.Normal, &other.Normal)
	f(&t.Final, &other.Final)
}
