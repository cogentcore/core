// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && !ios

// Drag-and-drop with other windows and apps. GLFW only supports file drops,
// without any feedback during the drag, so we replace the dragging
// destination methods of its content view (GLFWContentView) to send the
// whole drag to Go, and start drags from the view with a dragging source.

#include "_cgo_export.h"

#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

// the pasteboard type for mimedata.Mimes from Cogent Core apps,
// as multipart MIME data
static NSString* const cogentMimeType = @"org.cogentcore.mimedata";

// CogentDragSource is the dragging source for drags started by startDrag.
@interface CogentDragSource : NSObject <NSDraggingSource>
@property (retain) NSWindow* window;
@end

@implementation CogentDragSource

- (NSDragOperation)draggingSession:(NSDraggingSession*)session sourceOperationMaskForDraggingContext:(NSDraggingContext)context {
	if (context == NSDraggingContextWithinApplication) {
		return NSDragOperationCopy | NSDragOperationMove | NSDragOperationLink | NSDragOperationGeneric;
	}
	// other apps only copy: they do not move things (e.g., files) out
	// from under us, and Finder makes a symlink if Link is offered
	return NSDragOperationCopy;
}

// The view does not get the mouse up at the end of the drag, which GLFW
// and Go need to end the mouse press, so we send it.
- (void)draggingSession:(NSDraggingSession*)session endedAtPoint:(NSPoint)screenPoint operation:(NSDragOperation)operation {
	NSWindow* win = [[self.window retain] autorelease];
	self.window = nil;
	if (win == nil || !macDragWindowOpen(win)) { // closed
		return;
	}
	NSPoint wp = [win convertPointFromScreen:screenPoint];
	NSEvent* ev = [NSEvent mouseEventWithType:NSEventTypeLeftMouseUp location:wp modifierFlags:[NSEvent modifierFlags] timestamp:[[NSProcessInfo processInfo] systemUptime] windowNumber:[win windowNumber] context:nil eventNumber:0 clickCount:1 pressure:0];
	[[win contentView] mouseUp:ev];
}

@end

static CogentDragSource* dragSource = nil;

// dragOperation returns the drag operation for the given drag,
// based on the modifier keys as in events.DefaultModBits for drags from
// this app, and copy for drags from other apps, as allowed by the source.
static NSDragOperation dragOperation(id<NSDraggingInfo> sender) {
	NSDragOperation mask = [sender draggingSourceOperationMask];
	NSDragOperation op = NSDragOperationCopy;
	if ([sender draggingSource] != nil) {
		NSEventModifierFlags flags = [NSEvent modifierFlags];
		if (flags & NSEventModifierFlagControl) {
			op = NSDragOperationCopy;
		} else if (flags & (NSEventModifierFlagShift | NSEventModifierFlagCommand)) {
			op = NSDragOperationMove;
		} else if (flags & NSEventModifierFlagOption) {
			op = NSDragOperationLink;
		}
	}
	if (mask & op) {
		return op;
	}
	if (mask & NSDragOperationCopy) {
		return NSDragOperationCopy;
	}
	if (mask & NSDragOperationGeneric) {
		return NSDragOperationGeneric;
	}
	return mask & (NSDragOperationMove | NSDragOperationLink);
}

// dragLocation returns the location of the drag in the view,
// in points from the top left.
static NSPoint dragLocation(NSView* view, id<NSDraggingInfo> sender) {
	NSPoint p = [view convertPoint:[sender draggingLocation] fromView:nil];
	if (![view isFlipped]) {
		p.y = [view bounds].size.height - p.y;
	}
	return p;
}

static void sendDragMove(NSView* view, id<NSDraggingInfo> sender, bool leave) {
	NSPoint p = dragLocation(view, sender);
	macDragMove([view window], leave, p.x, p.y, [NSEvent modifierFlags], [sender draggingSource] != nil);
}

// CogentDragDest has the dragging destination methods that are
// copied to GLFWContentView, where self is the view.
@interface CogentDragDest : NSView
@end

@implementation CogentDragDest

- (NSDragOperation)cogent_draggingEntered:(id<NSDraggingInfo>)sender {
	sendDragMove(self, sender, false);
	return dragOperation(sender);
}

- (NSDragOperation)cogent_draggingUpdated:(id<NSDraggingInfo>)sender {
	sendDragMove(self, sender, false);
	return dragOperation(sender);
}

- (void)cogent_draggingExited:(id<NSDraggingInfo>)sender {
	sendDragMove(self, sender, true);
}

- (BOOL)cogent_wantsPeriodicDraggingUpdates {
	return NO;
}

