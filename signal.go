// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"fmt"

	"github.com/goki/ki/kit"
)

// note: Started this code based on: github.com/tucnak/meta/

// NodeSignals are signals that a Ki node sends about updates to the tree
// structure using the NodeSignal (convert sig int64 to NodeSignals to get the
// stringer name).
type NodeSignals int64

// Standard signal types sent by ki.Node on its NodeSig for tree state changes
const (
	// NodeSignalNil is a nil signal value
	NodeSignalNil NodeSignals = iota

	// NodeSignalUpdated indicates that the node was updated -- the node Flags
	// accumulate the specific changes made since the last update signal --
	// these flags are sent in the signal data -- strongly recommend using
	// that instead of the flags, which can be subsequently updated by the
	// time a signal is processed
	NodeSignalUpdated

	// NodeSignalDeleting indicates that the node is being deleted from its
	// parent children list -- this is not blocked by Updating status and is
	// delivered immediately
	NodeSignalDeleting

	// NodeSignalDestroying indicates that the node is about to be destroyed
	// -- this is a second pass after removal from parent -- all of its
	// children and Ki fields will be destroyed too -- not blocked by updating
	// status and delivered immediately
	NodeSignalDestroying

	NodeSignalsN
)

//go:generate stringer -type=NodeSignals

// SignalTrace can be set to true to automatically print out a trace of the
// signals as they are sent
var SignalTrace bool = false

// SignalTraceString can be set to a string that will then accumulate the
// trace of signals sent, for use in testing -- otherwise the trace just goes
// to stdout
var SignalTraceString *string

// RecvFunc is a receiver function type for signals -- gets the full
// connection information and signal, data as specified by the sender.  It is
// good practice to avoid closures in these functions, which can be numerous
// and have a long lifetime, by converting the recv, send into their known
// types and referring to them directly
type RecvFunc func(recv, send Ki, sig int64, data interface{})

// Signal implements general signal passing between Ki objects, like Qt's
// Signal / Slot system.
//
// This design pattern separates three different factors:
// * when to signal that something has happened
// * who should receive that signal
// * what should the receiver do in response to the signal
//
// Keeping these three things entirely separate greatly simplifies the overall
// logic.
//
// A receiver connects in advance to a given signal on a sender to receive its
// signals -- these connections are typically established in an initialization
// step.  There can be multiple different signals on a given sender, and to
// make more efficient use of signal connections, the sender can also send an
// int64 signal value that further discriminates the nature of the event, as
// documented in the code associated with the sender (typically an enum is
// used).  Furthermore, arbitrary data as an interface{} can be passed as
// well.
//
// The Signal uses a map indexed by the receiver pointer to hold the
// connections -- this means that there can only be one such connection per
// receiver, and the order of signal emission to different receiveres will be random.
//
// Typically an inline anonymous closure receiver function is used to keep all
// the relevant code in one place.  Due to the typically long-standing nature
// of these connections, it is more efficient to avoid capturing external
// variables, and rely instead on appropriately interpreting the sent argument
// values.  e.g.:
//
// send := sender.EmbeddedStruct(KiT_SendType).(*SendType)
//
// is guaranteed to result in a usable pointer to the sender of known type at
// least SendType, in a case where that sender might actually embed that
// SendType (otherwise if it is known to be of a given type, just directly
// converting as such is fine)
type Signal struct {
	Cons map[Ki]RecvFunc `json:"-" xml:"-" desc:"map of receivers and their functions"`
}

var KiT_Signal = kit.Types.AddType(&Signal{}, nil)

// ConnectOnly first deletes any existing connections and then attaches a new
// receiver to the signal
func (s *Signal) ConnectOnly(recv Ki, fun RecvFunc) {
	s.DisconnectAll()
	s.Connect(recv, fun)
}

// Connect attaches a new receiver and function to the signal -- only one such
// connection per receiver can be made, so any existing connection to that
// receiver will be overwritten
func (s *Signal) Connect(recv Ki, fun RecvFunc) {
	if s.Cons == nil {
		s.Cons = make(map[Ki]RecvFunc)
	}
	s.Cons[recv] = fun
}

