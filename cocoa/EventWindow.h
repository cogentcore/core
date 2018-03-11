//
//  EventWindow.h
//  gomacdraw
//
//  Created by John Asmuth on 5/11/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#import <Cocoa/Cocoa.h>
#import "EventHolder.h"

#import "gmd.h"

@class GoWindow;

@interface EventWindow : NSWindow <NSWindowDelegate> {
@private
    NSConditionLock* lock;
    NSMutableArray* eventQ;
    NSTrackingRectTag currentTrackingRect;
    
    GoWindow* gw;
}


@property (retain) NSMutableArray* eventQ;
@property (retain) NSConditionLock* lock;

@property (assign) GoWindow* gw;

- (void)nq:(GMDEvent)eh;
- (GMDEvent)dq;

@end
