// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

#import <Cocoa/Cocoa.h>
#include "_cgo_export.h"

// isDarkTheme returns whether the effective appearance of the application
// is dark. It must be called on the main thread.
bool isDarkTheme() {
	NSAppearanceName name = [[NSApp effectiveAppearance] bestMatchFromAppearancesWithNames:@[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]];
	return [name isEqualToString:NSAppearanceNameDarkAqua];
}

@interface ThemeObserver : NSObject
@end

@implementation ThemeObserver
- (void)observeValueForKeyPath:(NSString*)keyPath ofObject:(id)object change:(NSDictionary*)change context:(void*)context {
	themeChanged(isDarkTheme());
}
@end

static ThemeObserver* themeObserver = nil;

// watchTheme calls themeChanged whenever the effective appearance of the
// application changes, which includes changes of the system appearance.
// It must be called on the main thread.
void watchTheme() {
	if (themeObserver != nil) {
		return;
	}
	themeObserver = [[ThemeObserver alloc] init];
	[NSApp addObserver:themeObserver forKeyPath:@"effectiveAppearance" options:0 context:nil];
}
