// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !mpi && !mpich

package mpi

// this file provides dummy versions, built by default, so mpi can be included
// generically without incurring additional complexity.

// set LogErrors to control whether MPI errors are automatically logged or not
var LogErrors = true

// Op is an aggregation operation: Sum, Min, Max, etc
type Op int

const (
	OpSum Op = iota
	OpMax
	OpMin
	OpProd
	OpLAND // logical AND
	OpLOR  // logical OR
	OpBAND // bitwise AND
	OpBOR  // bitwise OR
)

const (
	// Root is the rank 0 node -- it is more semantic to use this
	Root int = 0
)

// IsOn tells whether MPI is on or not
//
//	NOTE: this returns true even after Stop
func IsOn() bool {
	return false
}

// Init initialises MPI
func Init() {
}

// InitThreadSafe initialises MPI thread safe
func InitThreadSafe() error {
	return nil
}

// Finalize finalises MPI (frees resources, shuts it down)
func Finalize() {
}

// WorldRank returns this proc's rank/ID within the World communicator.
// Returns 0 if not yet initialized, so it is always safe to call.
func WorldRank() (rank int) {
	return 0
}

// WorldSize returns the number of procs in the World communicator.
// Returns 1 if not yet initialized, so it is always safe to call.
func WorldSize() (size int) {
	return 1
}

// Comm is the MPI communicator -- all MPI communication operates as methods
// on this struct.  It holds the MPI_Comm communicator and MPI_Group for
// sub-World group communication.
type Comm struct {
}

// NewComm creates a new communicator.
// if ranks is nil, communicator is for World (all active procs).
// otherwise, defined a group-level commuicator for given ranks.
func NewComm(ranks []int) (*Comm, error) {
	cm := &Comm{}
	return cm, nil
}

// Rank returns the rank/ID for this proc
func (cm *Comm) Rank() (rank int) {
	return 0
}

// Size returns the number of procs in this communicator
func (cm *Comm) Size() (size int) {
	return 1
}

// Abort aborts MPI
func (cm *Comm) Abort() error {
	return nil
}

// Barrier forces synchronisation
func (cm *Comm) Barrier() error {
	return nil
}
