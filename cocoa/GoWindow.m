//
//  GoWindow.m
//  gomacdraw
//
//  Created by John Asmuth on 5/9/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#define NS_BUILD_32_LIKE_64 1

#import "GoWindow.h"
#import "ImageBuffer.h"

@implementation GoWindow

@synthesize imageView, eventWindow;

- (id)initWithCoder:(NSCoder *)aDecoder
{
    self = [super initWithCoder:aDecoder];
    if (self) {
        
    }
    
    return self;
}

- (void)setTitle:(NSString*)title
{
    [[self window] setTitle:title];
}
- (void)setSize:(CGSize)size
{
    size.height += 22;
    
    CGRect frame = [[self window] frame];
    frame.size = size;
    if ([self window] == nil) {
        fprintf(stderr, "nil window in gw\n");
    }
    [[self window] setFrame:frame display:NO];
    
    [imageView setFrame:frame];
    
    if (imageView == nil) {
        fprintf(stderr, "nil imageView in gw\n");
    }
    buffer = nil;
}

- (ImageBuffer*)buffer
{
    CGSize bsize = [buffer size];
    CGSize wsize = [self size];
    if (bsize.width == wsize.width && bsize.height == wsize.height) {
        return buffer;
    }
    return [self newBuffer];
}

- (ImageBuffer*)newBuffer
{
    CGSize bufsize = [self size];
    buffer = [[ImageBuffer alloc] initWithSize:bufsize];
    [imageView setImage:nil];
    return buffer;
}

- (CGSize)size
{
    CGSize size = [[self window] frame].size;
    size.height -= 22;
    return size;
}

- (void)flush
{
    CGImageRef cgimg = [[self buffer] image];
    if (cgimg == nil) {
        return;
    }
    // CGSize size = [self size];
    // size.width = CGImageGetWidth(cgimg);
    // size.height = CGImageGetHeight(cgimg);
 
    NSImage* img = [[[NSImage alloc] autorelease] initWithCGImage:cgimg size:NSZeroSize];
    
    CGRect frame = [[self window] frame];
    frame.size.height -= 22;
    frame.origin = CGPointMake(0, 0);
    
    [imageView setFrame:frame];
    
    [imageView setImage:img];
}

- (void)dealloc
{
    [super dealloc];
}

@end
