// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/rcoreilly/goki/ki/kit"
)

// Implements general signal passing between Ki objects, like Qt's Signal /
// Slot system started from: github.com/tucnak/meta/
//
// todo: Once I learn more about channels and concurrency, could add a channel
// as an alternative method of sending signals too, perhaps
//
// A receivier has to connect to a given signal on a sender to receive those
// signals, when the signal is emitted.  To make more efficient use of signal
// connections, we also support a signal type int64 that the receiver can
// decode depending on the type of signal that it is receiving -- completely
// up to the semantics of that particular signal.

// NodeSignals are signals that a Ki node sends about updates to the tree
// structure using the NodeSignal (convert sig int64 to NodeSignals to get the
// stringer name).
type NodeSignals int64

// Standard signal types sent by ki.Node on its NodeSig for tree state changes
const (
	// no signal
	NodeSignalNil NodeSignals = iota
	// node was updated -- see the node Flags field for the specific changes
	// that occured during the update -- data is those flags too
	NodeSignalUpdated
	// node is being deleted from its parent children list -- not blocked by updating status and delivered immediately
	NodeSignalDeleting
	// node is about to be destroyed -- second pass after removal from parent -- all of its children will be destroyed too -- not blocked by updating status and delivered immediately
	NodeSignalDestroying
	// number of signal type consts -- add this to any other signal types passed
	NodeSignalsN
)

//go:generate stringer -type=NodeSignals

// set this to true to automatically print out a trace of the signals as they are sent
var NodeSignalTrace bool = false

// set this to a string to receive trace in a string that can be compared for testing
// otherwise just goes to stdout
var NodeSignalTraceString *string

// Receiver function type on receiver node -- gets the sending node and arbitrary additional data
type RecvFunc func(receiver, sender Ki, sig int64, data interface{})

// use this to encode a custom signal type to be used over the ki.Node.NodeSig and not be confused with basic signals defined above
func SendCustomNodeSignal(sig int64) int64 {
	return sig + int64(NodeSignalsN)
}

// use this to encode a custom signal type to be used over the ki.Node.NodeSig and not be confused with basic signals defined above
func RecvCustomNodeSignal(sig int64) int64 {
	return sig - int64(NodeSignalsN)
}

// Signal -- add one of these for each signal a node can emit
type Signal struct {
	Cons []Connection
}

var KiT_Signal = kit.Types.AddType(&Signal{}, nil)

// Connection represents one connection between a signal and a receiving Ki and function to call
type Connection struct {
	// node that will receive the signal
	Recv Ki
	// function on the receiver node that will receive the signal
	Func RecvFunc
	// todo: path to Recv node (PathUnique), used for copying / moving nodes -- not copying yet
	// RecvPath string
}

// send the signal over this connection
func (con *Connection) SendSig(sender Ki, sig int64, data interface{}) {
	con.Func(con.Recv, sender, sig, data)
}

// Connect attaches a new receiver to the signal -- checks to make sure connection does not already exist -- error if not ok
func (sig *Signal) Connect(recv Ki, fun RecvFunc) error {
	if recv == nil {
		return errors.New("ki Signal Connect: no recv node provided")
	}
	if fun == nil {
		return errors.New("ki Signal Connect: no recv func provided")
	}

	if sig.FindConnectionIndex(recv, fun) >= 0 {
		return nil
	}

	con := Connection{recv, fun}
	sig.Cons = append(sig.Cons, con)

	// fmt.Printf("added connection to recv %v fun %v", recv.KiName(), reflect.ValueOf(fun))

	return nil
}

// Find any existing signal connection for given recv and fun
func (sig *Signal) FindConnectionIndex(recv Ki, fun RecvFunc) int {
	rfref := reflect.ValueOf(fun).Pointer()
	for i, con := range sig.Cons {
		if con.Recv == recv && rfref == reflect.ValueOf(con.Func).Pointer() {
			return i
		}
	}
	return -1
}

