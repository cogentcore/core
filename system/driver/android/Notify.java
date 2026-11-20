// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// VERY IMPORTANT: after making any changes to this file, you need
// to run go generate in cogentcore.org/core/cmd/mobile and reinstall
// the core tool.

package org.golang.app;

import android.app.AlarmManager;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.net.Uri;
import android.os.Build;
import java.util.HashSet;
import java.util.Set;

// Notify has the notifications of the app (see system/driver/android/notify.go):
// they are shown with the NotificationManager, and scheduled with the
// AlarmManager, whose alarms are received by NotifyReceiver, which shows them.
// A click on a notification opens the GoNativeActivity with the ID of the
// notification in the extra EXTRA_ID.
public class Notify {
	// the ID of the notification channel of the app
	static final String CHANNEL = "notifications";

	// the extras of the intents, with the notification
	static final String EXTRA_ID = "org.cogentcore.notification.id";
	static final String EXTRA_TITLE = "org.cogentcore.notification.title";
	static final String EXTRA_BODY = "org.cogentcore.notification.body";

	// the shared preferences with the IDs of the scheduled notifications,
	// so that Sync can cancel them after the app is started again
	static final String PREFS = "org.cogentcore.notify";
	static final String PREFS_SCHEDULED = "scheduled";

	// ensureChannel makes the notification channel of the app, which is
	// needed for notifications on Android 8 and later. Its importance is high,
	// so that they pop up, as for reminders.
	static void ensureChannel(Context ctx) {
		if (Build.VERSION.SDK_INT < 26) {
			return;
		}
		NotificationManager nm = ctx.getSystemService(NotificationManager.class);
		if (nm.getNotificationChannel(CHANNEL) == null) {
			nm.createNotificationChannel(new NotificationChannel(CHANNEL, "Notifications", NotificationManager.IMPORTANCE_HIGH));
		}
	}

	// post shows the given notification now, which opens the activity
	// when it is clicked.
	static void post(Context ctx, String id, String title, String body) {
		ensureChannel(ctx);
		Intent open = new Intent(ctx, GoNativeActivity.class);
		open.putExtra(EXTRA_ID, id);
		// an existing activity gets the intent with onNewIntent
		open.setFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP | Intent.FLAG_ACTIVITY_SINGLE_TOP);
		open.setData(idUri(id)); // distinct intents for the IDs
		PendingIntent pi = PendingIntent.getActivity(ctx, 0, open,
				PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
		Notification.Builder b = Build.VERSION.SDK_INT >= 26 ? new Notification.Builder(ctx, CHANNEL)
				: new Notification.Builder(ctx);
		b.setSmallIcon(ctx.getApplicationInfo().icon)
				.setContentTitle(title)
				.setContentText(body)
				.setContentIntent(pi)
				.setAutoCancel(true);
		NotificationManager nm = ctx.getSystemService(NotificationManager.class);
		nm.notify(id, 0, b.build()); // the ID is the tag
	}

	// idUri returns the URI that makes the intents of the notification
	// with the given ID distinct, as their extras are not compared.
	static Uri idUri(String id) {
		return Uri.parse("cogentcore-notification:" + Uri.encode(id));
	}

	// alarm returns the pending intent of the alarm of the notification
	// with the given ID, to schedule it with the given title and body, or
	// to cancel it, for which they are not needed.
	static PendingIntent alarm(Context ctx, String id, String title, String body) {
		Intent in = new Intent(ctx, NotifyReceiver.class);
		in.setData(idUri(id));
		in.putExtra(EXTRA_ID, id);
		in.putExtra(EXTRA_TITLE, title);
		in.putExtra(EXTRA_BODY, body);
		return PendingIntent.getBroadcast(ctx, 0, in,
				PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
	}

	// show shows the given notification now (called from Go).
	static void show(String id, String title, String body) {
		post(GoNativeActivity.context(), id, title, body);
	}

	// sync schedules the given notifications at the given times, in Unix
	// milliseconds, replacing the ones scheduled before (called from Go).
	// The alarms are exact if the app is allowed to use exact alarms,
	// and otherwise inexact, which the system can delay for a while.
	static void sync(String[] ids, String[] titles, String[] bodies, long[] times) {
		Context ctx = GoNativeActivity.context();
		AlarmManager am = ctx.getSystemService(AlarmManager.class);
		SharedPreferences prefs = ctx.getSharedPreferences(PREFS, Context.MODE_PRIVATE);
		for (String id : prefs.getStringSet(PREFS_SCHEDULED, new HashSet<String>())) {
			am.cancel(alarm(ctx, id, "", ""));
		}
		boolean exact = Build.VERSION.SDK_INT < 31 || am.canScheduleExactAlarms();
		Set<String> scheduled = new HashSet<String>();
		for (int i = 0; i < ids.length; i++) {
			PendingIntent pi = alarm(ctx, ids[i], titles[i], bodies[i]);
			if (exact) {
				am.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, times[i], pi);
			} else {
				am.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, times[i], pi);
			}
			scheduled.add(ids[i]);
		}
		prefs.edit().putStringSet(PREFS_SCHEDULED, scheduled).apply();
	}

	// allowed returns whether the app can show notifications.
	static boolean allowed(Context ctx) {
		return ctx.getSystemService(NotificationManager.class).areNotificationsEnabled();
	}
}
