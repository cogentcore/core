// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package num

// Abs returns the absolute value of the given value.
func Abs[T Signed | Float](x T) T {
	if x < 0 {
		return -x
	}
	return x
}
