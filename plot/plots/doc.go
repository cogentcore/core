// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Adapted from github.com/gonum/plot:
// Copyright ©2015 The Gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plots defines a variety of standard Plotters for the
// plot package.
//
// Plotters use the primitives provided by the plot package to draw to
// the data area of a plot. This package provides some standard data
// styles such as lines, scatter plots, box plots, labels, and more.
//
// Unlike the gonum/plot package, NaN values are treated as missing
// data points, and are just skipped over.
//
// New* functions return an error if the data contains Inf or is
// empty. Some of the New* functions return other plotter-specific errors
// too.
package plots
