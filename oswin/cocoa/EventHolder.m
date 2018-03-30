//
//  EventHolder.m
//  gomacdraw
//
//  Created by John Asmuth on 5/12/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#import "EventHolder.h"


@implementation EventHolder

@synthesize event;

- (id)init
{
    self = [super init];
    if (self) {
        // Initialization code here.
    }
    
    return self;
}

- (void)dealloc
{
    [super dealloc];
}

- (EventHolder*) initWithEvent:(GMDEvent)e
{
    [self setEvent:e];
    return self;
}

@end
