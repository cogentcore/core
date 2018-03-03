// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"errors"
	"fmt"
	"log"
	"reflect"
)

// based on: github.com/tucnak/meta/

// Receiver function type on receiver node -- gets the sending node and arbitrary additional data
type RecvFunc func(receiver, sender *Node, data interface{})

// Signal -- add one of these for each signal a node can emit
type Signal struct {
	Cons []Connection
}

type Connection struct {
	// node that will receive the signal
	Recv *Node
	// function on the receiver node that will receive the signal
	Func RecvFunc
}

// Connect attaches a new receiver to the signal -- error if not ok
func (sig *Signal) Connect(recv *Node, recvfun RecvFunc) error {
	if recv == nil {
		return errors.New("ki Signal Connect: no recv node provided")
	}
	if recvfun == nil {
		return errors.New("ki Signal Connect: no recv func provided")
	}

	con := Connection{
		Recv: recv,
		Func: recvfun,
	}
	sig.Cons = append(sig.Cons, con)

	log.Printf("added connection to recv %v fun %v", recv.Name, reflect.ValueOf(recvfun))

	return nil
}

// Disconnect receiver and signal
func (sig *Signal) Disconnect(recv *Node, recvfun RecvFunc) error {
	if recv == nil {
		return errors.New("ki Signal Disconnect: no recv node provided")
	}
	if recvfun == nil {
		return errors.New("ki Signal Disconnect: no recv func provided")
	}

	for i, con := range sig.Cons {
		if con.Recv == recv /* && con.Func == recvfun */ {
			// this copy makes sure there are no memory leaks
			copy(sig.Cons[i:], sig.Cons[i+1:])
			sig.Cons = sig.Cons[:len(sig.Cons)-1]
			return nil
		}
	}
	return errors.New(fmt.Sprintf("ki Signal Disconnect: connection not found for node: %v func: %v", recv.Name, reflect.ValueOf(recvfun)))
}

// Emit executes all the connected slots with data given.
func (sig *Signal) Emit(sender *Node, data interface{}) {
	for _, con := range sig.Cons {
		go con.Func(con.Recv, sender, data)
	}
}

// Emit given signal on node
func (n *Node) Emit(sig *Signal, data interface{}) {
	sig.Emit(n, data)
}
