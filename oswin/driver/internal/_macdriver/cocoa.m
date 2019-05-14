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
// +build !3d

#include "_cgo_export.h"
#include <pthread.h>
#include <stdio.h>

#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <QuartzCore/CAMetalLayer.h>
#import <OpenGL/gl3.h>
#import <IOKit/graphics/IOGraphicsLib.h>

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
	[ctx update];
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

@class MenuDelegate;

@interface ScreenGLView : NSOpenGLView<NSWindowDelegate>
{
    MenuDelegate* _menuDel;
    NSMenu* _mainMenu;
	 CAMetalLayer* _vklayer;
    BOOL _reallyClose;
    BOOL _menuUpdtMu;
}

@property (atomic, retain) MenuDelegate* menuDel;
@property (atomic, retain) NSMenu* mainMenu;
@property (atomic, retain) CAMetalLayer* vkLayer;
@property (atomic, assign) BOOL reallyClose;
@property (atomic, assign) BOOL menuUpdtMu;

@end

@interface MenuDelegate : NSObject <NSMenuDelegate> {
    ScreenGLView* _view;
    NSMenu* _mainMenu;
}

@property (atomic, retain) ScreenGLView *view;
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

    ScreenGLView* vw = [self view];
    
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

void menuSetAsMain(ScreenGLView* view);

////////////////////////////////////////////////////////////
//  ScreenGLView

@implementation ScreenGLView

@synthesize menuDel = _menuDel;
@synthesize mainMenu = _mainMenu;
@synthesize reallyClose = _reallyClose;
@synthesize menuUpdtMu = _menuUpdtMu;

