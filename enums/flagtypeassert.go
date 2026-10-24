// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !goscript

package enums

// FlagTypeAssert is whether the generated HasFlag and SetFlag methods of
// bit flag types use a type assertion to get the bit index of a flag of
// their own type, instead of calling its Int64 method through the [BitFlag]
// interface. It is only true in JavaScript compiled by goscript, where
// Int64 values are slow BigInts; natively, the type assertion would only
// prevent HasFlag from being inlined.
const FlagTypeAssert = false
