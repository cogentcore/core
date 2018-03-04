// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

type TestNode struct {
	Node
	sig1 Signal
	sig2 Signal
}

var KtTestNode = KiTypes.AddType(&TestNode{})

func Slot1(receiver, sender Ki, sig SignalType, data interface{}) {
	fmt.Printf("Slot1 called on recv: %v, from sender: %v sig: %v with data: %v",
		receiver.KiName(), sender.KiName(), sig, data)
}

func Slot2(receiver, sender Ki, sig SignalType, data interface{}) {
	fmt.Printf("Slot1 called on recv: %v, from sender: %v sig: %v with data: %v",
		receiver.KiName(), sender.KiName(), sig, data)
}

func TestSignalConnect(t *testing.T) {
	parent := TestNode{}
	parent.SetName("par1")
	parent.SetChildType(reflect.TypeOf(parent))
	child1, _ := parent.AddNewChildNamed("child1")
	child2, _ := parent.AddNewChildNamed("child2")

	parent.sig1.Connect(child1, Slot1)
	parent.sig1.Connect(child2, Slot2)

	parent.sig1.Emit(&parent, NoSignal, 1234)
	time.Sleep(time.Second * 2)
}
