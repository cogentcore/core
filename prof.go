// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package prof provides very basic but effective profiling of targeted
// functions or code sections, which can often be more informative than
// generic cpu profiling
//
// Here's how you use it:
//  // somewhere near start of program (e.g., using flag package)
//  profFlag := flag.Bool("prof", false, "turn on targeted profiling")
//  ...
//  flag.Parse()
//  prof.Profiling = *profFlag
//  ...
//  // surrounding the code of interest:
//  pr := prof.Start("name of function")
//  ... code
//  pr.End()
//  ...
//  // at end or whenever you've got enough data:
//  prof.Report(time.Millisecond) // or time.Second or whatever
//
package prof

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Main User API:

// Start starts profiling and returns a Profile struct that must have .End()
// called on it when done timing -- note will be nil if not the first to start
// timing on this function -- assumes nested inner / outer loop structure for
// calls to the same method
func Start(name string) *Profile {
	return Prof.Start(name)
}

// Report generates a report of all the profile data collected
func Report(units time.Duration) {
	Prof.Report(units)
}

// Reset all data
func Reset() {
	Prof.Reset()
}

////////////////////////////////////////////////////////////////////
// IMPL:

var Profiling = false

var Prof = Profiler{}

type Profile struct {
	Name   string
	Tot    time.Duration
	N      int64
	Avg    float64
	St     time.Time
	Timing bool
}

func (p *Profile) Start() *Profile {
	if !p.Timing {
		p.St = time.Now()
		p.Timing = true
		return p
	}
	return nil
}

func (p *Profile) End() {
	if p == nil || !Profiling {
		return
	}
	dur := time.Now().Sub(p.St)
	p.Tot += dur
	p.N++
	p.Avg = float64(p.Tot) / float64(p.N)
	p.Timing = false
}

func (p *Profile) Report(tot, units float64) {
	fmt.Printf("%24v:\tTot:\t%12.2f\tAvg:\t%12.2f\tN:\t%6d\tPct:\t%5.2f\n",
		p.Name, float64(p.Tot)/units, p.Avg/units, p.N, 100.0*float64(p.Tot)/tot)
}

// Profiler manages a map of profiled functions
type Profiler struct {
	Profs map[string]*Profile
	mu    sync.Mutex
}

// Start starts profiling and returns a Profile struct that must have .End()
// called on it when done timing
func (p *Profiler) Start(name string) *Profile {
	if !Profiling {
		return nil
	}
	p.mu.Lock()
	if p.Profs == nil {
		p.Profs = make(map[string]*Profile, 0)
	}
	pr, ok := p.Profs[name]
	if !ok {
		pr = &Profile{Name: name}
		p.Profs[name] = pr
	}
	prval := pr.Start()
	p.mu.Unlock()
	return prval
}

// Report generates a report of all the profile data collected
func (p *Profiler) Report(units time.Duration) {
	if !Profiling {
		// fmt.Printf("Profiling not turned on -- set global gi.Profiling variable\n")
		return
	}
	list := make([]*Profile, len(p.Profs))
	tot := 0.0
	idx := 0
	for _, pr := range p.Profs {
		tot += float64(pr.Tot)
		list[idx] = pr
		idx++
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Tot > list[j].Tot
	})
	for _, pr := range list {
		pr.Report(tot, float64(units))
	}
}

func (p *Profiler) Reset() {
	p.Profs = make(map[string]*Profile, 0)
}
