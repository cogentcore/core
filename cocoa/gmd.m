//
//  gmd.c
//  gomacdraw
//
//  Created by John Asmuth on 5/9/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#import "gmd.h"
#import "GoWindow.h"

#include "_cgo_export.h"

NSBundle* fw;

int initMacDraw() {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    NSThread* nop = [NSThread alloc];
    
    [NSApplication sharedApplication];

    // kickoff a thread that does nothing, so cocoa inits multi-threaded mode
    [[nop init] start];

    // setup the menu-bar. we just have a single "Quit" item.
    NSMenu* menu = [[NSMenu new] autorelease];
    NSMenuItem* appitem = [[NSMenuItem new] autorelease];
    [menu addItem:appitem];
    [NSApp setMainMenu:menu];
    NSMenu* appmenu = [[NSMenu new] autorelease];
    // XXX calling @selector(stop:) unconditionally stops the main-loop. we should instead signal the app that quit has been requested so it can save or whatever.
    [appmenu addItem:[[[NSMenuItem alloc] initWithTitle:@"Quit" action:@selector(stop:) keyEquivalent:@"q"] autorelease]];
    [appitem setSubmenu:appmenu];
    
    [pool release];
    
    return GMDNoError;
}

void releaseMacDraw() {
    [fw release];
}

void NSAppRun() {
    ProcessSerialNumber psn;
    psn.highLongOfPSN = 0;
    psn.lowLongOfPSN = kCurrentProcess;
    TransformProcessType(&psn, kProcessTransformToForegroundApplication);

    [NSApp run];
}

void NSAppStop() {
    [NSApp stop:nil];

    // In case NSAppStop is not called in response to a UI event, we need to trigger
    // a dummy event so the UI processing loop picks up the stop request.
    NSEvent* dummyEvent = [NSEvent
        otherEventWithType: NSApplicationDefined
                  location: NSZeroPoint
             modifierFlags: 0
                 timestamp: 0
              windowNumber: 0
                   context: nil
                   subtype:0
                     data1:0
                     data2:0];
    [NSApp postEvent: dummyEvent atStart: TRUE];

}

GMDWindow openWindow() {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    
    GoWindow* gw = [GoWindow alloc];
    if (gw == nil) {
        return nil;
    }
    NSRect rect = NSMakeRect(0, 0, 200, 200); // initial size isn't important
    int style = NSTitledWindowMask | NSClosableWindowMask | NSMiniaturizableWindowMask | NSResizableWindowMask;
    id window = [[[EventWindow alloc] initWithContentRect:rect styleMask:style backing:NSBackingStoreBuffered defer:NO] autorelease];
    [window makeKeyAndOrderFront:nil];
    [window setWindowController:gw];
    [window setGw:gw];
    [gw setWindow:window];
    NSImageView* view = [[[NSImageView alloc] initWithFrame:rect] autorelease];
    [view setImageFrameStyle:NSImageFrameNone];
    [view setImageScaling:NSImageScaleNone];
    NSTrackingAreaOptions tracking = NSTrackingMouseEnteredAndExited | NSTrackingActiveInActiveApp | NSTrackingInVisibleRect;
    [view addTrackingArea:[[[NSTrackingArea alloc] initWithRect:rect options:tracking owner:view userInfo:nil] autorelease]];
    [[window contentView] addSubview:view];
    [gw setImageView:view];
    [[gw window] orderFront:nil];

    [NSApp activateIgnoringOtherApps:YES];
    
    [pool release];
    
    return (GMDWindow)gw;
}

int closeWindow(GMDWindow gmdw) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    [gw close];
    [pool release];
    return 0;
}

void showWindow(GMDWindow gmdw) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    [gw showWindow:nil];
    [[gw window] orderFront:nil];
    [pool release];
}

void hideWindow(GMDWindow gmdw) {
    
}

void setWindowTitle(GMDWindow gmdw, char* title) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    NSString* nstitle = [NSString stringWithCString:title encoding:NSUTF8StringEncoding];
    [gw setTitle:nstitle];
    [pool release];
}

void setWindowSize(GMDWindow gmdw, int width, int height) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    [gw setSize:CGSizeMake(width, height)];
    [pool release];    
}

void getWindowSize(GMDWindow gmdw, int* width, int* height) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    CGSize size = [gw size];
    *width = size.width;
    *height = size.height;
    [pool release];
}

GMDEvent getNextEvent(GMDWindow gmdw) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    EventWindow* ew = (EventWindow*)[gw window];
    GMDEvent e = [ew dq];
    [pool release];
    return e;
}

GMDImage getWindowScreen(GMDWindow gmdw) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    ImageBuffer* ib = [gw buffer];
    [pool release];
    return ib;
}

void flushWindowScreen(GMDWindow gmdw) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    GoWindow* gw = (GoWindow*)gmdw;
    [gw flush];
    [pool release];
}

void setScreenData(GMDImage screen, void* data) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    ImageBuffer* ib = (ImageBuffer*)screen;
    [ib setData:(UInt8*)data];
    [pool release];
}

void getScreenSize(GMDImage screen, int* width, int* height) {
    NSAutoreleasePool* pool = [[NSAutoreleasePool alloc] init];
    ImageBuffer* ib = (ImageBuffer*)screen;
    CGSize size = [ib size];
    *width = size.width;
    *height = size.height;
    [pool release];
}

void taskReady() {
	dispatch_async(dispatch_get_main_queue(), ^{ runTask(); });
}

int isMainThread() {
	return [NSThread isMainThread];
}
