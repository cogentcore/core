// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && cgo

package spell

// The C functions are implemented with NSSpellChecker on macOS
// (native_darwin.m) and UITextChecker on iOS (native_ios.m).

/*
#cgo CFLAGS: -fobjc-arc
#cgo darwin,!ios LDFLAGS: -framework AppKit
#cgo ios LDFLAGS: -framework UIKit -framework Foundation
#include "native_darwin.h"
#include <stdlib.h>
*/
import "C"

import (
	"slices"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"
)

func init() {
	newNative = func() nativeChecker {
		if C.spellAvailable() == 0 {
			return nil
		}
		return &macChecker{}
	}
}

// macChecker uses NSSpellChecker on macOS, which uses the language(s)
// and user dictionary set in the system settings, and UITextChecker
// on iOS, which uses the user's preferred language.
type macChecker struct {
	// mu serializes access to the shared NSSpellChecker.
	mu sync.Mutex
}

// cstr returns a C copy of s, which must be freed.
func cstr(s string) (*C.char, C.int) {
	return C.CString(s), C.int(len(s))
}

func (mc *macChecker) misspelled(words []string) []int {
	// Check all of the words in one string, so that only one call to the
	// spelling server is needed when there are no errors, and record the
	// UTF-16 offset of each word to map the misspelled locations back.
	starts := make([]int, len(words))
	var b strings.Builder
	u16 := 0
	for i, w := range words {
		if i > 0 {
			b.WriteByte(' ')
			u16++
		}
		starts[i] = u16
		b.WriteString(w)
		for _, r := range w {
			u16 += utf16.RuneLen(r)
		}
	}
	text := b.String()
	ct, n := cstr(text)
	defer C.free(unsafe.Pointer(ct))
	mc.mu.Lock()
	res := C.spellMisspelled(ct, n)
	mc.mu.Unlock()
	defer C.spellFreeLocations(res)
	if res.count == 0 {
		return nil
	}
	var bad []int
	for _, loc := range unsafe.Slice(res.locs, int(res.count)) {
		i, found := slices.BinarySearch(starts, int(loc))
		if !found { // misspelling inside a word (e.g., part of a hyphenated word)
			i--
		}
		if i >= 0 && (len(bad) == 0 || bad[len(bad)-1] != i) {
			bad = append(bad, i)
		}
	}
	return bad
}

func (mc *macChecker) suggest(word string) []string {
	cw, n := cstr(word)
	defer C.free(unsafe.Pointer(cw))
	mc.mu.Lock()
	res := C.spellSuggest(cw, n)
	mc.mu.Unlock()
	defer C.spellFreeSuggestions(res)
	if res.count == 0 {
		return nil
	}
	cs := unsafe.Slice(res.words, int(res.count))
	sugs := make([]string, len(cs))
	for i, s := range cs {
		sugs[i] = C.GoString(s)
	}
	return sugs
}

func (mc *macChecker) learn(word string) {
	cw, n := cstr(word)
	defer C.free(unsafe.Pointer(cw))
	mc.mu.Lock()
	defer mc.mu.Unlock()
	C.spellLearn(cw, n)
}

func (mc *macChecker) unlearn(word string) {
	cw, n := cstr(word)
	defer C.free(unsafe.Pointer(cw))
	mc.mu.Lock()
	defer mc.mu.Unlock()
	C.spellUnlearn(cw, n)
}
