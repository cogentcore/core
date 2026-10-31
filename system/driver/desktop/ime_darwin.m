// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

#import <Cocoa/Cocoa.h>
#include <objc/runtime.h>

// Input method (IME) support. GLFW's content view sends keys through
// interpretKeyEvents, so the input method runs, but its NSTextInputClient
// methods discard the marked (composed) text and report the window origin
// as the caret. They are replaced here, as for drag-and-drop, to send the
// composition to Go, report the caret of the focused text widget, and keep
// keys from also being sent as key events while composing.

extern void macIMEComposition(void* nswin, char* text, int start, int end);

// CogentIME is the input method state of a view.
@interface CogentIME : NSObject
@property BOOL active;          // whether a text widget is focused
@property NSRect caret;         // in view points, with a top-left origin
@property (retain) NSString* marked; // the marked text, or nil
@end

@implementation CogentIME
- (void)dealloc {
	[_marked release];
	[super dealloc];
}
@end

// imeStates maps views to their CogentIME state, on the main thread.
static NSMapTable* imeStates;

static CogentIME* imeState(NSView* view) {
	CogentIME* st = [imeStates objectForKey:view];
	if (st == nil) {
		st = [[[CogentIME alloc] init] autorelease];
		[imeStates setObject:st forKey:view];
	}
	return st;
}

// runeCount returns the number of runes (Unicode code points) in s.
static int runeCount(NSString* s) {
	return (int)([s lengthOfBytesUsingEncoding:NSUTF32LittleEndianStringEncoding] / 4);
}

// sendComposition sends the composition of the view to Go,
// with the selected range given in UTF-16 units.
static void sendComposition(NSView* view, NSString* text, NSRange sel) {
	if (text == nil) {
		text = @"";
	}
	NSUInteger n = [text length];
	NSUInteger st = MIN(sel.location, n);
	NSUInteger ed = MIN(sel.location + sel.length, n);
	int start = runeCount([text substringToIndex:st]);
	int end = runeCount([text substringToIndex:ed]);
	macIMEComposition([view window], (char*)[text UTF8String], start, end);
}

// endComposition clears the marked text of the view,
// sending the end of the composition if it was active.
static void endComposition(NSView* view) {
	CogentIME* st = imeState(view);
	if (st.marked == nil) {
		return;
	}
	st.marked = nil;
	if (st.active) {
		sendComposition(view, nil, NSMakeRange(0, 0));
	}
}

static IMP origKeyDown;
static IMP origInsertText;

// CogentIMEView has the methods that replace those of GLFWContentView.
@interface CogentIMEView : NSView
@end

@implementation CogentIMEView

- (void)cogent_keyDown:(NSEvent*)event {
	CogentIME* st = imeState(self);
	if (st.active && st.marked != nil) { // composing: only for the input method
		[self interpretKeyEvents:@[event]];
		return;
	}
	((void (*)(id, SEL, NSEvent*))origKeyDown)(self, @selector(keyDown:), event);
}

- (void)cogent_setMarkedText:(id)string selectedRange:(NSRange)selectedRange replacementRange:(NSRange)replacementRange {
	NSString* text = [string isKindOfClass:[NSAttributedString class]] ? [string string] : string;
	CogentIME* st = imeState(self);
	if ([text length] == 0) {
		endComposition(self);
		return;
	}
	st.marked = [[text copy] autorelease];
	if (st.active) {
		sendComposition(self, text, selectedRange);
	}
}

- (void)cogent_unmarkText {
	endComposition(self);
}

- (BOOL)cogent_hasMarkedText {
	return imeState(self).marked != nil;
}

- (NSRange)cogent_markedRange {
	NSString* m = imeState(self).marked;
	if (m == nil) {
		return NSMakeRange(NSNotFound, 0);
	}
	return NSMakeRange(0, [m length]);
}

- (NSRange)cogent_selectedRange {
	NSString* m = imeState(self).marked;
	return NSMakeRange(m == nil ? 0 : [m length], 0);
}

- (NSRect)cogent_firstRectForCharacterRange:(NSRange)range actualRange:(NSRangePointer)actualRange {
	NSRect r = imeState(self).caret;
	if (![self isFlipped]) {
		r.origin.y = [self bounds].size.height - r.origin.y - r.size.height;
	}
	r = [self convertRect:r toView:nil];
	return [[self window] convertRectToScreen:r];
}

- (void)cogent_insertText:(id)string replacementRange:(NSRange)replacementRange {
	endComposition(self);
	((void (*)(id, SEL, id, NSRange))origInsertText)(self, @selector(insertText:replacementRange:), string, replacementRange);
}

@end

// replaceIMEMethod sets the method sel of class c to the implementation
// of the method src of CogentIMEView, returning the original one.
static IMP replaceIMEMethod(Class c, SEL sel, SEL src) {
	Method m = class_getInstanceMethod([CogentIMEView class], src);
	return class_replaceMethod(c, sel, method_getImplementation(m), method_getTypeEncoding(m));
}

// initIME sets up input methods for the given window.
// It must be called on the main thread.
void initIME(void* nswin) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		imeStates = [[NSMapTable weakToStrongObjectsMapTable] retain];
		Class c = objc_getClass("GLFWContentView");
		origKeyDown = replaceIMEMethod(c, @selector(keyDown:), @selector(cogent_keyDown:));
		origInsertText = replaceIMEMethod(c, @selector(insertText:replacementRange:), @selector(cogent_insertText:replacementRange:));
		replaceIMEMethod(c, @selector(setMarkedText:selectedRange:replacementRange:), @selector(cogent_setMarkedText:selectedRange:replacementRange:));
		replaceIMEMethod(c, @selector(unmarkText), @selector(cogent_unmarkText));
		replaceIMEMethod(c, @selector(hasMarkedText), @selector(cogent_hasMarkedText));
		replaceIMEMethod(c, @selector(markedRange), @selector(cogent_markedRange));
		replaceIMEMethod(c, @selector(selectedRange), @selector(cogent_selectedRange));
		replaceIMEMethod(c, @selector(firstRectForCharacterRange:actualRange:), @selector(cogent_firstRectForCharacterRange:actualRange:));
	});
	imeState([(NSWindow*)nswin contentView]);
}

// setTextInput sets whether text input is active in the given window,
// and the caret, in points with a top-left origin. It can be called
// from any thread: the state is updated on the main thread. When text
// input stops, any composition is discarded.
void setTextInput(void* nswin, int active, double x, double y, double w, double h) {
	NSWindow* win = (NSWindow*)nswin;
	[win retain];
	dispatch_async(dispatch_get_main_queue(), ^{
		NSView* view = [win contentView];
		if (view != nil) {
			CogentIME* st = imeState(view);
			if (!active && st.marked != nil) {
				[[view inputContext] discardMarkedText];
				endComposition(view);
			}
			st.active = active != 0;
			st.caret = NSMakeRect(x, y, w, h);
			if (active) {
				[[view inputContext] invalidateCharacterCoordinates];
			}
		}
		[win release];
	});
}
