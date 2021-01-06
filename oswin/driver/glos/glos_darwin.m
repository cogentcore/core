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
#include <stdio.h>

#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <objc/runtime.h>
//#import <IOKit/graphics/IOGraphicsLib.h>

///////////////////////////////////////////////////////////////////////
//   Clipboard / Pasteboard / drag-n-drop

// https://developer.apple.com/documentation/appkit/nspasteboard
// https://github.com/jtanx/libclipboard/blob/master/src/clipboard_cocoa.c
// https://developer.apple.com/documentation/appkit/nspasteboardreading
// last one shows that the list is -- basic classes, not fine-grained uti's

// for anything that is not a basic type (NSString, NSAttributedString,
// NSIMage, etc) we encode a NSString that gives the mimetype, e.g.,
// application/json, using standard mime format header

static const char* mime_hdr = "MIME-Version: 1.0\nContent-type: ";

// return true if empty
bool pasteIsEmpty(NSPasteboard* pb) {
    NSDictionary *options = [NSDictionary dictionary];
    NSArray *classes = [[NSArray alloc] initWithObjects:[NSString class], nil];
    bool has = [pb canReadObjectForClasses:classes options:options];
	[classes release];
    return !has;
}

// get all the regular text: NSString -- go-side deals with any further processing
void pasteReadText(NSPasteboard* pb) {
    NSDictionary *options = [NSDictionary dictionary];
    NSArray *classes = [[NSArray alloc] initWithObjects:[NSString class], nil];
    NSArray *itms = [pb readObjectsForClasses:classes options:options];
    if (itms != nil) {
        int n = [itms count];
        int i;
        for (i=0; i<n; i++) {
            NSString* clip = [itms objectAtIndex: i];
            const char* utf8_clip = [clip UTF8String];
            int datalen = (int)strlen(utf8_clip);
            addMimeText((char*)utf8_clip, datalen);
        }
    }
    [classes release];
    // [itms release]; // we do NOT own this!
}

// not using / tested yet: just grabbed from Qt..
void pasteReadHTML(NSPasteboard* pb, char* typ, int len) {
    NSDictionary *options = [NSDictionary dictionary];
    NSArray *classes = [[NSArray alloc] initWithObjects:[NSAttributedString class], nil];
    NSArray *itms = [pb readObjectsForClasses:classes options:options];
    if (itms != nil) {
        int n = [itms count];
        int i;
        for (i=0; i<n; i++) {
            NSAttributedString* clip = [itms objectAtIndex: i];
            NSError *error;
            NSRange range = NSMakeRange(0, [clip length]);
            NSDictionary *dict = [NSDictionary dictionaryWithObject:NSHTMLTextDocumentType forKey:NSDocumentTypeDocumentAttribute];
            NSData *dat = [clip dataFromRange:range documentAttributes:dict error:&error];

            NSUInteger datalen = [dat length];
            Byte *bytedata = (Byte*)malloc(datalen);
            [dat getBytes:bytedata length:datalen];
            addMimeText((char*)bytedata, datalen);
            free(bytedata);
        }
    }
    [classes release];
    // [itms release];
}

static NSMutableArray *pasteWriteItems = NULL;

// add text to the list of items to paste
void pasteWriteAddText(char* data, int len) {
    NSString *ns_clip;
    bool ret;

    if(pasteWriteItems == NULL) {
        pasteWriteItems = [NSMutableArray array];
		 [pasteWriteItems retain];
    }
    
    ns_clip = [[NSString alloc] initWithBytes:data length:len encoding:NSUTF8StringEncoding];
    [pasteWriteItems addObject:ns_clip];
	[ns_clip release]; // pastewrite owns
}	

void pasteWrite(NSPasteboard* pb) {
    if(pasteWriteItems == NULL) {
        return;
    }
    [pb writeObjects: pasteWriteItems];
	[pasteWriteItems release];
    pasteWriteItems = NULL;
}	

// clip just calls paste versions with generalPasteboard

bool clipIsEmpty() {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    if(pb == NULL) {
        return true;
    }
    return pasteIsEmpty(pb);
}

void clipReadText() {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    if(pb == NULL) {
        return;
    }
    pasteReadText(pb);
}

void clipWrite() {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    if(pb == NULL) {
        return;
    }
    pasteWrite(pb);
}

void clipClear() {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    if(pb == NULL) {
        return;
    }
    [pb clearContents];
    if(pasteWriteItems != NULL) {
		 [pasteWriteItems release];
        pasteWriteItems = NULL;
    }
}


///////////////////////////////////////////////////////////////////////
//   Cursor

// https://developer.apple.com/documentation/appkit/nscursor
// Qt: qtbase/src/plugins/platforms/cocoa/qcocoacursor.mm
// http://doc.qt.io/qt-5/qt.html#CursorShape-enum