// Disconnect disconnects (deletes) the connection for a given receiver
func (s *Signal) Disconnect(recv Ki) {
	delete(s.Cons, recv)
}

// DisconnectAll removes all connections
func (s *Signal) DisconnectAll() {
	s.Cons = make(map[Ki]RecvFunc)
}

// EmitTrace records a trace of signal being emitted
func (s *Signal) EmitTrace(sender Ki, sig int64, data interface{}) {
	if SignalTraceString != nil {
		*SignalTraceString += fmt.Sprintf("ki.Signal Emit from: %v sig: %v data: %v\n", sender.Name(), NodeSignals(sig), data)
	} else {
		fmt.Printf("ki.Signal Emit from: %v sig: %v data: %v\n", sender.PathUnique(), NodeSignals(sig), data)
	}
}

// Emit sends the signal across all the connections to the receivers --
// sequentially but in random order due to the randomization of map iteration
func (s *Signal) Emit(sender Ki, sig int64, data interface{}) {
	if sender == nil || sender.IsDestroyed() { // dead nodes don't talk..
		return
	}
	if SignalTrace {
		s.EmitTrace(sender, sig, data)
	}
	for recv, fun := range s.Cons {
		if recv.IsDestroyed() {
			// fmt.Printf("ki.Signal deleting destroyed receiver: %v type %T\n", recv.Name(), recv)
			delete(s.Cons, recv)
			continue
		}
		fun(recv, sender, sig, data)
	}
}

// EmitGo is the concurrent version of Emit -- sends the signal across all the
// connections to the receivers as separate goroutines
func (s *Signal) EmitGo(sender Ki, sig int64, data interface{}) {
	if sender == nil || sender.IsDestroyed() { // dead nodes don't talk..
		return
	}
	if SignalTrace {
		s.EmitTrace(sender, sig, data)
	}
	for recv, fun := range s.Cons {
		if recv.IsDestroyed() {
			// fmt.Printf("ki.Signal deleting destroyed receiver: %v type %T\n", recv.Name(), recv)
			delete(s.Cons, recv)
			continue
		}
		go fun(recv, sender, sig, data)
	}
}

// SignalFilterFunc is the function type for filtering signals before they are
// sent -- returns false to prevent sending, and true to allow sending
type SignalFilterFunc func(recv Ki) bool

// EmitFiltered calls function on each potential receiver, and only sends
// signal if function returns true
func (s *Signal) EmitFiltered(sender Ki, sig int64, data interface{}, filtFun SignalFilterFunc) {
	for recv, fun := range s.Cons {
		if recv.IsDestroyed() {
			// fmt.Printf("ki.Signal deleting destroyed receiver: %v type %T\n", recv.Name(), recv)
			delete(s.Cons, recv)
			continue
		}
		if filtFun(recv) {
			fun(recv, sender, sig, data)
		}
	}
}

// EmitGoFiltered is the concurrent version of EmitFiltered -- calls function
// on each potential receiver, and only sends signal if function returns true
// (filtering is sequential iteration over receivers)
func (s *Signal) EmitGoFiltered(sender Ki, sig int64, data interface{}, filtFun SignalFilterFunc) {
	for recv, fun := range s.Cons {
		if recv.IsDestroyed() {
			// fmt.Printf("ki.Signal deleting destroyed receiver: %v type %T\n", recv.Name(), recv)
			delete(s.Cons, recv)
			continue
		}
		if filtFun(recv) {
			go fun(recv, sender, sig, data)
		}
	}
}

// SendSig sends a signal to one given receiver -- receiver must already be
// connected so that its receiving function is available
func (s *Signal) SendSig(recv, sender Ki, sig int64, data interface{}) {
	fun := s.Cons[recv]
	if fun != nil {
		fun(recv, sender, sig, data)
	}
}
