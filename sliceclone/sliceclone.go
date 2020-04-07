// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package sliceclone provides those basic slice cloning methods that I finally got tired of
rewriting all the time.
*/
package sliceclone

// String returns a cloned copy of the given string slice -- returns nil
// if slice has zero length
func String(sl []string) []string {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]string, sz)
	copy(cp, sl)
	return cp
}

// Byte returns a cloned copy of the given byte slice -- returns nil
// if slice has zero length
func Byte(sl []byte) []byte {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]byte, sz)
	copy(cp, sl)
	return cp
}

// Rune returns a cloned copy of the given rune slice -- returns nil
// if slice has zero length
func Rune(sl []rune) []rune {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]rune, sz)
	copy(cp, sl)
	return cp
}

// Bool returns a cloned copy of the given bool slice -- returns nil
// if slice has zero length
func Bool(sl []bool) []bool {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]bool, sz)
	copy(cp, sl)
	return cp
}

// Int returns a cloned copy of the given int slice -- returns nil
// if slice has zero length
func Int(sl []int) []int {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]int, sz)
	copy(cp, sl)
	return cp
}

// Int32 returns a cloned copy of the given int32 slice -- returns nil
// if slice has zero length
func Int32(sl []int32) []int32 {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]int32, sz)
	copy(cp, sl)
	return cp
}

// Int64 returns a cloned copy of the given int64 slice -- returns nil
// if slice has zero length
func Int64(sl []int64) []int64 {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]int64, sz)
	copy(cp, sl)
	return cp
}

// Float64 returns a cloned copy of the given float64 slice -- returns nil
// if slice has zero length
func Float64(sl []float64) []float64 {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]float64, sz)
	copy(cp, sl)
	return cp
}

// Float32 returns a cloned copy of the given float32 slice -- returns nil
// if slice has zero length
func Float32(sl []float32) []float32 {
	sz := len(sl)
	if sz == 0 {
		return nil
	}
	cp := make([]float32, sz)
	copy(cp, sl)
	return cp
}
