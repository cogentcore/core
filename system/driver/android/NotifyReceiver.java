// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// VERY IMPORTANT: after making any changes to this file, you need
// to run go generate in cogentcore.org/core/cmd/mobile and reinstall
// the core tool.

package org.golang.app;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

// NotifyReceiver receives the alarms of the scheduled notifications
// (see Notify.sync), and shows them, also when the app is not running.
// It is declared in the manifest of the app.
public class NotifyReceiver extends BroadcastReceiver {
	@Override
	public void onReceive(Context ctx, Intent in) {
		String id = in.getStringExtra(Notify.EXTRA_ID);
		if (id == null) {
			return;
		}
		Notify.post(ctx, id, in.getStringExtra(Notify.EXTRA_TITLE), in.getStringExtra(Notify.EXTRA_BODY));
	}
}
