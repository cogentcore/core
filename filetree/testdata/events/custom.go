// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"fmt"
)

// CustomEvent is a user-specified event that can be sent and received
// as needed, and contains a Data field for arbitrary data, and
// optional position and focus parameters
type CustomEvent struct {
	Base

	// set to true if position is available
	PosAvail bool
}

func (ce CustomEvent) String() string {
	return fmt.Sprintf("%v{Data: %v, Time: %v}", ce.Type(), ce.Data, ce.Time())
}

func (ce CustomEvent) HasPos() bool {
	return ce.PosAvail
}
