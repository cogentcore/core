// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

// Notifications with the UserNotifications framework, whose delegate is
// the app delegate (see ios.m). This is as for macOS (see
// system/driver/desktop/notify_darwin.m).

#include "_cgo_export.h"

#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>

// the prefixes of the identifiers of the requests for notifications shown
// now and scheduled, so that Sync only removes the scheduled ones, and
// a shown one does not replace a scheduled one with the same ID
static NSString* const showPrefix = @"show.";
static NSString* const schedPrefix = @"sched.";

// notifyRequestPermission asks for permission, sending the answer to Go
// with the given handle of the function.
void notifyRequestPermission(uintptr_t h) {
	UNAuthorizationOptions opts = UNAuthorizationOptionAlert | UNAuthorizationOptionSound;
	[[UNUserNotificationCenter currentNotificationCenter] requestAuthorizationWithOptions:opts
		completionHandler:^(BOOL granted, NSError* error) {
			if (error != nil) {
				NSLog(@"system/driver/ios: notification permission: %@", error);
			}
			iosNotifyPermission(h, granted);
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
				NSLog(@"system/driver/ios: notification not added: %@", error);
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
// calendar, so that it is shown at that time after the device sleeps.
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
