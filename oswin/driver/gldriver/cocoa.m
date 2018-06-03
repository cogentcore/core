// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64
// +build !ios

#include "_cgo_export.h"
#include <pthread.h>
#include <stdio.h>

#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <OpenGL/gl3.h>

// The variables did not exist on older OS X releases,
// we use the old variables deprecated on macOS to define them.
#if __MAC_OS_X_VERSION_MAX_ALLOWED < 101200
enum
{
    NSEventTypeScrollWheel = NSScrollWheel,
    NSEventTypeKeyDown = NSKeyDown
};
enum
{
    NSWindowStyleMaskTitled = NSTitledWindowMask,
    NSWindowStyleMaskResizable = NSResizableWindowMask,
    NSWindowStyleMaskMiniaturizable = NSMiniaturizableWindowMask,
    NSWindowStyleMaskClosable = NSClosableWindowMask
};
#endif

void makeCurrentContext(uintptr_t context) {
	NSOpenGLContext* ctx = (NSOpenGLContext*)context;
	[ctx makeCurrentContext];
}

void flushContext(uintptr_t context) {
	NSOpenGLContext* ctx = (NSOpenGLContext*)context;
	[ctx flushBuffer];
}

uint64 threadID() {
	uint64 id;
	if (pthread_threadid_np(pthread_self(), &id)) {
		abort();
	}
	return id;
}

@interface ScreenGLView : NSOpenGLView<NSWindowDelegate>
{
}
@end

@implementation ScreenGLView
- (void)prepareOpenGL {
	[self setWantsBestResolutionOpenGLSurface:YES];
	GLint swapInt = 1;
	NSOpenGLContext *ctx = [self openGLContext];
	[ctx setValues:&swapInt forParameter:NSOpenGLCPSwapInterval];

	// Using attribute arrays in OpenGL 3.3 requires the use of a VBA.
	// But VBAs don't exist in ES 2. So we bind a default one.
	GLuint vba;
	glGenVertexArrays(1, &vba);
	glBindVertexArray(vba);

	preparedOpenGL((GoUintptr)self, (GoUintptr)ctx, (GoUintptr)vba);
}

- (void)callSetGeom {
	// Calculate screen PPI.
	//
	// Note that the backingScaleFactor converts from logical
	// pixels to actual pixels, but both of these units vary
	// independently from real world size. E.g.
	//
	// 13" Retina Macbook Pro, 2560x1600, 227ppi, backingScaleFactor=2, scale=3.15
	// 15" Retina Macbook Pro, 2880x1800, 220ppi, backingScaleFactor=2, scale=3.06
	// 27" iMac,               2560x1440, 109ppi, backingScaleFactor=1, scale=1.51
	// 27" Retina iMac,        5120x2880, 218ppi, backingScaleFactor=2, scale=3.03
	NSScreen *screen = self.window.screen;

	double pixratio = [screen backingScaleFactor];
        NSArray *screens = [NSScreen screens];
	int scrno = [screens indexOfObject:screen];

	double screenH = [screen frame].size.height;
	double screenPixW = [screen frame].size.width * pixratio;

	CGDirectDisplayID display = (CGDirectDisplayID)[[[screen deviceDescription] valueForKey:@"NSScreenNumber"] intValue];
	CGSize screenSizeMM = CGDisplayScreenSize(display); // in millimeters
	float dpi = 25.4 * screenPixW / screenSizeMM.width;

	// The width and height reported to the geom package are the
	// bounds of the OpenGL view. Several steps are necessary.
	// First, [self bounds] gives us the number of logical pixels
	// in the view. Multiplying this by the backingScaleFactor
	// gives us the number of actual pixels.
	NSRect r = [self bounds];
	int w = r.size.width * pixratio;
	int h = r.size.height * pixratio;

	// origin in mac for position is lower-left
	NSRect p = [self.window frame];
	int l = p.origin.x * pixratio;
	int t = (screenH - (p.origin.y + p.size.height)) * pixratio;

	setGeom((GoUintptr)self, scrno, dpi, w, h, l, t);
}

- (void)reshape {
	[super reshape];
	[self callSetGeom];
}

- (void)drawRect:(NSRect)theRect {
	// Called during resize. Do an extra draw if we are visible.
	// This gets rid of flicker when resizing.
	drawgl((GoUintptr)self);
}

