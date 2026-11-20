// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin && !ios

// Notifications with the UserNotifications framework, which only works for
// an app in an app bundle (see notifyAvailable).

#include "_cgo_export.h"

#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

// the prefixes of the identifiers of the requests for notifications shown
// now and scheduled, so that Sync only removes the scheduled ones, and
// a shown one does not replace a scheduled one with the same ID
static NSString* const showPrefix = @"show.";
static NSString* const schedPrefix = @"sched.";

@interface CogentNotifyDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation CogentNotifyDelegate

// the notifications are shown while the app is in front too
- (void)userNotificationCenter:(UNUserNotificationCenter*)center
       willPresentNotification:(UNNotification*)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
	completionHandler(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList | UNNotificationPresentationOptionSound);
}

// a click on the notification is sent to Go with its ID
- (void)userNotificationCenter:(UNUserNotificationCenter*)center
didReceiveNotificationResponse:(UNNotificationResponse*)response
         withCompletionHandler:(void (^)(void))completionHandler {
	if ([response.actionIdentifier isEqualToString:UNNotificationDefaultActionIdentifier]) {
		NSString* nid = response.notification.request.content.userInfo[@"id"];
		if (nid != nil) {
			macNotifyClicked((char*)[nid UTF8String]);
		}
	}
	completionHandler();
}

@end

static CogentNotifyDelegate* notifyDelegate;

// notifyAvailable returns whether notifications can be used: the
// framework raises an exception for an executable that is not in an app
// bundle, as for go run.
bool notifyAvailable(void) {
	NSBundle* b = [NSBundle mainBundle];
	return b.bundleIdentifier != nil && [b.bundlePath hasSuffix:@".app"];
}

// notifyInit sets the delegate of the notifications.
void notifyInit(void) {
	notifyDelegate = [[CogentNotifyDelegate alloc] init];
	[UNUserNotificationCenter currentNotificationCenter].delegate = notifyDelegate;
}

// notifyRequestPermission asks for permission, sending the answer to Go
// with the given handle of the function.
void notifyRequestPermission(uintptr_t h) {
	UNAuthorizationOptions opts = UNAuthorizationOptionAlert | UNAuthorizationOptionSound;
	[[UNUserNotificationCenter currentNotificationCenter] requestAuthorizationWithOptions:opts
		completionHandler:^(BOOL granted, NSError* error) {
			if (error != nil) {
				NSLog(@"system/driver/desktop: notification permission: %@", error);
			}
			macNotifyPermission(h, granted);
		}];
}

// notifyAdd adds the request for a notification with the given trigger,
// nil for now.
static void notifyAdd(NSString* prefix, char* nid, char* title, char* body, UNNotificationTrigger* trigger) {
	NSString* ids = [NSString stringWithUTF8String:nid];
	UNMutableNotificationContent* content = [[UNMutableNotificationContent new] autorelease];
	content.title = [NSString stringWithUTF8String:title];
	content.body = [NSString stringWithUTF8String:body];
	content.sound = [UNNotificationSound defaultSound];
	content.userInfo = @{@"id": ids};
	UNNotificationRequest* req = [UNNotificationRequest
		requestWithIdentifier:[prefix stringByAppendingString:ids] content:content trigger:trigger];
	[[UNUserNotificationCenter currentNotificationCenter] addNotificationRequest:req
		withCompletionHandler:^(NSError* error) {
			if (error != nil) {
				NSLog(@"system/driver/desktop: notification not added: %@", error);
			}
		}];
}

// notifyShow shows a notification now.
void notifyShow(char* nid, char* title, char* body) {
	@autoreleasepool {
		notifyAdd(showPrefix, nid, title, body, nil);
	}
}

// notifyClearScheduled removes the scheduled notifications that have not
// been shown yet.
void notifyClearScheduled(void) {
	@autoreleasepool {
		[[UNUserNotificationCenter currentNotificationCenter] removeAllPendingNotificationRequests];
	}
}

// notifySchedule schedules a notification at the given Unix time, in the
// calendar, so that it is shown at that time after the computer sleeps.
void notifySchedule(char* nid, char* title, char* body, double unix) {
	@autoreleasepool {
		NSDate* date = [NSDate dateWithTimeIntervalSince1970:unix];
		NSCalendarUnit units = NSCalendarUnitYear | NSCalendarUnitMonth | NSCalendarUnitDay |
			NSCalendarUnitHour | NSCalendarUnitMinute | NSCalendarUnitSecond;
		NSDateComponents* comps = [[NSCalendar currentCalendar] components:units fromDate:date];
		UNCalendarNotificationTrigger* trigger = [UNCalendarNotificationTrigger
			triggerWithDateMatchingComponents:comps repeats:NO];
		notifyAdd(schedPrefix, nid, title, body, trigger);
	}
}
