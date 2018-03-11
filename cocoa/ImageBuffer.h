//
//  ImageBuffer.h
//  gomacdraw
//
//  Created by John Asmuth on 5/10/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#import <Foundation/Foundation.h>

@interface ImageBuffer : NSObject {
@private
    CGContextRef imageContext;
    CGColorSpaceRef colorSpace;
    CGSize size;
    int numBytes;
    UInt8* data;
    
    CGImageRef currentCGImage;
}

- (id)initWithSize:(CGSize)size;
- (void)setPixel:(CGPoint)point r:(UInt8)r g:(UInt8)g b:(UInt8)b a:(UInt8)a;
- (void)setData:(UInt8*)indata;
- (CGImageRef)image;
- (CGSize)size;

@end
