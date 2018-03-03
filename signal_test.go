// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
/*	"fmt"
	"testing"
*/
)

/*
type TestNode struct {
	node Node
	sig1 Signal
	sig2 Signal
}

func (n *TestNode) Node() *Node {
	return &n.node
}

func (n *TestNode) Slot1(sender *Node, data interface{}) {
	fmt.Printf("Slot1 called from sender: %v with data: $v", sender.Name, data)
}

func (n *TestNode) Slot2(sender *Node, data interface{}) {
	fmt.Printf("Slot2 called from sender: %v with data: $v", sender.Name, data)
}

func TestSignalConnect(t *testing.T) {
	sender1 := TestNode{node: Node{Name: "sender1"}}
	recv1 := TestNode{node: Node{Name: "recv1"}}
	recv2 := TestNode{node: Node{Name: "recv2"}}

	sender1.node.AddChildren(&recv1, &recv2)

	// sender1.sig1.Connect(recv1, TestNode.Slot1)
	// sender1.sig1.Connect(recv2, TestNode.Slot2)

	// sender1.Emit(sender.sig1, 1234)
}
*/
