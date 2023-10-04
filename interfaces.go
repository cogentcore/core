// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package laser

// SetAnyer represents a type that can be set from any value.
// It is checked in [SetRobust].
type SetAnyer interface {
	SetAny(v any) error
}

// SetStringer represents a type that can be set from a string
// value. It is checked in [SetRobust].
type SetStringer interface {
	SetString(s string) error
}