- (void)mouseEventNS:(NSEvent *)theEvent {
	NSPoint p = [theEvent locationInWindow];
	double h = self.frame.size.height;

	// Both h and p are measured in Cocoa pixels, which are a fraction of
	// physical pixels, so we multiply by backingScaleFactor.
	float scale = [self.window.screen backingScaleFactor];

	float x = p.x * scale;
	float y = (h - p.y) * scale - 1; // flip origin from bottom-left to top-left.

	float dx, dy;
	if (theEvent.type == NSEventTypeScrollWheel) {
		dx = theEvent.scrollingDeltaX;
		dy = theEvent.scrollingDeltaY;
	}

	mouseEvent((GoUintptr)self, x, y, dx, dy, theEvent.type, theEvent.buttonNumber, theEvent.modifierFlags);
}

- (void)mouseMoved:(NSEvent *)theEvent        { [self mouseEventNS:theEvent]; }
- (void)mouseDown:(NSEvent *)theEvent         { [self mouseEventNS:theEvent]; }
- (void)mouseUp:(NSEvent *)theEvent           { [self mouseEventNS:theEvent]; }
- (void)mouseDragged:(NSEvent *)theEvent      { [self mouseEventNS:theEvent]; }
- (void)rightMouseDown:(NSEvent *)theEvent    { [self mouseEventNS:theEvent]; }
- (void)rightMouseUp:(NSEvent *)theEvent      { [self mouseEventNS:theEvent]; }
- (void)rightMouseDragged:(NSEvent *)theEvent { [self mouseEventNS:theEvent]; }
- (void)otherMouseDown:(NSEvent *)theEvent    { [self mouseEventNS:theEvent]; }
- (void)otherMouseUp:(NSEvent *)theEvent      { [self mouseEventNS:theEvent]; }
- (void)otherMouseDragged:(NSEvent *)theEvent { [self mouseEventNS:theEvent]; }
- (void)scrollWheel:(NSEvent *)theEvent       { [self mouseEventNS:theEvent]; }

// raw modifier key presses
- (void)flagsChanged:(NSEvent *)theEvent {
	flagEvent((GoUintptr)self, theEvent.modifierFlags);
}

// overrides special handling of escape and tab
- (BOOL)performKeyEquivalent:(NSEvent *)theEvent {
	[self key:theEvent];
	return YES;
}

- (void)keyDown:(NSEvent *)theEvent { [self key:theEvent]; }
- (void)keyUp:(NSEvent *)theEvent   { [self key:theEvent]; }

- (void)key:(NSEvent *)theEvent {
	NSRange range = [theEvent.characters rangeOfComposedCharacterSequenceAtIndex:0];

	uint8_t buf[4] = {0, 0, 0, 0};
	if (![theEvent.characters getBytes:buf
			maxLength:4
			usedLength:nil
			encoding:NSUTF32LittleEndianStringEncoding
			options:NSStringEncodingConversionAllowLossy
			range:range
			remainingRange:nil]) {
		NSLog(@"failed to read key event %@", theEvent);
		return;
	}

	uint32_t rune = (uint32_t)buf[0]<<0 | (uint32_t)buf[1]<<8 | (uint32_t)buf[2]<<16 | (uint32_t)buf[3]<<24;

	uint8_t action;
	if ([theEvent isARepeat]) {
		action = 1;
	} else if (theEvent.type == NSEventTypeKeyDown) {
		action = 1;
	} else {
		action = 2;
	}

	keyEvent((GoUintptr)self, (int32_t)rune, action, theEvent.keyCode, theEvent.modifierFlags);
}

- (void)windowDidChangeScreenProfile:(NSNotification *)notification {
	[self callSetGeom];
}

- (void)windowDidMove:(NSNotification *)notification {
	[self callSetGeom];
}

// TODO: catch windowDidMiniaturize?

- (void)windowDidExpose:(NSNotification *)notification {
	lifecycleVisible((GoUintptr)self, true);
}

- (void)windowDidBecomeKey:(NSNotification *)notification {
	lifecycleFocused((GoUintptr)self, true);
}

- (void)windowDidResignKey:(NSNotification *)notification {
	lifecycleFocused((GoUintptr)self, false);
	if ([NSApp isHidden]) {
		lifecycleVisible((GoUintptr)self, false);
	}
}

- (void)windowWillClose:(NSNotification *)notification {
	// TODO: is this right? Closing a window via the top-left red button
	// seems to return early without ever calling windowClosing.
	if (self.window.nextResponder == NULL) {
		return; // already called close
	}

	windowClosing((GoUintptr)self);
	[self.window.nextResponder release];
	self.window.nextResponder = NULL;
}
@end

