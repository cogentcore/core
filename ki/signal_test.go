// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/goki/ki/kit"
)

type TestNode struct {
	Node
	sig1 Signal
	sig2 Signal
}

var KiTTestNode = kit.Types.AddType(&TestNode{}, nil)

func TestSignalConnect(t *testing.T) {
	parent := TestNode{}
	parent.InitName(&parent, "par1")
	child1 := parent.AddNewChild(nil, "child1")
	// child2 := parent.AddNewChild(nil, "child2")

	// note: now that signal is a map, cannot test reliably due to ordering
	res := make([]string, 0, 10)
	parent.sig1.Connect(child1, func(receiver, sender Ki, sig int64, data interface{}) {
		res = append(res, fmt.Sprintf("recv: %v, sender: %v sig: %v data: %v",
			receiver.Name(), sender.Name(), NodeSignals(sig), data))
	})
	// parent.sig1.Connect(child2, func(receiver, sender Ki, sig int64, data interface{}) {
	// 	res = append(res, fmt.Sprintf("recv: %v, sender: %v sig: %v data: %v",
	// 		receiver.Name(), sender.Name(), NodeSignals(sig), data))
	// })

	parent.sig1.Emit(&parent, int64(NodeSignalNil), 1234)

	// fmt.Printf("res: %v\n", res)
	trg := []string{"recv: child1, sender: par1 sig: NodeSignalNil data: 1234"}
	if !reflect.DeepEqual(res, trg) {
		t.Errorf("Add child sigs error -- results: %v != target: %v\n", res, trg)
	}
	res = res[:0]

	// time.Sleep(time.Second * 2)
}

func TestSignalNameToInt(t *testing.T) {
	for i := NodeSignalNil; i < NodeSignalsN; i++ {
		st := NodeSignals(i)
		str := st.String()
		stc := NodeSignalNil
		err := stc.FromString(str)
		if err != nil {
			t.Error(err)
		}
		stnm := stc.String()
		if stnm != str {
			t.Errorf("could not convert from signal type name %v -- got: %v -- maybe need to run go generate?", str, stnm)
		}
	}

	str := "NodeSignalUpdated"
	stc := NodeSignalNil
	stc.FromString(str)
	if stc.String() != str {
		t.Errorf("could not convert from signal type name %v -- got: %v -- maybe need to run go generate?", str, stc.String())
	}
}
