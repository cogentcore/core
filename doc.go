// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// mat32 is our version of the G3N math32 32-bit, 3D rendering-based
// math library.
//
// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// A major modification from the G3N version is to use value-based
// logic instead of pointer-based self-modification in the Vector classes,
// As used in e.g., go-gl/mathgl (which uses slice-based vectors instead
// of the more intuitive and likely more efficient struct-based ones here).
//
// The pointer-based approach was retained for the Matrix classes,
// which are larger and that is likely more performant.
//
// Many names were shortened, consistent with more idiomatic Go naming.
// and again with the go-gl/mathgl library
package mat32
