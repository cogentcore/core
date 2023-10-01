// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
	"image"
)

// CustomEvent is a user-specified event that can be sent and received
// as needed, and contains a Data field for arbitrary data, and
// optional position and focus parameters
type CustomEvent struct {
	Base

	// set to true if position is available
	PosAvail bool `desc:"set to true if position is available"`
}

func (ce CustomEvent) String() string {
	return fmt.Sprintf("%v{Data: %v, Time: %v}", ce.Type(), ce.Data, ce.Time())
}

func (ce CustomEvent) HasPos() bool {
	return ce.PosAvail
}

func (ce CustomEvent) Pos() image.Point {
	return ce.Pos()
}
