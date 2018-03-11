//
//  EventWindow.m
//  gomacdraw
//
//  Created by John Asmuth on 5/11/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#define NS_BUILD_32_LIKE_64 1

#import "EventWindow.h"
#import "GoWindow.h"

@implementation EventWindow

@synthesize eventQ, lock, gw;

- (id)initWithContentRect:(NSRect)contentRect styleMask:(NSUInteger)aStyle backing:(NSBackingStoreType)bufferingType defer:(BOOL)flag
{
    self = [super initWithContentRect:contentRect styleMask:aStyle backing:bufferingType defer:flag];
    if (self) {
        [self setLock:[[[NSConditionLock alloc] initWithCondition:0] autorelease]];
        [self setEventQ:[[[NSMutableArray alloc] initWithCapacity:10] autorelease]];
        [self setDelegate:self];
        [self setAcceptsMouseMovedEvents:YES];
    }
    
    return self;
}

- (void)dealloc
{
    [super dealloc];
}

- (void)setCallback:(void*)incallback
{
    
}

- (void)nq:(GMDEvent)e
{
    [lock lock];
    
    EventHolder* eh = [[[EventHolder alloc] initWithEvent:e] autorelease];
    [eventQ addObject:eh];
    
    
    if ([eventQ count] == 0) {
        [lock unlockWithCondition:0];
    } else {
        [lock unlockWithCondition:1];
    }
}

- (GMDEvent)dq
{
    [lock lockWhenCondition:1];
    
    EventHolder* eh = [eventQ objectAtIndex:0];
    GMDEvent e = [eh event];
    [eventQ removeObjectAtIndex:0];
    
    if ([eventQ count] == 0) {
        [lock unlockWithCondition:0];
    } else {
        [lock unlockWithCondition:1];
    }
    
    return e;
}
- (GMDEvent)mouseEvent:(NSEvent *)theEvent
{
    GMDEvent e;
    CGPoint loc = [theEvent locationInWindow];
    e.data[0] = (int)loc.x;
    e.data[1] = [self frame].size.height - (int)loc.y - (22 + 1); // not 22 so we allow 0
    e.data[2] = (int)[theEvent buttonNumber];
    return e;
}

- (void)mouseDown:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent];
    e.kind = GMDMouseDown;
    [self nq:e];
}

- (void)mouseUp:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent];
    e.kind = GMDMouseUp;
    [self nq:e];
}

- (void)mouseDragged:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent];
    e.kind = GMDMouseDragged;
    [self nq:e];
}

- (void)rightMouseDown:(NSEvent *)theEvent
{
    [self mouseDown:theEvent];
}

- (void)otherMouseDown:(NSEvent *)theEvent
{
    [self mouseDown:theEvent];
}

- (void)rightMouseUp:(NSEvent *)theEvent
{
    [self mouseUp:theEvent];
}

- (void)otherMouseUp:(NSEvent *)theEvent
{
    [self mouseUp:theEvent];
}

- (void)rightMouseDragged:(NSEvent *)theEvent
{
    [self mouseDragged:theEvent];
}

- (void)otherMouseDragged:(NSEvent *)theEvent
{
    [self mouseDragged:theEvent];
}

static int sgn(CGFloat delta)
{
    if (delta < 0)
        return -1;
    if (delta > 0)
        return 1;
    return 0;
}

- (void)scrollWheel:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent]; // fill in x/y coords
    if ([theEvent hasPreciseScrollingDeltas]) {
        e.kind = GMDScroll;
        e.data[2] = (int)[theEvent scrollingDeltaX];
        e.data[3] = (int)[theEvent scrollingDeltaY];
    } else {
        e.kind = GMDMouseWheel;
        e.data[2] = sgn([theEvent scrollingDeltaX]);
        // cocoa uses an opposite mouse wheel convention to other systems
        // we flip the sign here to provide a consistent wde interface
        e.data[3] = -sgn([theEvent scrollingDeltaY]);
    }
    [self nq:e];
}

