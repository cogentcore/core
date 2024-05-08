// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package timer provides a simple wall-clock duration timer based on standard
// time.  Accumulates total and average over multiple Start / Stop intervals.
package timer

//go:generate core generate -add-types

import "time"

// Time manages the timer accumulated time and count
type Time struct {

	// the most recent starting time
	St time.Time

	// the total accumulated time
	Total time.Duration

	// the number of start/stops
	N int
}

// Reset resets the overall accumulated Total and N counters and start time to zero
func (t *Time) Reset() {
	t.St = time.Time{}
	t.Total = 0
	t.N = 0
}

// Start starts the timer
func (t *Time) Start() {
	t.St = time.Now()
}

// ResetStart reset then start the timer
func (t *Time) ResetStart() {
	t.Reset()
	t.Start()
}

// Stop stops the timer and accumulates the latest start - stop interval, and also returns it
func (t *Time) Stop() time.Duration {
	if t.St.IsZero() {
		t.Total = 0
		t.N = 0
		return 0
	}
	iv := time.Since(t.St)
	t.Total += iv
	t.N++
	return iv
}

// Avg returns the average start / stop interval (assumes each was measuring the same thing).
func (t *Time) Avg() time.Duration {
	if t.N == 0 {
		return 0
	}
	return t.Total / time.Duration(t.N)
}
