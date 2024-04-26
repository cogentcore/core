// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package metric provides various similarity / distance metrics for comparing
floating-point vectors.
All functions have 32 and 64 bit variants, and skip NaN's (often used for missing)
and will panic if the lengths of the two slices are unequal (no error return).

The signatures of all such metric functions are identical, captured as types:
metric.Func32 and metric.Func64 so that other functions that use a metric
can take a pointer to any such function.
*/
package metric
