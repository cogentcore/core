// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tmath

import "cogentcore.org/core/tensor"

//go:generate core generate

// Func1in1out is a function that has 1 tensor input and 1 output.
type Func1in1out func(in, out *tensor.Indexed)

// Func2in1out is a function that has 2 tensor input and 1 output.
type Func2in1out func(in, out *tensor.Indexed)

// Funcs1in1out is a registry of named math functions that
// take one input and one output tensor.
var Funcs1in1out map[string]Func1in1out

// Funcs2in1out is a registry of named math functions that
// take two inputs and one output tensor.
var Funcs2in1out map[string]Func2in1out