- (void)prepareOpenGL {
    self.reallyClose = NO;
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
    NSRect crect = [self.window contentRectForFrameRect: p ];
    int l = p.origin.x * pixratio;
    int t = (screenH - (p.origin.y + crect.size.height)) * pixratio;
//  	printf("res: pixratio: %g  frame origin: %g, %g, l,t: %d, %d\n", pixratio, p.origin.x, p.origin.y, l, t);

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

- (void)windowDidResize:(NSNotification *)notification {
    [self callSetGeom];
}

- (void)windowDidMiniaturize:(NSNotification *)notification {
    windowMinimized((GoUintptr)self);
}

- (void)windowDidBecomeKey:(NSNotification *)notification {
	// menuSetAsMain(self); // this is a recipe for crashing!
	windowFocused((GoUintptr)self); // instead do in response to focus event..
}

- (void)windowDidResignKey:(NSNotification *)notification {
    windowDeFocused((GoUintptr)self);
}

- (BOOL)windowShouldClose:(NSNotification *)notification {
    if (self.reallyClose == YES) {
        return YES;
    } else {
        bool close = windowCloseReq((GoUintptr)self);
        if (close) {
            return YES;
        }
        return NO;
    }
}

- (void)windowWillClose:(NSNotification *)notification {
	if (self.window.nextResponder != NULL) {
  		[self.window.nextResponder release];
  		self.window.nextResponder = NULL;
 	}
    windowClosing((GoUintptr)self);
}

- (NSMenu*) newMainMenu {
	NSMenu* mm = [[NSMenu alloc] init];
	self.mainMenu = mm; // does retain
	MenuDelegate* md = [[MenuDelegate alloc] init];
	self.menuDel = md; // does retain
	[md setView: self];
 	[md setMainMenu: mm];
 	[mm setAutoenablesItems:NO];
	uintptr_t sm = doAddSubMenu((uintptr_t)mm, "app");
	doAddMenuItem((uintptr_t)self, sm, "app", "", false, false, false, false, 0, false);
	return mm;
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
    appQuitting();
}

- (NSApplicationTerminateReply)applicationShouldTerminate:(NSNotification *)aNotification {
    bool qr = quitReq();
    if (qr) {
        return NSTerminateNow;
    } else {
        return NSTerminateCancel; // we never actually quit
    }
}

- (void)applicationWillHide:(NSNotification *)aNotification {
    // lifecycleHideAll();
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
    double b = (screenH - ((double)(top/pixratio + h)));
    // printf("new: pixratio: %g  left, top: %d, %d, l,b: %g, %g  screenH: %g\n", pixratio, left, top, l, b, screenH);

    __block ScreenGLView* view = NULL;

    dispatch_sync(dispatch_get_main_queue(), ^{
            NSString* name = [[NSString alloc] initWithUTF8String:title];

            NSRect rect = NSMakeRect(0, 0, w, h);
		
            NSWindow* window = NULL;
  
			// dialog windows on mac are actually pretty annoying -- can't see them between
			// two apps as it causes them to disappear entire when switching
            if (false) { // dialog || tool) {
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
			  [view newMainMenu];
   			  view.menuUpdtMu = NO; // not locked
												
            [window setContentView:view];
            [window setDelegate:view];
            [window makeFirstResponder:view];
            // this works to absolutely set the position:
            NSRect fr = [window frame];
            fr.origin.x = l;
            fr.origin.y = b;
            [window setFrame:fr display:YES animate:NO];

			  [name release];
			  [pixFormat release];
        });

    return (uintptr_t)view;
}

void doUpdateTitle(uintptr_t viewID, char* title) {
    NSString* nst = [[NSString alloc] initWithUTF8String:title];
    ScreenGLView* view = (ScreenGLView*)viewID;
    dispatch_async(dispatch_get_main_queue(), ^{
	    [view.window setTitle:nst];
		});
	// [nst release];
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

void doMoveWindow(uintptr_t viewID, int left, int top) {
    ScreenGLView* view = (ScreenGLView*)viewID;
    NSScreen *screen = [view.window screen];
    double pixratio = [screen backingScaleFactor];
    double screenH = [screen frame].size.height;
    NSRect fr = [view.window frame];
    NSRect crect = [view.window contentRectForFrameRect: fr ];
    double l = (double)left / pixratio;
	 double y = screenH - crect.size.height - top / pixratio;
 //    printf("new: pixratio: %g  left, top: %d, %d, l,b: %g, %g  screenH: %g\n", pixratio, left, top, l, y, screenH);
    fr.origin.x = l;
    fr.origin.y = y;
    dispatch_async(dispatch_get_main_queue(), ^{
            [view.window setFrame:fr display:YES animate:NO];
    });
}
    
void doGeomWindow(uintptr_t viewID, int left, int top, int width, int height) {
    ScreenGLView* view = (ScreenGLView*)viewID;
    NSScreen *screen = [view.window screen];
    double pixratio = [screen backingScaleFactor];
    double screenH = [screen frame].size.height;
    NSRect fr = [view.window frame];
    NSRect crect = [view.window contentRectForFrameRect: fr ];
    double frexw = (fr.size.width - crect.size.width) / pixratio;
    double frexh = (fr.size.height - crect.size.height);
    fr.size.width = ((double)width / pixratio) + frexw;
    fr.size.height = ((double)height / pixratio) + frexh;

    double crh = fr.size.height - frexh;
    double l = (double)left / pixratio;
	// t = (screenH - (p.origin.y + crect.size.height)) * pixratio;	
	// t / pix = scH - y - crh;  t/p + crh -sch = -y;   y = sch - crh - t/p
	 double y = screenH - crh - top / pixratio;
    fr.origin.x = l;
    fr.origin.y = y;
//  	 printf("new: pixratio: %g  left, top: %d, %d, l,b: %g, %g  screenH: %g\n", pixratio, left, top, l, y, screenH);
    dispatch_async(dispatch_get_main_queue(), ^{
            [view.window setFrame:fr display:YES animate:NO];
    });
}
    
void doCloseWindow(uintptr_t viewID) {
    ScreenGLView* view = (ScreenGLView*)viewID;
	//  printf("doCloseWindow: %ld\n", viewID);
    view.reallyClose = YES;
    dispatch_sync(dispatch_get_main_queue(), ^{
            [view.window performClose:view];
        });
}

void doWindowShouldClose(uintptr_t viewID) {
    ScreenGLView* view = (ScreenGLView*)viewID;
    view.reallyClose = YES;
}

void doRaiseWindow(uintptr_t viewID) {
    ScreenGLView* view = (ScreenGLView*)viewID;
	//  windowFocused((GoUintptr)self); // not getting sent otherwise apparently
    dispatch_sync(dispatch_get_main_queue(), ^{
            [view.window makeKeyAndOrderFront:view];
        });
}

void doMinimizeWindow(uintptr_t viewID) {
    ScreenGLView* view = (ScreenGLView*)viewID;
    dispatch_sync(dispatch_get_main_queue(), ^{
            [view.window performMiniaturize:view];
        });
}

void monitorEvents() {
	[NSEvent addLocalMonitorForEventsMatchingMask:
	// (NSEventAppKitDefined | NSEventMaskSystemDefined | NSMaskCursorUpdate)
	(NSEventMaskAny & ~(1 << NSEventTypeMouseMoved))
	handler:^(NSEvent *incomingEvent) {
		NSEvent *result = incomingEvent;
		NSWindow *win = [incomingEvent window];
		printf("ev: %ld  win: %ld  nm: %s\n",  (unsigned long)result.type, (long)win, (char*)[win.title UTF8String]);
		return result;
	}];
}

void startDriver() {
 // [NSAutoreleasePool new];
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    AppDelegate* delegate = [[AppDelegate alloc] init];
    [NSApp setDelegate:delegate];
	// monitorEvents();
    [NSApp run];
}

void stopDriver() {
    dispatch_async(dispatch_get_main_queue(), ^{
            [NSApp terminate:nil];
        });
}

// this is directly from https://github.com/glfw/glfw/cocoa_monitor.m 
static io_service_t IOServicePortFromCGDisplayID(CGDirectDisplayID displayID)
{
    io_iterator_t iter;
    io_service_t serv, servicePort = 0;
    
    CFMutableDictionaryRef matching = IOServiceMatching("IODisplayConnect");
    
    // releases matching for us
    kern_return_t err = IOServiceGetMatchingServices(kIOMasterPortDefault,
                             matching,
                             &iter);
    if (err)
    {
        return 0;
    }
    
    while ((serv = IOIteratorNext(iter)) != 0)
    {
        CFDictionaryRef info;
        CFIndex vendorID, productID;
        CFNumberRef vendorIDRef, productIDRef;
        Boolean success;
        
        info = IODisplayCreateInfoDictionary(serv,
                             kIODisplayOnlyPreferredName);
        
        vendorIDRef = CFDictionaryGetValue(info,
                           CFSTR(kDisplayVendorID));
        productIDRef = CFDictionaryGetValue(info,
                            CFSTR(kDisplayProductID));
        
        success = CFNumberGetValue(vendorIDRef, kCFNumberCFIndexType,
                                   &vendorID);
        success &= CFNumberGetValue(productIDRef, kCFNumberCFIndexType,
                                    &productID);

        if (!success)
        {
            CFRelease(info);
            continue;
        }
        
        if (CGDisplayVendorNumber(displayID) != vendorID ||
            CGDisplayModelNumber(displayID) != productID)
        {
            CFRelease(info);
            continue;
        }
        
        // we're a match
        servicePort = serv;
        CFRelease(info);
        break;
    }
    
    IOObjectRelease(iter);
    return servicePort;
}

// https://github.com/glfw/glfw/cocoa_monitor.m has good code
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

        const char* screenName = NULL;
        int snlen = 0;
        NSDictionary *deviceInfo = NULL;
        io_service_t serv = IOServicePortFromCGDisplayID(display);
        if (serv != 0) {
            deviceInfo = (NSDictionary *)IODisplayCreateInfoDictionary(serv, kIODisplayOnlyPreferredName);
            IOObjectRelease(serv);
            NSDictionary *localizedNames = [deviceInfo objectForKey:[NSString stringWithUTF8String:kDisplayProductName]];
            if ([localizedNames count] > 0) {
                screenName = [[localizedNames objectForKey:[[localizedNames allKeys] objectAtIndex:0]] UTF8String];
                snlen = (int)strlen(screenName);
            }
        }

        setScreen(i, dpi, pixratio, (int)screenPixW, (int)screenPixH, (int)screenSizeMM.width, (int)screenSizeMM.height, depth, (char*)screenName, snlen);

        if(deviceInfo != NULL) {
            [deviceInfo release];
        }
    }
}

///////////////////////////////////////////////////////////////////////
//   MainMenu

// qtbase/src/plugins/platforms/cocoa/qcocoamenu.mm

// called during expose or focus
void menuSetAsMain(ScreenGLView* view) {
	while (view.menuUpdtMu == YES) { }; // busy wait..
   view.menuUpdtMu = YES; // locked
	NSMenu* men = [view mainMenu];
 	[NSApp setMainMenu: men];
   view.menuUpdtMu = NO; // unlocked
}

void doSetMainMenu(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	menuSetAsMain(view);
}

uintptr_t doGetMainMenu(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	NSMenu* men = [view mainMenu];
	return (uintptr_t)men;
}

uintptr_t doGetMainMenuLock(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
	while (view.menuUpdtMu == YES) { }; // busy wait..
   view.menuUpdtMu = YES; // locked
	NSMenu* men = [view mainMenu];
	return (uintptr_t)men;
}

void doMainMenuUnlock(uintptr_t viewID) {
	ScreenGLView* view = (ScreenGLView*)viewID;
   view.menuUpdtMu = NO; // unlocked
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
	[ssmen release]; // really?
   return (uintptr_t)ssmen;
}

uintptr_t doAddMenuItem(uintptr_t viewID, uintptr_t submID, char* itmnm, char* sc, bool scShift, bool scCommand, bool scAlt, bool scControl, int tag, bool active) {
    ScreenGLView* view = (ScreenGLView*)viewID;
    NSMenu* subm  = (NSMenu*)submID;
    MenuDelegate* md = [view menuDel];
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


