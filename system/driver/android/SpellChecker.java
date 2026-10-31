// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// VERY IMPORTANT: after making any changes to this file, you need
// to run go generate in github.com/cogentcore/core/mobile and reinstall
// the core tool.

package org.golang.app;

import android.content.Context;
import android.os.Handler;
import android.os.HandlerThread;
import android.util.Log;
import android.view.textservice.SentenceSuggestionsInfo;
import android.view.textservice.SpellCheckerSession;
import android.view.textservice.SuggestionsInfo;
import android.view.textservice.TextInfo;
import android.view.textservice.TextServicesManager;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

// SpellChecker provides synchronous access to the system spell checker
// for the text/spell package, which calls it through JNI from Go.
// The SpellCheckerSession API is asynchronous, delivering results to a
// listener on the thread that created the session, so the session is
// created on its own HandlerThread, and each request waits for its result.
public class SpellChecker implements SpellCheckerSession.SpellCheckerSessionListener {
	private static final long TIMEOUT_MS = 2000;

	// RETRY_MS is how long requests fail immediately after a timeout,
	// so that an unresponsive spell checker does not block every call.
	private static final long RETRY_MS = 10000;

	private SpellCheckerSession session;

	// lock protects the fields below. It is separate from the SpellChecker
	// monitor, which serializes requests and is held while waiting.
	private final Object lock = new Object();

	// cookie identifies the current request, to ignore late results
	// of earlier requests that timed out.
	private int cookie;

	// result is the result of the current request, and done
	// is counted down when it arrives.
	private SentenceSuggestionsInfo[] result;
	private CountDownLatch done;

	// failedUntil is the time until which requests fail, after a timeout.
	private long failedUntil;

	// create returns a new SpellChecker using the user's spell checker
	// and its language settings, or null if there is none or it is disabled.
	static SpellChecker create(final Context ctx) {
		final SpellChecker sc = new SpellChecker();
		HandlerThread thread = new HandlerThread("SpellChecker");
		thread.start();
		final CountDownLatch created = new CountDownLatch(1);
		new Handler(thread.getLooper()).post(new Runnable() {
			@Override
			public void run() {
				try {
					TextServicesManager tsm = (TextServicesManager) ctx.getSystemService(Context.TEXT_SERVICES_MANAGER_SERVICE);
					if (tsm != null) {
						sc.session = tsm.newSpellCheckerSession(null, null, sc, true);
					}
				} catch (Exception e) {
					Log.e("Go", "cannot create spell checker session", e);
				}
				created.countDown();
			}
		});
		try {
			created.await();
		} catch (InterruptedException e) {
			return null;
		}
		if (sc.session == null) {
			thread.quit();
			return null;
		}
		return sc;
	}

	// request checks the text, returning the result,
	// or null if the spell checker does not respond.
	private SentenceSuggestionsInfo[] request(String text, int suggestionsLimit) {
		CountDownLatch d = new CountDownLatch(1);
		int c;
		synchronized (lock) {
			if (System.currentTimeMillis() < failedUntil) {
				return null;
			}
			c = ++cookie;
			result = null;
			done = d;
		}
		session.getSentenceSuggestions(new TextInfo[] { new TextInfo(text, 0, text.length(), c, c) }, suggestionsLimit);
		boolean ok = false;
		try {
			ok = d.await(TIMEOUT_MS, TimeUnit.MILLISECONDS);
		} catch (InterruptedException e) {
		}
		synchronized (lock) {
			done = null;
			if (!ok) {
				Log.e("Go", "spell checker did not respond");
				failedUntil = System.currentTimeMillis() + RETRY_MS;
			}
			return result;
		}
	}

	// isMisspelled returns whether the attributes are for a misspelled word,
	// using the same test as android.widget.SpellChecker.
	private static boolean isMisspelled(SuggestionsInfo si) {
		int attrs = si.getSuggestionsAttributes();
		return (attrs & SuggestionsInfo.RESULT_ATTR_IN_THE_DICTIONARY) == 0
				&& (attrs & SuggestionsInfo.RESULT_ATTR_LOOKS_LIKE_TYPO) != 0;
	}

	// misspelled returns the sorted UTF-16 offsets of the misspelled
	// words in the text, or null if the spell checker does not respond.
	synchronized int[] misspelled(String text) {
		SentenceSuggestionsInfo[] res = request(text, 1);
		if (res == null) {
			return null;
		}
		ArrayList<Integer> offsets = new ArrayList<Integer>();
		for (SentenceSuggestionsInfo ssi : res) {
			if (ssi == null) {
				continue;
			}
			for (int i = 0; i < ssi.getSuggestionsCount(); i++) {
				SuggestionsInfo si = ssi.getSuggestionsInfoAt(i);
				if (si != null && ssi.getOffsetAt(i) >= 0 && isMisspelled(si)) {
					offsets.add(ssi.getOffsetAt(i));
				}
			}
		}
		int[] r = new int[offsets.size()];
		for (int i = 0; i < r.length; i++) {
			r[i] = offsets.get(i);
		}
		Arrays.sort(r);
		return r;
	}

	// suggest returns suggested corrections for the misspelled word.
	synchronized String[] suggest(String word) {
		SentenceSuggestionsInfo[] res = request(word, 10);
		ArrayList<String> sugs = new ArrayList<String>();
		if (res != null) {
			for (SentenceSuggestionsInfo ssi : res) {
				if (ssi == null) {
					continue;
				}
				for (int i = 0; i < ssi.getSuggestionsCount(); i++) {
					SuggestionsInfo si = ssi.getSuggestionsInfoAt(i);
					if (si == null) {
						continue;
					}
					for (int j = 0; j < si.getSuggestionsCount(); j++) {
						sugs.add(si.getSuggestionAt(j));
					}
				}
			}
		}
		return sugs.toArray(new String[0]);
	}

	@Override
	public void onGetSuggestions(SuggestionsInfo[] results) {
	}

	@Override
	public void onGetSentenceSuggestions(SentenceSuggestionsInfo[] results) {
		synchronized (lock) {
			if (done == null || resultCookie(results) != cookie) {
				return;
			}
			result = results;
			done.countDown();
		}
	}

	// resultCookie returns the cookie of the request that the results are for,
	// which the word results carry, or the current cookie if there are none.
	private int resultCookie(SentenceSuggestionsInfo[] results) {
		for (SentenceSuggestionsInfo ssi : results) {
			if (ssi != null && ssi.getSuggestionsCount() > 0 && ssi.getSuggestionsInfoAt(0) != null) {
				return ssi.getSuggestionsInfoAt(0).getCookie();
			}
		}
		return cookie;
	}
}
