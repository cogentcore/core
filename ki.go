// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Ki is the base element of GoKi Trees
// Ki = Tree in Japanese, and "Key" in English
package goki

import (
)

// General Ki interface for GoKi Tree elements -- any Ki element must implement this
type Ki interface {
	KiParent() Ki
	KiChildren() []Ki

	// These allow generic GUI / Text / etc representation of Trees
	// The user-defined name of the object, for finding elements, generating paths, io, etc
	KiName() string
	// A name that is guaranteed to be unique within the children of this node -- important for generating unique paths
	KiUniqueName() string
	// Properties tell GUI or other frameworks operating on Trees about special features of each node -- todo: this should be a map!
	KiProperties() []string
}

// see node for struct implementing this interface

