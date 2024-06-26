// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package profile provides basic but effective profiling of targeted
// functions or code sections, which can often be more informative than
// generic cpu profiling.
//
// Here's how you use it:
//
//	// somewhere near start of program (e.g., using flag package)
//	profileFlag := flag.Bool("profile", false, "turn on targeted profiling")
//	...
//	flag.Parse()
//	profile.Profiling = *profileFlag
//	...
//	// surrounding the code of interest:
//	pr := profile.Start()
//	... code
//	pr.End()
//	...
//	// at the end or whenever you've got enough data:
//	profile.Report(time.Millisecond) // or time.Second or whatever
package profile

import (
	"cmp"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
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
	if !Profiling {
		return nil
	}
	name := ""
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		name = runtime.FuncForPC(pc).Name()
		// get rid of everything before the package
		if li := strings.LastIndex(name, "/"); li >= 0 {
			name = name[li+1:]
		}
	} else {
		err := "profile.Start: unexpected error: unable to get caller"
		errors.Log(errors.New(err))
		name = "!(" + err + ")"
	}
	if len(info) > 0 {
		name += "-" + strings.Join(info, "-")
	}
	return TheProfiler.Start(name)
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
	return TheProfiler.Start(name)
}

// Report generates a report of all the profile data collected.
func Report(units time.Duration) {
	TheProfiler.Report(units)
}

// Reset resets all of the profiling data.
func Reset() {
	TheProfiler.Reset()
}

// Profiling is whether profiling is currently enabled.
var Profiling = false

// TheProfiler is the global instance of [Profiler].
var TheProfiler = Profiler{}

// Profile represents one profiled function.
type Profile struct {
	Name   string
	Total  time.Duration
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
	p.Total += dur
	p.N++
	p.Avg = float64(p.Total) / float64(p.N)
	p.Timing = false
}

func (p *Profile) Report(tot float64, units time.Duration) {
	us := strings.TrimPrefix(units.String(), "1")
	fmt.Printf("%-60sTotal:%8.2f %s\tAvg:%6.2f\tN:%6d\tPct:%6.2f\n",
		p.Name, float64(p.Total)/float64(units), us, p.Avg/float64(units), p.N, 100.0*float64(p.Total)/tot)
}

// Profiler manages a map of profiled functions.
type Profiler struct {
	Profiles map[string]*Profile
	mu       sync.Mutex
}

// Start starts profiling and returns a Profile struct that must have .End()
// called on it when done timing
func (p *Profiler) Start(name string) *Profile {
	if !Profiling {
		return nil
	}
	p.mu.Lock()
	if p.Profiles == nil {
		p.Profiles = make(map[string]*Profile, 0)
	}
	pr, ok := p.Profiles[name]
	if !ok {
		pr = &Profile{Name: name}
		p.Profiles[name] = pr
	}
	prval := pr.Start()
	p.mu.Unlock()
	return prval
}

// Report generates a report of all the profile data collected
func (p *Profiler) Report(units time.Duration) {
	if !Profiling {
		return
	}
	list := make([]*Profile, len(p.Profiles))
	tot := 0.0
	idx := 0
	for _, pr := range p.Profiles {
		tot += float64(pr.Total)
		list[idx] = pr
		idx++
	}
	slices.SortFunc(list, func(a, b *Profile) int {
		return cmp.Compare(b.Total, a.Total)
	})
	for _, pr := range list {
		pr.Report(tot, units)
	}
}

func (p *Profiler) Reset() {
	p.Profiles = make(map[string]*Profile, 0)
}