- (void)mouseMoved:(NSEvent *)theEvent
{
    CGRect frameOrigin = [self frame];
    frameOrigin.origin = CGPointMake(0, 0);
    frameOrigin.size.height -= 22;
    if (!CGRectContainsPoint(frameOrigin, [theEvent locationInWindow])) {
        return;
    }
    GMDEvent e = [self mouseEvent:theEvent];
    e.kind = GMDMouseMoved;
    [self nq:e];
}

- (void)mouseEntered:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent];
    e.kind = GMDMouseEntered;
    [self nq:e];
}

- (void)mouseExited:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent];
    e.kind = GMDMouseExited;
    [self nq:e];
}

- (void)magnifyWithEvent:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent]; // fill in x/y coords
    e.kind = GMDMagnify;
    e.data[2] = (int)([theEvent magnification] * 65536.0);
    [self nq:e];
}

- (void)rotateWithEvent:(NSEvent *)theEvent
{
    GMDEvent e = [self mouseEvent:theEvent]; // fill in x/y coords
    e.kind = GMDRotate;
    e.data[2] = (int)([theEvent rotation] * 65536.0);
    [self nq:e];
}

- (void)flagsChanged:(NSEvent *)theEvent {
    int keycode = [theEvent keyCode];
    GMDEvent e;
    e.data[1] = keycode;
    e.data[2] = [theEvent modifierFlags];
    if ((keycode == 58 && (e.data[2] & NSAlternateKeyMask)) || 
        (keycode == 61 && (e.data[2] & NSAlternateKeyMask)) || 
        (keycode == 54 && (e.data[2] & NSCommandKeyMask)) || 
        (keycode == 55 && (e.data[2] & NSCommandKeyMask)) || 
        (keycode == 63 && (e.data[2] & NSFunctionKeyMask)) || 
        (keycode == 59 && (e.data[2] & NSControlKeyMask)) || 
        (keycode == 62 && (e.data[2] & NSControlKeyMask)) || 
        (keycode == 60 && (e.data[2] & NSShiftKeyMask)) || 
        (keycode == 56 && (e.data[2] & NSShiftKeyMask))) {
        e.kind = GMDKeyDown;
    } else if (keycode == 58 || keycode == 61 || keycode == 54 || keycode == 55 || keycode == 59 ||  keycode == 62 || keycode == 56 || keycode == 60 || keycode == 63) {
        e.kind = GMDKeyUp;
    }

    [self nq:e];
}

- (GMDEvent)keyEvent:(NSEvent*)theEvent
{
    int keycode = [theEvent keyCode];
    GMDEvent e;
    if ([[theEvent characters] length] == 0) {
        e.data[0] = [[theEvent charactersIgnoringModifiers] characterAtIndex:0];
    } else {
        e.data[0] = [[theEvent characters] characterAtIndex:0];
    }
    e.data[1] = keycode;
    e.data[2] = [theEvent modifierFlags];
    return e;
}

- (void)keyDown:(NSEvent*)theEvent
{
    GMDEvent e = [self keyEvent:theEvent];
    e.kind = GMDKeyDown;
    [self nq:e];
}

- (void)keyUp:(NSEvent*)theEvent
{
    GMDEvent e = [self keyEvent:theEvent];
    e.kind = GMDKeyUp;
    [self nq:e];
}

- (void)windowDidResize:(NSNotification *)notification
{
    NSView* view = [self contentView];
    [view removeTrackingRect:currentTrackingRect];
    currentTrackingRect = [view addTrackingRect:[view frame] owner:self userData:nil assumeInside:NO];
    GMDEvent e;
    e.kind = GMDResize;
    e.data[0] = [self frame].size.width;
    e.data[1] = [self frame].size.height-22;
    [self nq:e];
    //[gw newBuffer];
}

- (void)windowDidBecomeKey:(NSNotification *)notification
{
    GMDEvent e;
    e.kind = GMDKeyFocus;
    [self nq:e];
}

- (void)windowDidBecomeMain:(NSNotification *)notification
{
    GMDEvent e;
    e.kind = GMDMainFocus;
    [self nq:e];
}

- (void)windowWillClose:(NSNotification *)notification
{
    GMDEvent e;
    e.kind = GMDClose;
    [self nq:e];
}

@end
