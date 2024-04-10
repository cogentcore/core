// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package complete

import (
	"reflect"
	"testing"
)

var (
	completions = []string{
		"Settings",
		"Inspect",
		"Git: Commit",
		"Git: Push",
		"Git: Pull",
		"Core: Run",
		"Core: Build",
		"Go: Build",
		"Go: Install",
		"Apple",
		"Peach Ice Cream",
		"tb.Kids",
		"core.Button.OnClick()",
		"func (e events.Event)",
		"package main",
		"package complete",
		"func main() {}",
		"core init",
	}

	// seeds to matches
	seeds = map[string][]string{
		"":           completions,
		"s":          {"Settings", "Inspect", "Git: Push", "Go: Install", "tb.Kids", "func (e events.Event)"},
		"gi":         {"Git: Commit", "Git: Push", "Git: Pull", "core.Button.OnClick()", "Go: Install"},
		"uild":       {"Core: Build", "Go: Build"},
		"spect":      {"Inspect"},
		"PAC":        {"package main", "package complete"},
		".":          {"tb.Kids", "core.Button.OnClick()", "func (e events.Event)"},
		"gc":         {"Git: Commit"},
		"fee":        {"func (e events.Event)"},
		"feee":       {"func (e events.Event)"},
		"fM":         {"func main() {}"},
		"pi":         {"Peach Ice Cream"},
		"Ci":         {"core init"},
		"gboc":       {"core.Button.OnClick()"},
		"gb":         {"Go: Build", "core.Button.OnClick()"},
		"git commit": {"Git: Commit"},
		"Commit":     {"Git: Commit"},
		"Git: ":      {"Git: Commit", "Git: Push", "Git: Pull"},
	}
)

func TestMatchSeedString(t *testing.T) {
	for seed, want := range seeds {
		have := MatchSeedString(completions, seed)
		if !reflect.DeepEqual(have, want) {
			t.Errorf("expected for %q\n%#v\n\tbut got\n%#v", seed, want, have)
		}
	}
}

func TestMatchSeedCompletion(t *testing.T) {
	for seed, want := range seeds {
		cs := make([]Completion, len(completions))
		for i, c := range completions {
			cs[i].Text = c
		}
		ms := MatchSeedCompletion(cs, seed)
		have := make([]string, len(ms))
		for i, m := range ms {
			have[i] = m.Text
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("expected for %q\n%#v\n\tbut got\n%#v", seed, want, have)
		}
	}
}
