// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package ki provides the top-level repository for GoKi Trees: Ki = Tree in Japanese, and
"Key" in English -- powerful tree structures supporting scenegraphs, programs,
parsing, etc.

The sub-packages contain all the relevant code:

* ki: is the main Ki interface and Node implementation thereof.

* kit: is a type registry that ki uses in various ways and provides
useful type-level properties that are used in the GoGi GUI.  It also
is a powerful 'kit for dealing with Go's reflect system.

* ints, floats, dirs, bitflag, atomctr, indent all provide basic
Go infrastructure that one could argue should have been in the
standard library, but isn't..
*/
package ki
