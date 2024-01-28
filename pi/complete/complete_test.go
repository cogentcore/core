// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package complete

import (
	"reflect"
	"testing"
)

func TestMatchSeedString(t *testing.T) {
	completions := []string{
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
		"gi.Button.OnClick()",
		"func (e events.Event)",
		"package main",
		"package complete",
		"func main() {}",
		"core init",
	}
	// seeds to matches
	seeds := map[string][]string{
		"":           completions,
		"s":          {"Settings", "Inspect", "Git: Push", "Go: Install", "tb.Kids", "func (e events.Event)"},
		"gi":         {"Git: Commit", "Git: Push", "Git: Pull", "Go: Install", "gi.Button.OnClick()"},
		"uild":       {"Core: Build", "Go: Build"},
		"spect":      {"Inspect"},
		"pac":        {"package main", "package complete"},
		".":          {"tb.Kids", "gi.Button.OnClick()", "func (e events.Event)"},
		"gc":         {"Git: Commit"},
		"fee":        {"func (e events.Event)"},
		"feee":       {"func (e events.Event)"},
		"fm":         {"func main() {}"},
		"pi":         {"Peach Ice Cream"},
		"ci":         {"core init"},
		"gboc":       {"gi.Button.OnClick()"},
		"gb":         {"Go: Build", "gi.Button.OnClick()"},
		"git commit": {"Git: Commit"},
		"Commit":     {"Git: Commit"},
		"Git: ":      {"Git: Commit", "Git: Push", "Git: Pull"},
	}
	for seed, want := range seeds {
		have := MatchSeedString(completions, seed)
		if !reflect.DeepEqual(have, want) {
			t.Errorf("expected\n%#v\n\tbut got\n%#v", want, have)
		}
	}
}
