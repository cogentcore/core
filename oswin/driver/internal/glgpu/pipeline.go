// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glgpu

import (
	"log"

	"github.com/goki/gi/oswin/gpu"
)

// Pipeline manages a sequence of Programs that can be activated in an
// appropriate order to achieve some overall step of rendering.
// A new Pipeline can be created in TheGPU.NewPipeline().
type Pipeline struct {
	name  string
	progs map[string]*Program
}

// Name returns name of this pipeline
func (pl *Pipeline) Name() string {
	return pl.name
}

// SetName sets name of this pipeline
func (pl *Pipeline) SetName(name string) {
	pl.name = name
}

// AddProgram adds program with given name to the pipeline
func (pl *Pipeline) AddProgram(name string) gpu.Program {
	if pl.progs == nil {
		pl.progs = make(map[string]*Program)
	}
	pr := &Program{name: name}
	pl.progs[name] = pr
	return pr
}

// ProgramByName returns Program by name.
// Returns nil if not found (error auto logged).
func (pl *Pipeline) ProgramByName(name string) gpu.Program {
	pr, ok := pl.progs[name]
	if !ok {
		log.Printf("glgpu Pipeline ProgramByName: Program: %s not found in pipeline: %s\n", name, pl.name)
		return nil
	}
	return pr
}

// Programs returns list (slice) of Programs in pipeline
func (pl *Pipeline) Programs() []gpu.Program {
	return nil
}

func (pl *Pipeline) Delete() {
	for _, pr := range pl.progs {
		pr.Delete()
	}
}
