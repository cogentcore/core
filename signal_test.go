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

var KtTestNode = Types.AddType(&TestNode{}, nil)

func TestSignalConnect(t *testing.T) {
	parent := TestNode{}
	parent.SetThisName(&parent, "par1")
	child1 := parent.AddNewChildNamed(nil, "child1")
	child2 := parent.AddNewChildNamed(nil, "child2")

	res := make([]string, 0, 10)
	parent.sig1.Connect(child1, func(receiver, sender Ki, sig int64, data interface{}) {
		res = append(res, fmt.Sprintf("recv: %v, sender: %v sig: %v data: %v",
			receiver.KiName(), sender.KiName(), SignalType(sig), data))
	})
	parent.sig1.Connect(child2, func(receiver, sender Ki, sig int64, data interface{}) {
		res = append(res, fmt.Sprintf("recv: %v, sender: %v sig: %v data: %v",
			receiver.KiName(), sender.KiName(), SignalType(sig), data))
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

func TestSignalNameToInt(t *testing.T) {
	for i := NilSignal; i < SignalTypeN; i++ {
		st := SignalType(i)
		str := st.String()
		stc, err := StringToSignalType(str)
		if err != nil {
			t.Error(err)
		}
		stnm := stc.String()
		if stnm != str {
			t.Errorf("could not convert from signal type name %v -- got: %v -- maybe need to run go generate?", str, stnm)
		}
	}

	str := "SignalFieldUpdated"
	st, _ := StringToSignalType(str)
	if st.String() != str {
		t.Errorf("could not convert from signal type name %v -- got: %v -- maybe need to run go generate?", str, st.String())
	}
}
