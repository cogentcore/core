// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copied and significantly modified from: https://github.com/cpmech/gosl/mpi
// Copyright 2016 The Gosl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build mpi || mpich

package mpi

/*
#cgo mpi pkg-config: ompi
#cgo mpich pkg-config: mpich
#include "mpi.h"

MPI_Comm     World     = MPI_COMM_WORLD;
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"
)

// set LogErrors to control whether MPI errors are automatically logged or not
var LogErrors = true

// Error takes an MPI error code and returns an appropriate error
// value -- either nil if no error, or the MPI error message
// with given context
func Error(ec C.int, ctxt string) error {
	if ec == C.MPI_SUCCESS {
		return nil
	}
	var rsz C.int
	str := C.malloc(C.size_t(C.MPI_MAX_ERROR_STRING))

	C.MPI_Error_string(C.int(ec), (*C.char)(str), &rsz)
	gstr := C.GoStringN((*C.char)(str), rsz)
	// C.free(str)
	err := fmt.Errorf("MPI Error: %d %s %s", ec, ctxt, gstr)
	if LogErrors {
		log.Println(err)
	}
	return err
}

// Op is an aggregation operation: Sum, Min, Max, etc
type Op int

const (
	OpSum Op = iota
	OpMax
	OpMin
	OpProd // Product
	OpLAND // logical AND
	OpLOR  // logical OR
	OpBAND // bitwise AND
	OpBOR  // bitwise OR

)

func (op Op) ToC() C.MPI_Op {
	switch op {
	case OpSum:
		return C.MPI_SUM
	case OpMax:
		return C.MPI_MAX
	case OpMin:
		return C.MPI_MIN
	case OpProd:
		return C.MPI_PROD
	case OpLAND:
		return C.MPI_LAND
	case OpLOR:
		return C.MPI_LOR
	case OpBAND:
		return C.MPI_BAND
	case OpBOR:
		return C.MPI_BOR
	}
	return C.MPI_SUM
}

const (
	// Root is the rank 0 node -- it is more semantic to use this
	Root int = 0
)

// IsOn tells whether MPI is on or not
//
//	NOTE: this returns true even after Stop
func IsOn() bool {
	var flag C.int
	C.MPI_Initialized(&flag)
	if flag != 0 {
		return true
	}
	return false
}

// Init initialises MPI
func Init() {
	C.MPI_Init(nil, nil)
}

// InitThreadSafe initialises MPI thread safe
func InitThreadSafe() error {
	var r int32
	C.MPI_Init_thread(nil, nil, C.MPI_THREAD_MULTIPLE, (*C.int)(unsafe.Pointer(&r)))
	if r != C.MPI_THREAD_MULTIPLE {
		return fmt.Errorf("MPI_THREAD_MULTIPLE can't be set: got %d", r)
	}
	return nil
}

// Finalize finalises MPI (frees resources, shuts it down)
func Finalize() {
	C.MPI_Finalize()
}

// WorldRank returns this proc's rank/ID within the World communicator.
// Returns 0 if not yet initialized, so it is always safe to call.
func WorldRank() (rank int) {
	if !IsOn() {
		return 0
	}
	var r int32
	C.MPI_Comm_rank(C.World, (*C.int)(unsafe.Pointer(&r)))
	return int(r)
}

// WorldSize returns the number of procs in the World communicator.
// Returns 1 if not yet initialized, so it is always safe to call.
func WorldSize() (size int) {
	if !IsOn() {
		return 1
	}
	var s int32
	C.MPI_Comm_size(C.World, (*C.int)(unsafe.Pointer(&s)))
	return int(s)
}

// Comm is the MPI communicator -- all MPI communication operates as methods
// on this struct.  It holds the MPI_Comm communicator and MPI_Group for
// sub-World group communication.
type Comm struct {
	comm  C.MPI_Comm
	group C.MPI_Group
}

// NewComm creates a new communicator.
// if ranks is nil, communicator is for World (all active procs).
// otherwise, defined a group-level commuicator for given ranks.
func NewComm(ranks []int) (*Comm, error) {
	cm := &Comm{}
	if len(ranks) == 0 {
		cm.comm = C.World
		return cm, Error(C.MPI_Comm_group(C.World, &cm.group), "MPI_Comm_group")
	}
	rs := make([]int32, len(ranks))
	for i := 0; i < len(ranks); i++ {
		rs[i] = int32(ranks[i])
	}
	n := C.int(len(ranks))
	r := (*C.int)(unsafe.Pointer(&rs[0]))
	var wgroup C.MPI_Group
	C.MPI_Comm_group(C.World, &wgroup)
	C.MPI_Group_incl(wgroup, n, r, &cm.group)
	return cm, Error(C.MPI_Comm_create(C.World, cm.group, &cm.comm), "Comm_create")
}

// Rank returns the rank/ID for this proc
func (cm *Comm) Rank() (rank int) {
	var r int32
	C.MPI_Comm_rank(cm.comm, (*C.int)(unsafe.Pointer(&r)))
	return int(r)
}

// Size returns the number of procs in this communicator
func (cm *Comm) Size() (size int) {
	var s int32
	C.MPI_Comm_size(cm.comm, (*C.int)(unsafe.Pointer(&s)))
	return int(s)
}

// Abort aborts MPI
func (cm *Comm) Abort() error {
	return Error(C.MPI_Abort(cm.comm, 0), "Abort")
}

// Barrier forces synchronisation
func (cm *Comm) Barrier() error {
	return Error(C.MPI_Barrier(cm.comm), "Barrier")
}
