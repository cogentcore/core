// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package spell provides functions for spell check and correction.

It uses the operating system spell checker when available:
NSSpellChecker on macOS, the Windows Spell Checking API (ISpellChecker)
on Windows, and Enchant-2 on Linux (loaded at run time if installed).
These use the language and user dictionary of the system, so words
learned here are shared with other apps, and vice versa.

Otherwise (web, mobile, or if [UseNative] is false), a built-in English
dictionary checker is used, based on https://github.com/sajari/fuzzy,
with learned words saved in its own user dictionary file.
That file is never added to the system dictionary.

A single globally usable spell checker is managed in [Spell].
*/
package spell
