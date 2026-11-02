// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && !ios

#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>

// errString returns a malloced copy of the description of the given error.
static char* errString(NSError* err) {
	return strdup([[err localizedDescription] UTF8String]);
}

// appName returns the name of the app at the given path, as shown in the
// Finder, without the .app extension, which it shows when set to show all
// extensions.
static NSString* appName(NSString* path) {
	NSString* name = [[NSFileManager defaultManager] displayNameAtPath:path];
	if ([name hasSuffix:@".app"]) {
		name = [name stringByDeletingPathExtension];
	}
	return name;
}

// appsForFile returns the apps that can open the file at the given path,
// one per line, as the path of the app, a tab, and its name, with the
// default app first, marked with a * at the start. The result is malloced.
char* appsForFile(const char* path) {
	@autoreleasepool {
		NSURL* url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSWorkspace* ws = [NSWorkspace sharedWorkspace];
		NSURL* def = [ws URLForApplicationToOpenURL:url];
		NSArray<NSURL*>* all = @[];
		if (@available(macOS 12.0, *)) {
			all = [ws URLsForApplicationsToOpenURL:url];
		}
		NSMutableString* out = [NSMutableString string];
		if (def != nil) {
			[out appendFormat:@"*%@\t%@\n", def.path, appName(def.path)];
		}
		for (NSURL* app in all) {
			if (def != nil && [app.path isEqualToString:def.path]) {
				continue;
			}
			[out appendFormat:@"%@\t%@\n", app.path, appName(app.path)];
		}
		return strdup([out UTF8String]);
	}
}

// waitDone waits up to 10 seconds for the given semaphore, which the
// completion handlers of NSWorkspace signal, returning an error message
// if it times out.
static char* waitDone(dispatch_semaphore_t done) {
	if (dispatch_semaphore_wait(done, dispatch_time(DISPATCH_TIME_NOW, 10 * NSEC_PER_SEC)) != 0) {
		return strdup("timed out");
	}
	return NULL;
}

// openWithApp opens the file at the given path with the app at the given
// path, returning a malloced error message, or NULL.
char* openWithApp(const char* path, const char* app) {
	@autoreleasepool {
		NSURL* url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSURL* appURL = [NSURL fileURLWithPath:[NSString stringWithUTF8String:app]];
		__block char* msg = NULL;
		dispatch_semaphore_t done = dispatch_semaphore_create(0);
		[[NSWorkspace sharedWorkspace] openURLs:@[url] withApplicationAtURL:appURL
			configuration:[NSWorkspaceOpenConfiguration configuration]
			completionHandler:^(NSRunningApplication* ra, NSError* err) {
				if (err != nil) {
					msg = errString(err);
				}
				dispatch_semaphore_signal(done);
			}];
		char* tm = waitDone(done);
		return tm != NULL ? tm : msg;
	}
}

// setDefaultApp makes the app at the given path the default app for the
// content type of the file at the given path, returning a malloced error
// message, or NULL.
char* setDefaultApp(const char* path, const char* app) {
	@autoreleasepool {
		if (@available(macOS 12.0, *)) {
			NSURL* url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
			NSURL* appURL = [NSURL fileURLWithPath:[NSString stringWithUTF8String:app]];
			UTType* type = nil;
			NSError* err = nil;
			if (![url getResourceValue:&type forKey:NSURLContentTypeKey error:&err] || type == nil) {
				return err != nil ? errString(err) : strdup("unknown type of file");
			}
			__block char* msg = NULL;
			dispatch_semaphore_t done = dispatch_semaphore_create(0);
			[[NSWorkspace sharedWorkspace] setDefaultApplicationAtURL:appURL toOpenContentType:type
				completionHandler:^(NSError* err) {
					if (err != nil) {
						msg = errString(err);
					}
					dispatch_semaphore_signal(done);
				}];
			char* tm = waitDone(done);
			return tm != NULL ? tm : msg;
		}
		return strdup("requires macOS 12 or later");
	}
}
