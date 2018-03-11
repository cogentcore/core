//
//  EventHolder.h
//  gomacdraw
//
//  Created by John Asmuth on 5/12/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#import <Foundation/Foundation.h>
#import "gmd.h"

@interface EventHolder : NSObject {
@private
    GMDEvent event;
}

@property (assign) GMDEvent event;

- (EventHolder*) initWithEvent:(GMDEvent)e;

@end
