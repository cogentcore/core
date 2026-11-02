// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

#import <Cocoa/Cocoa.h>

// setAppIconPNG sets the icon of the application to the given PNG image.
// It must be called on the main thread.
void setAppIconPNG(const void* data, int len) {
	NSImage* img = [[NSImage alloc] initWithData:[NSData dataWithBytes:data length:len]];
	if (img == nil) {
		return;
	}
	[NSApp setApplicationIconImage:img];
	[img release];
}