// Disconnect all connections for receiver and/or function if they exist in our list -- can pass nil for either (or both) to match only on one or the other -- both nil means disconnect from all, but more efficient to use DisconnectAll
func (sig *Signal) Disconnect(recv Ki, fun RecvFunc) bool {
	rfref := reflect.ValueOf(fun).Pointer()
	sz := len(sig.Cons)
	got := false
	for i := sz - 1; i >= 0; i-- {
		con := sig.Cons[i]
		if recv != nil && con.Recv != recv {
			continue
		}
		if fun != nil && rfref != reflect.ValueOf(con.Func).Pointer() {
			continue
		}
		// this copy makes sure there are no memory leaks
		copy(sig.Cons[i:], sig.Cons[i+1:])
		sig.Cons = sig.Cons[:len(sig.Cons)-1]
		got = true
	}
	return got
}

// Disconnect all connections
func (sig *Signal) DisconnectAll() {
	sig.Cons = sig.Cons[:0]
}

// record a trace of signal being emitted
func (s *Signal) EmitTrace(sender Ki, sig int64, data interface{}) {
	if NodeSignalTraceString != nil {
		*NodeSignalTraceString += fmt.Sprintf("ki.Signal Emit from: %v sig: %v data: %v\n", sender.Name(), NodeSignals(sig), data)
	} else {
		fmt.Printf("ki.Signal Emit from: %v sig: %v data: %v\n", sender.PathUnique(), NodeSignals(sig), data)
	}
}

// Emit sends the signal across all the connections to the receivers -- sequential
func (s *Signal) Emit(sender Ki, sig int64, data interface{}) {
	if sender == nil || sender.IsDestroyed() { // dead nodes don't talk..
		return
	}
	if NodeSignalTrace {
		s.EmitTrace(sender, sig, data)
	}
	deleted := 0 // using this construct from https://stackoverflow.com/questions/20545743/delete-entries-from-a-slice-while-iterating-over-it-in-go
	for i := range s.Cons {
		j := i - deleted
		con := s.Cons[j]
		if con.Recv.IsDestroyed() {
			fmt.Printf("ki.Signal deleting destroyed receiver: %v path %v\n", con.Recv.Name(), con.Recv.PathUnique())
			s.Cons = s.Cons[:j+copy(s.Cons[j:], s.Cons[j+1:])]
			deleted++
			continue
		}
		con.Func(con.Recv, sender, sig, data)
	}
}

// EmitGo concurrent version -- sends the signal across all the connections to the receivers
func (s *Signal) EmitGo(sender Ki, sig int64, data interface{}) {
	if sender == nil || sender.IsDestroyed() { // dead nodes don't talk..
		return
	}
	if NodeSignalTrace {
		s.EmitTrace(sender, sig, data)
	}
	deleted := 0
	for i := range s.Cons {
		j := i - deleted
		con := s.Cons[j]
		if con.Recv.IsDestroyed() {
			s.Cons = s.Cons[:j+copy(s.Cons[j:], s.Cons[j+1:])]
			deleted++
			continue
		}
		go con.Func(con.Recv, sender, sig, data)
	}
}

// function type for filtering signals
type SignalFilterFunc func(ki Ki) bool

// Emit Filtered calls function on each item only sends signal if function returns true
func (s *Signal) EmitFiltered(sender Ki, sig int64, data interface{}, fun SignalFilterFunc) {
	deleted := 0
	for i := range s.Cons {
		j := i - deleted
		con := s.Cons[j]
		if con.Recv.IsDestroyed() {
			s.Cons = s.Cons[:j+copy(s.Cons[j:], s.Cons[j+1:])]
			deleted++
			continue
		}
		if fun(con.Recv) {
			con.Func(con.Recv, sender, sig, data)
		}
	}
}

// EmitGo Filtered calls function on each item only sends signal if function returns true -- concurrent version
func (s *Signal) EmitGoFiltered(sender Ki, sig int64, data interface{}, fun SignalFilterFunc) {
	deleted := 0
	for i := range s.Cons {
		j := i - deleted
		con := s.Cons[j]
		if con.Recv.IsDestroyed() {
			s.Cons = s.Cons[:j+copy(s.Cons[j:], s.Cons[j+1:])]
			deleted++
			continue
		}
		if fun(con.Recv) {
			go con.Func(con.Recv, sender, sig, data)
		}
	}
}
