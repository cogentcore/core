//go:build darwin

// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#ifndef SPELL_NATIVE_DARWIN_H
#define SPELL_NATIVE_DARWIN_H

// SpellSuggestions holds suggestion strings allocated with malloc.
typedef struct {
	char **words;
	int    count;
} SpellSuggestions;

// SpellLocations holds UTF-16 locations allocated with malloc.
typedef struct {
	long *locs;
	int   count;
} SpellLocations;

int spellAvailable(void);
SpellLocations spellMisspelled(const char *text, int textLen);
SpellSuggestions spellSuggest(const char *word, int wordLen);
void spellLearn(const char *word, int wordLen);
void spellUnlearn(const char *word, int wordLen);
void spellFreeSuggestions(SpellSuggestions s);
void spellFreeLocations(SpellLocations l);

#endif
