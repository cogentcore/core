// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fmath provides the float32 math functions used by math32. Natively,
// they are those of github.com/chewxy/math32, which are implemented in
// float32 arithmetic. With goscript (the js platform), they use the float64
// math package, which goscript implements with the JavaScript Math functions,
// since the float32 algorithms are much slower there.
package fmath
