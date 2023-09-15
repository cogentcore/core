// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

//go:generate enumgen

// TestEnum is an enum type for testing
type TestEnum int32 //enums:enum

const (
	TestValue1 TestEnum = iota
	TestValue2
)
