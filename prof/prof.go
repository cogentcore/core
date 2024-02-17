// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package prof provides very basic but effective profiling of targeted
// functions or code sections, which can often be more informative than
// generic cpu profiling
//
// Here's how you use it:
//
//	// somewhere near start of program (e.g., using flag package)
//	profFlag := flag.Bool("prof", false, "turn on targeted profiling")
//	...
//	flag.Parse()
//	prof.Profiling = *profFlag
//	...
//	// surrounding the code of interest:
//	pr := prof.Start()
//	... code
//	pr.End()
//	...
//	// at end or whenever you've got enough data:
//	prof.Report(time.Millisecond) // or time.Second or whatever
package prof

import (
	"cmp"
	"fmt"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

// Main User API:

// Start starts profiling and returns a Profile struct that must have [Profile.End]
// called on it when done timing. It will be nil if not the first to start
// timing on this function; it assumes nested inner / outer loop structure for
// calls to the same method. It uses the short, package-qualified name of the
// calling function as the name of the profile struct. Extra information can be
// passed to Start, which will be added at the end of the name in a dash-delimited
// format. See [StartName] for a version that supports a custom name.
func Start(info ...string) *Profile {
	name := ""
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		name = runtime.FuncForPC(pc).Name()
		// get rid of everything before the package
		if li := strings.LastIndex(name, "/"); li >= 0 {
			name = name[li+1:]
		}
	} else {
		err := "prof.Start: unexpected error: unable to get caller"
		slog.Error(err)
		name = "!(" + err + ")"
	}
	if len(info) > 0 {
		name += "-" + strings.Join(info, "-")
	}
	return Prof.Start(name)
}

// StartName starts profiling and returns a Profile struct that must have
// [Profile.End] called on it when done timing. It will be nil if not the first
// to start timing on this function; it assumes nested inner / outer loop structure
// for calls to the same method. It uses the given name as the name of the profile
// struct. Extra information can be passed to StartName, which will be added at
// the end of the name in a dash-delimited format. See [Start] for a version that
// automatically determines the name from the name of the calling function.
func StartName(name string, info ...string) *Profile {
	if len(info) > 0 {
		name += "-" + strings.Join(info, "-")
	}
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
	dur := time.Since(p.St)
	p.Tot += dur
	p.N++
	p.Avg = float64(p.Tot) / float64(p.N)
	p.Timing = false
}

func (p *Profile) Report(tot, units float64) {
	fmt.Printf("%-60sTotal:%8.2f\tAvg:%6.2f\tN:%6d\tPct:%6.2f\n",
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
	slices.SortFunc(list, func(a, b *Profile) int {
		return cmp.Compare(b.Tot, a.Tot)
	})
	for _, pr := range list {
		pr.Report(tot, float64(units))
	}
}

func (p *Profiler) Reset() {
	p.Profs = make(map[string]*Profile)
}
