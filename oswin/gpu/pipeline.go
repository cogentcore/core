// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

// Pipeline manages a sequence of Programs that can be activated in an
// appropriate order to achieve some overall step of rendering.
// A new Pipeline can be created in TheGPU.NewPipeline().
type Pipeline interface {
	// Name returns name of this pipeline
	Name() string

	// SetName sets name of this pipeline
	SetName(name string)

	// AddProgram adds program with given name to the pipeline
	AddProgram(name string) Program

	// ProgramByName returns program by name.
	// Returns nil if not found (error auto logged).
	ProgramByName(name string) Program

	// Programs returns list (slice) of programs in pipeline, in order added
	Programs() []Program

	// Delete deletes the GPU resources associated with this pipeline
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	Delete()
}
