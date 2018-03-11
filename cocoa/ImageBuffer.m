//
//  ImageBuffer.m
//  gomacdraw
//
//  Created by John Asmuth on 5/10/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

#import "ImageBuffer.h"


static const void *
dataPointer(void *info) {
    //fprintf(stderr, "dataPointer\n");
	return info;
}

static void
releaseData(void *info, const void *pointer) {
    //fprintf(stderr, "releaseData\n");
	free(info);
}

static size_t
bytesAtPosition(void *info, void *buffer, off_t pos, size_t n) {
    //fprintf(stderr, "bytesAtPosition\n");
	unsigned char *data;
	data = info;
	memmove(buffer, data + pos, n);
	return n;
}

static void
releaseInfo(void *info) {
    //fprintf(stderr, "releaseInfo\n");
}

static CGDataProviderDirectCallbacks callbacks = {
	0,
	dataPointer,
	releaseData,
	bytesAtPosition,
	releaseInfo,
};


@implementation ImageBuffer

- (id)initWithSize:(CGSize)sizein
{
    self = [super init];
    if (self) {
        colorSpace = CGColorSpaceCreateDeviceRGB();
        size = sizein;
        numBytes = sizeof(UInt8)*4*size.width*size.height;
        currentCGImage = nil;
    }
    
    return self;
}

- (void)dealloc
{
    CGColorSpaceRelease(colorSpace);
    [super dealloc];
}

- (CGImageRef)image
{
    if (data == nil) {
        return nil;
    }
    if (currentCGImage != nil) {
        CGImageRelease(currentCGImage);
    }
    UInt8* copyData = (UInt8*)malloc(numBytes);
    memcpy(copyData, data, numBytes);
    
    CGDataProviderRef dp = CGDataProviderCreateDirect(copyData, numBytes, &callbacks);
    
    currentCGImage = CGImageCreate(size.width, size.height,
                     8, //bitsPerComponent
                     32, //bitsPerPixel
                     sizeof(UInt8)*4*size.width, //bytesPerRow
                     colorSpace, //space
                     kCGBitmapByteOrder32Big, //bitmapInfo
                     dp, //provider
                     nil, //decode
                     NO, //shouldInterpolaten
                     kCGRenderingIntentDefault); //intent
    
    CGImageRetain(currentCGImage);
    return currentCGImage;
}

- (void)setPixel:(CGPoint)point r:(UInt8)r g:(UInt8)g b:(UInt8)b a:(UInt8)a
{
    int x = point.x;
    int y = point.y;
    int index = 4*(x+y*size.width);
    data[index] = r;
    data[index+1] = g;
    data[index+2] = b;
    data[index+3] = a;
}

- (void)setData:(UInt8*)indata
{
    data = indata;
}

- (CGSize)size
{
    return size;
}

@end
