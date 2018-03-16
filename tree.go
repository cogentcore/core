// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/ki"
	// "log"
	// "reflect"
)

////////////////////////////////////////////////////////////////////////////////////////
//  Node Widget -- represents one node in the tree -- fully recursive -- creates
//  sub-nodes etc

// NodeWidget
type NodeWidget struct {
	NodePtr   ki.Ptr `desc:"Ki Node that this widget represents"`
	Collapsed []bool `desc:"collapsed state of each child in this node"`
}
