// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

// nativeChecker is a platform spell checker provided by the operating
// system (NSSpellChecker on macOS, UITextChecker on iOS, ISpellChecker
// on Windows, Enchant on Linux). Words learned through it go to the system user dictionary,
// shared with other apps. Implementations must be safe to call from
// any goroutine.
type nativeChecker interface {
	// misspelled returns the indexes of the misspelled words
	// in the given list of words, in increasing order.
	misspelled(words []string) []int

	// suggest returns suggested corrections for a misspelled word.
	suggest(word string) []string

	// learn adds word to the system user dictionary.
	learn(word string)

	// unlearn removes word from the system user dictionary.
	unlearn(word string)
}

// newNative returns the platform spell checker, or nil if there is none
// available. It is set by the platform-specific files.
var newNative func() nativeChecker

// UseNative determines whether the operating system spell checker is used
// when available. If false, or if there is no system spell checker, the
// built-in dictionary-based checker is used instead. It must be set before
// [NewSpell] is called.
var UseNative = true

// HasNative returns true if an operating system spell checker is
// supported on this platform and [UseNative] is true. The checker
// may still turn out to be unavailable when [NewSpell] is called.
func HasNative() bool {
	return UseNative && newNative != nil
}
