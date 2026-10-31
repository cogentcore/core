//go:build darwin && !ios

// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <AppKit/AppKit.h>
#include "native_darwin.h"
#include <stdlib.h>
#include <string.h>

// newString makes an NSString from UTF-8 bytes of given length.
static NSString *newString(const char *word, int wordLen) {
	return [[NSString alloc] initWithBytes:word length:wordLen encoding:NSUTF8StringEncoding];
}

int spellAvailable(void) {
	@autoreleasepool {
		return [NSSpellChecker sharedSpellChecker] != nil;
	}
}

// spellLanguage returns the language to check against: the one selected in
// the system settings, or the user's primary language if set to automatic.
// We cannot pass nil for automatic language identification, because that
// accepts a word if it is valid in any of the user's preferred languages,
// so common English misspellings such as "teh" are accepted.
static NSString *spellLanguage(NSSpellChecker *checker) {
	return checker.language;
}

SpellLocations spellMisspelled(const char *text, int textLen) {
	@autoreleasepool {
		SpellLocations res = {0};
		NSString *str = newString(text, textLen);
		if (str == nil) {
			return res;
		}
		NSSpellChecker *checker = [NSSpellChecker sharedSpellChecker];
		NSString *lang = spellLanguage(checker);
		NSInteger n = str.length;
		int capacity = 0;
		NSInteger offset = 0;
		while (offset < n) {
			NSRange r = [checker checkSpellingOfString:str startingAt:offset
				language:lang wrap:NO inSpellDocumentWithTag:0 wordCount:NULL];
			if (r.location == NSNotFound || r.length == 0) {
				break;
			}
			if (res.count == capacity) {
				capacity = capacity == 0 ? 8 : 2 * capacity;
				long *locs = (long *)realloc(res.locs, capacity * sizeof(long));
				if (locs == NULL) {
					break;
				}
				res.locs = locs;
			}
			res.locs[res.count++] = r.location;
			offset = r.location + r.length;
		}
		return res;
	}
}

SpellSuggestions spellSuggest(const char *word, int wordLen) {
	@autoreleasepool {
		SpellSuggestions res = {0};
		NSString *str = newString(word, wordLen);
		if (str == nil) {
			return res;
		}
		NSSpellChecker *checker = [NSSpellChecker sharedSpellChecker];
		NSArray<NSString *> *guesses = [checker
			guessesForWordRange:NSMakeRange(0, str.length) inString:str
			language:spellLanguage(checker) inSpellDocumentWithTag:0];
		if (guesses == nil || guesses.count == 0) {
			return res;
		}
		res.words = (char **)calloc(guesses.count, sizeof(char *));
		if (res.words == NULL) {
			return res;
		}
		for (NSString *g in guesses) {
			const char *s = [g UTF8String];
			if (s == NULL) {
				continue;
			}
			res.words[res.count++] = strdup(s);
		}
		return res;
	}
}

void spellLearn(const char *word, int wordLen) {
	@autoreleasepool {
		NSString *str = newString(word, wordLen);
		if (str != nil) {
			[[NSSpellChecker sharedSpellChecker] learnWord:str];
		}
	}
}

void spellUnlearn(const char *word, int wordLen) {
	@autoreleasepool {
		NSString *str = newString(word, wordLen);
		if (str != nil) {
			[[NSSpellChecker sharedSpellChecker] unlearnWord:str];
		}
	}
}

void spellFreeSuggestions(SpellSuggestions s) {
	for (int i = 0; i < s.count; i++) {
		free(s.words[i]);
	}
	free(s.words);
}

void spellFreeLocations(SpellLocations l) {
	free(l.locs);
}
