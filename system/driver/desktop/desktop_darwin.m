// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build 386 amd64 arm64
// +build !ios

#include "_cgo_export.h"
#include <stdio.h>

#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <AppKit/AppKit.h>
#import <objc/runtime.h>
#import <sys/qos.h>
#import <pthread/qos.h>
//#import <IOKit/graphics/IOGraphicsLib.h>

int setThreadPri(double p) {
	return pthread_set_qos_class_self_np(QOS_CLASS_USER_INTERACTIVE,0);
// return setpriority(PRIO_PROCESS, 0, -20);
// [NSThread setThreadPriority:p];
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
	if(ns_clip == NULL) {
		return;
	}
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
	// NSLog(@"in open file %@\n", filename);
	const char* utf_fn = filename.UTF8String;
   int flen = (int)strlen(utf_fn);
	macOpenFile((char*)utf_fn, flen);
	return true;
}

- (void)swz_application:(NSApplication *)sender openFiles:(NSArray<NSString *> *)filenames{
	int n = [filenames count];
	// NSLog(@"in open files: %d nfiles\n", n);
	int i;
	for (i=0; i<n; i++) {
		NSString* fnm = [filenames objectAtIndex: i];
		const char* utf_fn = fnm.UTF8String;
		int flen = (int)strlen(utf_fn);
		// NSLog(@"open file: %@\n", fnm);
		macOpenFile((char*)utf_fn, flen);
		// NSLog(@"done file\n");
	}
}

@end