@interface AppDelegate : NSObject<NSApplicationDelegate>
{
}
@end

@implementation AppDelegate
- (void)applicationDidFinishLaunching:(NSNotification *)aNotification {
	driverStarted();
	[[NSRunningApplication currentApplication] activateWithOptions:(NSApplicationActivateAllWindows | NSApplicationActivateIgnoringOtherApps)];
}

- (void)applicationWillTerminate:(NSNotification *)aNotification {
	lifecycleDeadAll();
}

- (void)applicationWillHide:(NSNotification *)aNotification {
	lifecycleHideAll();
}
@end

// modal is pretty tricky:
// https://developer.apple.com/library/content/documentation/Cocoa/Conceptual/WinPanel/Concepts/UsingModalWindows.html#//apple_ref/doc/uid/20000223-CJBEADBA

uintptr_t doNewWindow(int width, int height, int left, int top, char* title, bool dialog, bool modal, bool tool, bool fullscreen) {
	NSScreen *screen = [NSScreen mainScreen];
	double pixratio = [screen backingScaleFactor];
	double screenH = [screen frame].size.height;
	double w = (double)width / pixratio;
	double h = (double)height / pixratio;

	double l = (double)left / pixratio;
	double b = (screenH - ((double)(top + height))) / pixratio;

	__block ScreenGLView* view = NULL;

	dispatch_sync(dispatch_get_main_queue(), ^{
		id menuBar = [NSMenu new];
		id menuItem = [NSMenuItem new];
		[menuBar addItem:menuItem];
		[NSApp setMainMenu:menuBar];

		id menu = [NSMenu new];
		NSString* name = [[NSString alloc] initWithUTF8String:title];

		id hideMenuItem = [[NSMenuItem alloc] initWithTitle:@"Hide"
			action:@selector(hide:) keyEquivalent:@"h"];
		[menu addItem:hideMenuItem];

		id quitMenuItem = [[NSMenuItem alloc] initWithTitle:@"Quit"
			action:@selector(terminate:) keyEquivalent:@"q"];
		[menu addItem:quitMenuItem];
		[menuItem setSubmenu:menu];

		NSRect rect = NSMakeRect(0, 0, w, h);
		
		NSWindow* window = NULL;
  
		if (dialog || tool) {
		  window = [[NSPanel alloc] initWithContentRect:rect
						       styleMask:NSWindowStyleMaskTitled
							backing:NSBackingStoreBuffered
							  defer:NO];
		  if (dialog) {
		    window.styleMask |= NSWindowStyleMaskResizable;
		    window.styleMask |= NSWindowStyleMaskMiniaturizable;
		    window.styleMask |= NSWindowStyleMaskClosable;
		  }

		  window.title = name;
		  window.displaysWhenScreenProfileChanges = YES;
		  [window setAcceptsMouseMovedEvents:YES];
		}
		else {
		  window = [[NSWindow alloc] initWithContentRect:rect
						       styleMask:NSWindowStyleMaskTitled
							 backing:NSBackingStoreBuffered
							   defer:NO];
		  window.styleMask |= NSWindowStyleMaskResizable;
		  window.styleMask |= NSWindowStyleMaskMiniaturizable;
		  window.styleMask |= NSWindowStyleMaskClosable;

		  if (fullscreen) {
		    window.styleMask |= NSWindowStyleMaskFullScreen;
		  }

		  window.title = name;
		  window.displaysWhenScreenProfileChanges = YES;
		  [window setAcceptsMouseMovedEvents:YES];
		}

		NSOpenGLPixelFormatAttribute attr[] = {
			NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core,
			NSOpenGLPFAColorSize,     24,
			NSOpenGLPFAAlphaSize,     8,
			NSOpenGLPFADepthSize,     16,
			NSOpenGLPFADoubleBuffer,
			NSOpenGLPFAAllowOfflineRenderers,
			0
		};
		id pixFormat = [[NSOpenGLPixelFormat alloc] initWithAttributes:attr];
		view = [[ScreenGLView alloc] initWithFrame:rect pixelFormat:pixFormat];
		[window setContentView:view];
		[window setDelegate:view];
		[window makeFirstResponder:view];
		// this works to absolutely set the position:
		NSRect fr = [window frame];
		fr.origin.x = l;
		fr.origin.y = b;
		[window setFrame:fr display:YES animate:NO];
		
	});

	return (uintptr_t)view;
}

void doShowWindow(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	dispatch_async(dispatch_get_main_queue(), ^{
		[view.window makeKeyAndOrderFront:view.window];
	});
}