NSCursor* getCursor(int curs) {
    NSCursor* cr = NULL;
    switch (curs) {
    case 0: // Arrow
        cr = [NSCursor arrowCursor];
        break;
    case 1: // Cross
        cr = [NSCursor crosshairCursor];
        break;
    case 2: // DragCopy
        cr = [NSCursor dragCopyCursor];
        break;
    case 3: // DragMove
        cr = [NSCursor arrowCursor];
        break;
    case 4: // DragLink
        cr = [NSCursor dragLinkCursor];
        break;
    case 5: // HandPointing
        cr = [NSCursor pointingHandCursor];
        break;
    case 6: // HandOpen
        cr = [NSCursor openHandCursor];
        break;
    case 7: // HandClosed
        cr = [NSCursor closedHandCursor];
        break;
    case 8: // Help
        cr = [NSCursor arrowCursor]; // todo: needs custom
        break;
    case 9: // IBeam
        cr = [NSCursor IBeamCursor];
        break;
    case 10: // Not
        cr = [NSCursor operationNotAllowedCursor];
        break;
    case 11: // UpDown
        cr = [NSCursor resizeUpDownCursor];
        break;
    case 12: // LeftRight
        cr = [NSCursor resizeLeftRightCursor];
        break;
    case 13: // UpRight
        cr = [NSCursor resizeUpDownCursor]; // todo: needs custom
        break;
    case 14: // UpLeft
        cr = [NSCursor resizeLeftRightCursor]; // todo: needs custom
        break;
    case 15: // AllArrows
        cr = [NSCursor arrowCursor]; // todo: needs custom
        break;
    case 16: // Wait
        cr = [NSCursor disappearingItemCursor]; // todo: needs custom
        break;
    }
    return cr;
}

void pushCursor(int curs) {
    NSCursor* cr = getCursor(curs);
    if (cr != NULL) {
        [cr push];
    }
}

void setCursor(int curs) {
    NSCursor* cr = getCursor(curs);
    if (cr != NULL) {
        [cr set];
    }
}

void popCursor() {
    [NSCursor pop];
}

void hideCursor() {
    [NSCursor hide];
}

void showCursor() {
    [NSCursor unhide];
}

///////////////////////////////////////////////////////////////////////
//   MainMenu

// qtbase/src/plugins/platforms/cocoa/qcocoamenu.mm

@interface MenuDelegate : NSObject <NSMenuDelegate> {
    NSWindow* _view;
    NSMenu* _mainMenu;
}

@property (atomic, retain) NSWindow *view;
@property (atomic, retain) NSMenu *mainMenu;

@end

@implementation MenuDelegate

@synthesize view = _view;
@synthesize mainMenu = _mainMenu;

- (void) itemFired:(NSMenuItem*) item
{
    NSString* title = [item title];
    const char* utf8_title = [title UTF8String];
    long tag = (long)[item tag];

    NSWindow* vw = [self view];
    
    int tilen = (int)strlen(utf8_title);
    menuFired((GoUintptr)vw, (char*)utf8_title, tilen, tag);
}

// Cocoa will query the menu item's target for the worksWhenModal selector.
// So we need to implement this to allow the items to be handled correctly
// when a modal dialog is visible.
- (BOOL)worksWhenModal
{
    return YES;
}

@end

MenuDelegate* newMainMenu(NSWindow* view) {
	NSMenu* mm = [[NSMenu alloc] init];
	MenuDelegate* md = [[MenuDelegate alloc] init];
	[md retain];
	[mm retain];
	[md setView: view];
 	[md setMainMenu: mm];
 	[mm setAutoenablesItems:NO];
	uintptr_t sm = doAddSubMenu((uintptr_t)mm, "app");
	doAddMenuItem((uintptr_t)view, sm, (uintptr_t)md, "app", "", false, false, false, false, 0, false);
	return md;
}

uintptr_t doNewMainMenu(uintptr_t viewID) {
	NSWindow* view = (NSWindow*)viewID;
	MenuDelegate* md = newMainMenu(view);
	return (uintptr_t)md;
}

uintptr_t mainMenuFromDelegate(uintptr_t delID) {
	MenuDelegate* md = (MenuDelegate*)delID;
	NSMenu* mm = md.mainMenu;
	return (uintptr_t)mm;
}

void menuSetAsMain(NSWindow* view, NSMenu* men) {
 	[NSApp setMainMenu: men];
}

void doSetMainMenu(uintptr_t viewID, uintptr_t menID) {
	NSWindow* view = (NSWindow*)viewID;
	NSMenu* men = (NSMenu*)menID;
	menuSetAsMain(view, men);
}

void doMenuReset(uintptr_t menuID) {
    NSMenu* men  = (NSMenu*)menuID;
    [men removeAllItems];
}

uintptr_t doAddSubMenu(uintptr_t menuID, char* mnm) {
   NSMenu* men  = (NSMenu*)menuID;
   NSString* title = [[NSString alloc] initWithUTF8String:mnm];
    
   NSMenuItem* smen = [men addItemWithTitle:title action:nil keyEquivalent: @""];
   NSMenu* ssmen = [[NSMenu alloc] initWithTitle:title];
   smen.submenu = ssmen;
   [ssmen setAutoenablesItems:NO];
	[title release];
	[ssmen release]; // yes really!
   return (uintptr_t)ssmen;
}

