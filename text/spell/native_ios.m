//go:build ios

// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

#import <UIKit/UIKit.h>
#include "native_darwin.h"
#include <stdlib.h>
#include <string.h>

// newString makes an NSString from UTF-8 bytes of given length.
static NSString *newString(const char *word, int wordLen) {
	return [[NSString alloc] initWithBytes:word length:wordLen encoding:NSUTF8StringEncoding];
}

static UITextChecker *checker;
static NSString *language;

// spellLanguage returns the first of the user's preferred languages
// that UITextChecker supports, matching on the language alone if
// needed, or English.
static NSString *spellLanguage(void) {
	NSArray<NSString *> *avail = [UITextChecker availableLanguages];
	if (avail.count == 0) {
		return nil;
	}
	for (NSString *pref in [NSLocale preferredLanguages]) {
		// preferred languages are like en-US, available ones like en_US.
		NSString *lang = [pref stringByReplacingOccurrencesOfString:@"-" withString:@"_"];
		if ([avail containsObject:lang]) {
			return lang;
		}
		NSString *base = [lang componentsSeparatedByString:@"_"][0];
		for (NSString *a in avail) {
			if ([a isEqualToString:base] || [a hasPrefix:[base stringByAppendingString:@"_"]]) {
				return a;
			}
		}
	}
	for (NSString *a in avail) {
		if ([a hasPrefix:@"en"]) {
			return a;
		}
	}
	return avail[0];
}

int spellAvailable(void) {
	@autoreleasepool {
		language = spellLanguage();
		if (language == nil) {
			return 0;
		}
		checker = [[UITextChecker alloc] init];
		return checker != nil;
	}
}

SpellLocations spellMisspelled(const char *text, int textLen) {
	@autoreleasepool {
		SpellLocations res = {0};
		NSString *str = newString(text, textLen);
		if (str == nil) {
			return res;
		}
		NSInteger n = str.length;
		int capacity = 0;
		NSInteger offset = 0;
		while (offset < n) {
			NSRange r = [checker rangeOfMisspelledWordInString:str range:NSMakeRange(0, n)
				startingAt:offset wrap:NO language:language];
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
		NSArray<NSString *> *guesses = [checker guessesForWordRange:NSMakeRange(0, str.length)
			inString:str language:language];
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
			[UITextChecker learnWord:str];
		}
	}
}

void spellUnlearn(const char *word, int wordLen) {
	@autoreleasepool {
		NSString *str = newString(word, wordLen);
		if (str != nil) {
			[UITextChecker unlearnWord:str];
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