void doResizeWindow(uintptr_t viewID, int width, int height) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	dispatch_async(dispatch_get_main_queue(), ^{
	    NSSize sz;
	    sz.width = width;
	    sz.height = height;
	    [view.window setContentSize:sz];
	});
}

void doCloseWindow(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	dispatch_sync(dispatch_get_main_queue(), ^{
		[view.window performClose:view];
	});
}

void startDriver() {
	[NSAutoreleasePool new];
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
	AppDelegate* delegate = [[AppDelegate alloc] init];
	[NSApp setDelegate:delegate];
	[NSApp run];
}

void stopDriver() {
	dispatch_async(dispatch_get_main_queue(), ^{
		[NSApp terminate:nil];
	});
}

void getScreens() {
        NSArray *screens = [NSScreen screens];
	int nscr = [screens count];
	resetScreens();
	int i;
	for (i=0; i < nscr; i++) {
	  NSScreen *screen = [screens objectAtIndex: i];
	  float pixratio = [screen backingScaleFactor];
	  float screenPixW = [screen frame].size.width * pixratio;
	  float screenPixH = [screen frame].size.height * pixratio;
	  int depth = [screen depth];
	  CGDirectDisplayID display = (CGDirectDisplayID)[[[screen deviceDescription] valueForKey:@"NSScreenNumber"] intValue];
	  CGSize screenSizeMM = CGDisplayScreenSize(display); // in millimeters
	  float dpi = 25.4 * screenPixW / screenSizeMM.width;
	  setScreen(i, dpi, pixratio, (int)screenPixW, (int)screenPixH, (int)screenSizeMM.width, (int)screenSizeMM.height, depth);
	}
}

///////////////////////////////////////////////////////////////////////
//   Clipboard

// https://developer.apple.com/documentation/appkit/nspasteboard
// https://github.com/jtanx/libclipboard/blob/master/src/clipboard_cocoa.c

void clipAvail() {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
	// NSArray *itms = [pb pasteboardItems];
	char *fmt = NULL;
	// todo: decode actual format!
	NSString *ns_clip = [pb stringForType:NSPasteboardTypeString];
	if (ns_clip == NULL) {
	  setClipAvail(false);
	  return;
	}
	fmt = "text/plain";
	setClipFmt(fmt, strlen(fmt));
	setClipAvail(true);
}

void clipAvailFmt(char* fmt, int len) {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
	// NSArray *itms = [pb pasteboardItems];
	// todo: support more formats!
	if (strncmp(fmt, "text/plain", len) == 0) {
	  NSString *ns_clip = [pb stringForType:NSPasteboardTypeString];
	  if (ns_clip == NULL) {
	    setClipAvail(false);
	    return;
	  }
	  setClipAvail(true);
	  return;
	}
}

void clipRead() {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
	// NSArray *itms = [pb pasteboardItems];
	NSString *ns_clip = [pb stringForType:NSPasteboardTypeString];
	if (ns_clip == NULL) {
	  setClipData(NULL, 0);
	  return;
	}
	char* fmt = "text/plain";
	setClipFmt(fmt, strlen(fmt));
	const char* utf8_clip = [ns_clip UTF8String];
	int len = (int)strlen(utf8_clip);
	setClipData((char*)utf8_clip, len);
}

void clipReadFmt(char* fmt, int len) {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
	// NSArray *itms = [pb pasteboardItems];
	// todo: support more formats!
	if (strncmp(fmt, "text/plain", len) == 0) {
	  NSString *ns_clip = [pb stringForType:NSPasteboardTypeString];
	  if (ns_clip == NULL) {
	    return;
	  }
	  const char* utf8_clip = [ns_clip UTF8String];
	  int len = (int)strlen(utf8_clip);
	  setClipData((char*)utf8_clip, len);
	}
}

void clipWrite(char* data, int len, char* fmt, int fmtlen) {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
	NSString *ns_clip;
	bool ret;

	// todo: support more formats!
	if (strncmp(fmt, "text/plain", fmtlen) == 0) {
	  ns_clip = [[NSString alloc] initWithBytes:data length:len encoding:NSUTF8StringEncoding];

	  [pb declareTypes:[NSArray arrayWithObject:NSStringPboardType] owner:nil];
	  ret = [pb setString:ns_clip forType:NSStringPboardType];
	  [ns_clip release];

	  // long serial = [pb changeCount];
	  // OSAtomicCompareAndSwapLong(cb->last_cb_serial, serial, &cb->last_cb_serial);
	}
}	