uintptr_t doAddMenuItem(uintptr_t viewID, uintptr_t submID, uintptr_t delID, char* itmnm, char* sc, bool scShift, bool scCommand, bool scAlt, bool scControl, int tag, bool active) {
 	 NSWindow* view = (NSWindow*)viewID;
    NSMenu* subm  = (NSMenu*)submID;
    MenuDelegate* md = (MenuDelegate*)delID;
    NSString* title = [[NSString alloc] initWithUTF8String:itmnm];
    NSString* scut = [[NSString alloc] initWithUTF8String:sc];

    // todo: always seems to assume command
    NSMenuItem* mi = [subm addItemWithTitle:title action:@selector(itemFired:) keyEquivalent: scut];
    mi.target = md;
    mi.tag = tag;
    mi.enabled = active;
    if (scCommand) {
        if (scShift) {
            mi.keyEquivalentModifierMask = NSEventModifierFlagShift | NSEventModifierFlagCommand;
        } else {
            mi.keyEquivalentModifierMask = NSEventModifierFlagCommand;
        }
    } else if (scAlt) {
        if (scShift) {
            mi.keyEquivalentModifierMask = NSEventModifierFlagShift | NSEventModifierFlagOption;
        } else {
            mi.keyEquivalentModifierMask = NSEventModifierFlagOption;
        }
    } else if (scControl) {
        if (scShift) {
            mi.keyEquivalentModifierMask = NSEventModifierFlagShift | NSEventModifierFlagControl;
        } else {
            mi.keyEquivalentModifierMask = NSEventModifierFlagControl;
        }
    }
	[title release];
	[scut release];
    return (uintptr_t)mi;
}

void doAddSeparator(uintptr_t menuID) {
    NSMenu* menu  = (NSMenu*)menuID;
    NSMenuItem* sep = [NSMenuItem separatorItem];
    [menu addItem: sep];
}

uintptr_t doMenuItemByTitle(uintptr_t menuID, char* mnm) {
    NSMenu* men  = (NSMenu*)menuID;
    NSString* title = [[NSString alloc] initWithUTF8String:mnm];
    NSMenuItem* mi = [men itemWithTitle:title];
	[title release];
    return (uintptr_t)mi;
}

uintptr_t doMenuItemByTag(uintptr_t menuID, int tag) {
    NSMenu* men  = (NSMenu*)menuID;
    NSMenuItem* mi = [men itemWithTag:tag];
    return (uintptr_t)mi;
}

void doSetMenuItemActive(uintptr_t mitmID, bool active) {
    NSMenuItem* mi  = (NSMenuItem*)mitmID;
    mi.enabled = active;
}

/////////////////////////////////////////////////////////////////
// OpenFiles event delegate
// https://github.com/glfw/glfw/issues/1024

NS_ASSUME_NONNULL_BEGIN

@interface GLFWCustomDelegate : NSObject
+ (void)load; // load is called before even main() is run (as part of objc class registration)
@end

NS_ASSUME_NONNULL_END

@implementation GLFWCustomDelegate

+ (void)load{
	static dispatch_once_t onceToken;
	dispatch_once(&onceToken, ^{
		Class class = objc_getClass("GLFWApplicationDelegate");
	
		[GLFWCustomDelegate swizzle:class src:@selector(application:openFile:) tgt:@selector(swz_application:openFile:)];
		[GLFWCustomDelegate swizzle:class src:@selector(application:openFiles:) tgt:@selector(swz_application:openFiles:)];
	});
}

+ (void) swizzle:(Class) original_c src:(SEL)original_s tgt:(SEL)target_s{
	Class target_c = [GLFWCustomDelegate class];
	Method originalMethod = class_getInstanceMethod(original_c, original_s);
	Method swizzledMethod = class_getInstanceMethod(target_c, target_s);

	BOOL didAddMethod =
	class_addMethod(original_c,
					original_s,
					method_getImplementation(swizzledMethod),
					method_getTypeEncoding(swizzledMethod));

	if (didAddMethod) {
		class_replaceMethod(original_c,
							target_s,
							method_getImplementation(originalMethod),
							method_getTypeEncoding(originalMethod));
	} else {
		method_exchangeImplementations(originalMethod, swizzledMethod);
	}
}

- (BOOL)swz_application:(NSApplication *)sender openFile:(NSString *)filename{
	const char* utf_fn = filename.UTF8String;
   int flen = (int)strlen(utf_fn);
	macOpenFile((char*)utf_fn, flen);
	return true;
}

- (void)swz_application:(NSApplication *)sender openFiles:(NSArray<NSString *> *)filenames{
	int n = [filenames count];
	int i;
	for (i=0; i<n; i++) {
  		NSString* fnm = [filenames objectAtIndex: i];
		const char* utf_fn = fnm.UTF8String;
		int flen = (int)strlen(utf_fn);
		macOpenFile((char*)utf_fn, flen);
	}
}

@end