- (BOOL)cogent_performDragOperation:(id<NSDraggingInfo>)sender {
	NSPoint p = dragLocation(self, sender);
	NSPasteboard* pb = [sender draggingPasteboard];
	NSDictionary* options = @{NSPasteboardURLReadingFileURLsOnlyKey: @YES};
	NSArray* urls = [pb readObjectsForClasses:@[[NSURL class]] options:options];
	NSUInteger n = [urls count];
	char** files = NULL;
	if (n > 0) {
		files = (char**)calloc(n, sizeof(char*));
		for (NSUInteger i = 0; i < n; i++) {
			files[i] = (char*)[urls[i] fileSystemRepresentation];
		}
	}
	NSData* mime = nil;
	if (n == 0) {
		mime = [pb dataForType:cogentMimeType];
	}
	if (n == 0 && mime == nil) {
		return NO;
	}
	macDragDrop([self window], p.x, p.y, [NSEvent modifierFlags], [sender draggingSource] != nil, files, (int)n, (void*)[mime bytes], (int)[mime length]);
	free(files);
	return YES;
}

@end

// replaceMethod sets the method sel of class c to the implementation
// of the method src of CogentDragDest.
static void replaceMethod(Class c, SEL sel, SEL src) {
	Method m = class_getInstanceMethod([CogentDragDest class], src);
	class_replaceMethod(c, sel, method_getImplementation(m), method_getTypeEncoding(m));
}

// registerDrag sets up drag-and-drop for the given window.
// It must be called on the main thread.
void registerDrag(void* nswin) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		Class c = objc_getClass("GLFWContentView");
		replaceMethod(c, @selector(draggingEntered:), @selector(cogent_draggingEntered:));
		replaceMethod(c, @selector(draggingUpdated:), @selector(cogent_draggingUpdated:));
		replaceMethod(c, @selector(draggingExited:), @selector(cogent_draggingExited:));
		replaceMethod(c, @selector(wantsPeriodicDraggingUpdates), @selector(cogent_wantsPeriodicDraggingUpdates));
		replaceMethod(c, @selector(performDragOperation:), @selector(cogent_performDragOperation:));
		dragSource = [[CogentDragSource alloc] init];
	});
	NSView* view = [(NSWindow*)nswin contentView];
	[view registerForDraggedTypes:@[NSPasteboardTypeFileURL, NSPasteboardTypeURL, cogentMimeType]];
}

// startDrag starts a drag from the given window, at the current mouse
// location, of the given files, or else the given multipart mime data
// and text. The png image is shown with its top left at the mouse,
// with the given size in points.
bool startDrag(void* nswin, char** files, int nfiles, void* mime, int mimelen, void* text, int textlen, void* png, int pnglen, double imgw, double imgh) {
	NSWindow* win = (NSWindow*)nswin;
	NSView* view = [win contentView];
	NSPoint mouse = [win mouseLocationOutsideOfEventStream];
	NSPoint vp = [view convertPoint:mouse fromView:nil];

	NSImage* img = nil;
	if (pnglen > 0) {
		img = [[[NSImage alloc] initWithData:[NSData dataWithBytes:png length:pnglen]] autorelease];
		[img setSize:NSMakeSize(imgw, imgh)];
	} else if (nfiles > 0) {
		img = [[NSWorkspace sharedWorkspace] iconForFile:[NSString stringWithUTF8String:files[0]]];
		imgw = imgh = 32;
		[img setSize:NSMakeSize(imgw, imgh)];
	}
	NSRect frame = NSMakeRect(vp.x, [view isFlipped] ? vp.y : vp.y - imgh, imgw, imgh);

	NSMutableArray* items = [NSMutableArray array];
	if (nfiles > 0) {
		for (int i = 0; i < nfiles; i++) {
			NSURL* url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:files[i]]];
			NSDraggingItem* it = [[[NSDraggingItem alloc] initWithPasteboardWriter:url] autorelease];
			[it setDraggingFrame:frame contents:(i == 0 ? img : nil)];
			[items addObject:it];
		}
	} else {
		NSPasteboardItem* pi = [[[NSPasteboardItem alloc] init] autorelease];
		if (mimelen > 0) {
			[pi setData:[NSData dataWithBytes:mime length:mimelen] forType:cogentMimeType];
		}
		if (textlen > 0) {
			NSString* s = [[[NSString alloc] initWithBytes:text length:textlen encoding:NSUTF8StringEncoding] autorelease];
			if (s != nil) {
				[pi setString:s forType:NSPasteboardTypeString];
			}
		}
		NSDraggingItem* it = [[[NSDraggingItem alloc] initWithPasteboardWriter:pi] autorelease];
		[it setDraggingFrame:frame contents:img];
		[items addObject:it];
	}

	NSEvent* ev = [NSEvent mouseEventWithType:NSEventTypeLeftMouseDragged location:mouse modifierFlags:[NSEvent modifierFlags] timestamp:[[NSProcessInfo processInfo] systemUptime] windowNumber:[win windowNumber] context:nil eventNumber:0 clickCount:1 pressure:1];
	dragSource.window = win;
	NSDraggingSession* session = [view beginDraggingSessionWithItems:items event:ev source:dragSource];
	if (session == nil) {
		dragSource.window = nil;
		return false;
	}
	session.animatesToStartingPositionsOnCancelOrFail = NO;
	return true;
}
