// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
)

func testNative(t *testing.T) nativeChecker {
	if newNative == nil {
		t.Skip("no native spell checker on this platform")
	}
	nc := newNative()
	if nc == nil {
		t.Skip("native spell checker not available")
	}
	return nc
}

func known(nc nativeChecker, w string) bool {
	return len(nc.misspelled([]string{w})) == 0
}

func TestNative(t *testing.T) {
	nc := testNative(t)
	for _, w := range []string{"hello", "world", "don't", "Paris", "naïve"} {
		if !known(nc, w) {
			t.Errorf("%q should be known", w)
		}
	}
	for _, w := range []string{"teh", "speling", "wrold", "recieve"} {
		if known(nc, w) {
			t.Errorf("%q should not be known", w)
		}
	}
	words := []string{"naïve", "teh", "über", "quick", "wrold", "don't", "recieve", "the", "the"}
	if bad := nc.misspelled(words); !slices.Equal(bad, []int{1, 4, 6}) {
		t.Errorf("misspelled: got %v, want [1 4 6]", bad)
	}
	sugs := nc.suggest("speling")
	if !slices.Contains(sugs, "spelling") {
		t.Errorf("suggestions for speling: %v", sugs)
	}
}

func TestNativeConcurrent(t *testing.T) {
	nc := testNative(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 50 {
				if !known(nc, "hello") || known(nc, "wrold") {
					t.Error("wrong result")
				}
				nc.suggest("wrold")
			}
		})
	}
	wg.Wait()
}

// TestNativeLearn modifies the system user dictionary (and then restores it),
// so it only runs when SPELL_LEARN_TEST is set.
func TestNativeLearn(t *testing.T) {
	if os.Getenv("SPELL_LEARN_TEST") == "" {
		t.Skip("set SPELL_LEARN_TEST=1 to modify the system user dictionary")
	}
	nc := testNative(t)
	w := "zqxcogentspelltest"
	if known(nc, w) {
		t.Fatalf("%q already known", w)
	}
	nc.learn(w)
	if !known(nc, w) {
		t.Errorf("%q not known after learn", w)
	}
	nc.unlearn(w)
	if known(nc, w) {
		t.Errorf("%q still known after unlearn", w)
	}
}

var benchWords = strings.Fields("the quick brown fox jumps over the lazy dog while " +
	"spelling checkers examine every word in this line of text")

func BenchmarkNativeWord(b *testing.B) {
	nc := newNative()
	if nc == nil {
		b.Skip()
	}
	for b.Loop() {
		for _, w := range benchWords {
			known(nc, w)
		}
	}
}

func BenchmarkNativeLine(b *testing.B) {
	nc := newNative()
	if nc == nil {
		b.Skip()
	}
	for b.Loop() {
		nc.misspelled(benchWords)
	}
}

func TestSpellDataNative(t *testing.T) {
	testNative(t)
	sp := NewSpell("")
	if !sp.IsNative() {
		t.Fatal("not native")
	}
	if _, ok := sp.CheckWord("hello"); !ok {
		t.Error("hello should be known")
	}
	sugs, ok := sp.CheckWord("speling")
	if ok || !slices.Contains(sugs, "spelling") {
		t.Errorf("speling: %v %v", ok, sugs)
	}
	words := []string{"hello", "wrold", "hello", "wrold"}
	if bad := sp.Misspelled(words); !slices.Equal(bad, []int{1, 3}) {
		t.Errorf("misspelled: %v", bad)
	}
	sp.IgnoreWord("Wrold")
	if bad := sp.Misspelled(words); len(bad) != 0 {
		t.Errorf("misspelled after ignore: %v", bad)
	}
}

func TestSpellDataFallback(t *testing.T) {
	UseNative = false
	defer func() { UseNative = true }()
	sp := NewSpell("")
	if sp.IsNative() {
		t.Fatal("native")
	}
	if bad := sp.Misspelled([]string{"hello", "wrold"}); !slices.Equal(bad, []int{1}) {
		t.Errorf("misspelled: %v", bad)
	}
}

func TestCheckLexLine(t *testing.T) {
	UseNative = false
	defer func() { UseNative = true }()
	sv := Spell
	defer func() { Spell = sv }()
	Spell = NewSpell("")
	src := []rune("the “wrold” is, «quite» nice speling.")
	ser := CheckLexLine(src, nil)
	var got []string
	for _, t := range ser {
		got = append(got, string(t.Src(src)))
	}
	if !slices.Equal(got, []string{"wrold", "speling"}) {
		t.Errorf("misspelled: %q", got)
	}
}
