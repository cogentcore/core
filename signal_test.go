// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"reflect"
	"testing"
	// "time"
)

type TestNode struct {
	Node
	sig1 Signal
	sig2 Signal
}

var KtTestNode = KiTypes.AddType(&TestNode{})

func TestSignalConnect(t *testing.T) {
	parent := TestNode{}
	parent.SetName("par1")
	parent.SetRoot(&parent)
	child1 := parent.AddNewChildNamed("child1")
	child2 := parent.AddNewChildNamed("child2")

	res := make([]string, 0, 10)
	parent.sig1.Connect(child1, func(receiver, sender Ki, sig SignalType, data interface{}) {
		res = append(res, fmt.Sprintf("recv: %v, sender: %v sig: %v data: %v",
			receiver.KiName(), sender.KiName(), sig, data))
	})
	parent.sig1.Connect(child2, func(receiver, sender Ki, sig SignalType, data interface{}) {
		res = append(res, fmt.Sprintf("recv: %v, sender: %v sig: %v data: %v",
			receiver.KiName(), sender.KiName(), sig, data))
	})
	parent.sig1.Emit(&parent, NilSignal, 1234)

	// fmt.Printf("res: %v\n", res)
	trg := []string{"recv: child1, sender: par1 sig: NilSignal data: 1234",
		"recv: child2, sender: par1 sig: NilSignal data: 1234"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("Add child sigs error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	// time.Sleep(time.Second * 2)
}
