// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// VERY IMPORTANT: after making any changes to this file, you need
// to run go generate in github.com/cogentcore/core/mobile and reinstall
// the core tool.

package org.golang.app;

import android.app.Activity;
import android.app.NativeActivity;
import android.content.Context;
import android.content.Intent;
import android.content.pm.ActivityInfo;
import android.content.pm.PackageManager;
import android.content.res.Configuration;
import android.graphics.Rect;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.text.Editable;
import android.text.InputType;
import android.text.TextWatcher;
import android.util.Log;
import android.view.Gravity;
import android.view.KeyCharacterMap;
import android.view.View;
import android.view.WindowInsets;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputMethodManager;
import android.view.KeyEvent;
import android.view.GestureDetector;
import android.view.ScaleGestureDetector;
import android.view.MotionEvent;
import android.widget.EditText;
import android.widget.FrameLayout;
import android.widget.TextView;
import android.widget.TextView.OnEditorActionListener;

public class GoNativeActivity extends NativeActivity {
	private static GoNativeActivity goNativeActivity;

	private static final int DEFAULT_INPUT_TYPE = InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS;

	private native void insetsChanged(int top, int bottom, int left, int right);

	private native void keyboardTyped(String str);

	private native void keyboardDelete();

	private native void setDarkMode(boolean dark);

	private native void scaled(float scaleFactor, float posX, float posY);

	private EditText mTextEdit;
	private boolean ignoreKey = false;

	public GoNativeActivity() {
		super();
		goNativeActivity = this;
	}

	String getTmpdir() {
		return getCacheDir().getAbsolutePath();
	}

	void updateLayout() {
		try {
			WindowInsets insets = getWindow().getDecorView().getRootWindowInsets();
			if (insets == null) {
				return;
			}

			insetsChanged(insets.getSystemWindowInsetTop(), insets.getSystemWindowInsetBottom(),
					insets.getSystemWindowInsetLeft(), insets.getSystemWindowInsetRight());
		} catch (java.lang.NoSuchMethodError e) {
			Rect insets = new Rect();
			getWindow().getDecorView().getWindowVisibleDisplayFrame(insets);

			View view = findViewById(android.R.id.content).getRootView();
			insetsChanged(insets.top, view.getHeight() - insets.height() - insets.top,
					insets.left, view.getWidth() - insets.width() - insets.left);
		}
	}

	static void showKeyboard(int keyboardType) {
		goNativeActivity.doShowKeyboard(keyboardType);
	}

	void doShowKeyboard(final int keyboardType) {
		runOnUiThread(new Runnable() {
			@Override
			public void run() {
				int imeOptions = EditorInfo.IME_ACTION_DONE;
				int inputType = DEFAULT_INPUT_TYPE;
				switch (keyboardType) {
					case 1:
						break;
					case 2:
						imeOptions = EditorInfo.IME_FLAG_NO_ENTER_ACTION;
						break;
					case 3:
						inputType |= InputType.TYPE_CLASS_NUMBER | InputType.TYPE_NUMBER_VARIATION_NORMAL;
						break;
					case 4:
						inputType |= InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD;
						break;
					case 5:
						inputType |= InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_EMAIL_ADDRESS;
						break;
					case 6:
						inputType |= InputType.TYPE_CLASS_PHONE;
						break;
					case 7:
						inputType |= InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_URI;
						break;
					default:
						Log.e("Go", "unknown keyboard type, use default");
				}
				mTextEdit.setImeOptions(imeOptions);
				mTextEdit.setInputType(inputType);

				mTextEdit.setOnEditorActionListener(new OnEditorActionListener() {
					@Override
					public boolean onEditorAction(TextView v, int actionId, KeyEvent event) {
						if (actionId == EditorInfo.IME_ACTION_DONE) {
							keyboardTyped("\n");
						}
						return false;
					}
				});

				// always place one character so all keyboards can send backspace
				ignoreKey = true;
				mTextEdit.setText("0");
				mTextEdit.setSelection(mTextEdit.getText().length());
				ignoreKey = false;

				mTextEdit.setVisibility(View.VISIBLE);
				mTextEdit.bringToFront();
				mTextEdit.requestFocus();

				InputMethodManager m = (InputMethodManager) getSystemService(Context.INPUT_METHOD_SERVICE);
				m.showSoftInput(mTextEdit, 0);
			}
		});
	}

	static void hideKeyboard() {
		goNativeActivity.doHideKeyboard();
	}

	void doHideKeyboard() {
		InputMethodManager imm = (InputMethodManager) getSystemService(Context.INPUT_METHOD_SERVICE);
		View view = findViewById(android.R.id.content).getRootView();
		imm.hideSoftInputFromWindow(view.getWindowToken(), 0);

		runOnUiThread(new Runnable() {
			@Override
			public void run() {
				mTextEdit.setVisibility(View.GONE);
			}
		});
	}

	static int getRune(int deviceId, int keyCode, int metaState) {
		try {
			int rune = KeyCharacterMap.load(deviceId).get(keyCode, metaState);
			if (rune == 0) {
				return -1;
			}
			return rune;
		} catch (KeyCharacterMap.UnavailableException e) {
			return -1;
		} catch (Exception e) {
			Log.e("Go", "exception reading KeyCharacterMap", e);
			return -1;
		}
	}

	private void load() {
		// Interestingly, NativeActivity uses a different method
		// to find native code to execute, avoiding
		// System.loadLibrary. The result is Java methods
		// implemented in C with JNIEXPORT (and JNI_OnLoad) are not
		// available unless an explicit call to System.loadLibrary
		// is done. So we do it here, borrowing the name of the
		// library from the same AndroidManifest.xml metadata used
		// by NativeActivity.
		try {
			ActivityInfo ai = getPackageManager().getActivityInfo(
					getIntent().getComponent(), PackageManager.GET_META_DATA);
			if (ai.metaData == null) {
				Log.e("Go", "loadLibrary: no manifest metadata found");
				return;
			}
			String libName = ai.metaData.getString("android.app.lib_name");
			System.loadLibrary(libName);
		} catch (Exception e) {
			Log.e("Go", "loadLibrary failed", e);
		}
	}

	@Override
	public void onCreate(Bundle savedInstanceState) {
		load();
		super.onCreate(savedInstanceState);
		setupEntry();
		updateTheme(getResources().getConfiguration());

		View view = findViewById(android.R.id.content).getRootView();
		view.addOnLayoutChangeListener(new View.OnLayoutChangeListener() {
			public void onLayoutChange(View v, int left, int top, int right, int bottom,
					int oldLeft, int oldTop, int oldRight, int oldBottom) {
				GoNativeActivity.this.updateLayout();
			}
		});

		mScaleDetector = new ScaleGestureDetector(this, new ScaleGestureListener());
	}

	private void setupEntry() {
		runOnUiThread(new Runnable() {
			@Override
			public void run() {
				mTextEdit = new EditText(goNativeActivity);
				mTextEdit.setVisibility(View.GONE);
				mTextEdit.setInputType(DEFAULT_INPUT_TYPE);

				FrameLayout.LayoutParams mEditTextLayoutParams = new FrameLayout.LayoutParams(
						FrameLayout.LayoutParams.WRAP_CONTENT, FrameLayout.LayoutParams.WRAP_CONTENT);
				mTextEdit.setLayoutParams(mEditTextLayoutParams);
				addContentView(mTextEdit, mEditTextLayoutParams);

				// always place one character so all keyboards can send backspace
				mTextEdit.setText("0");
				mTextEdit.setSelection(mTextEdit.getText().length());

				mTextEdit.addTextChangedListener(new TextWatcher() {
					@Override
					public void onTextChanged(CharSequence s, int start, int before, int count) {
						if (ignoreKey) {
							return;
						}
						if (count > 0) {
							keyboardTyped(s.subSequence(start, start + count).toString());
						}
					}

					@Override
					public void beforeTextChanged(CharSequence s, int start, int count, int after) {
						if (ignoreKey) {
							return;
						}
						if (count > 0) {
							for (int i = 0; i < count; i++) {
								// send a backspace
								keyboardDelete();
							}
						}
					}

					@Override
					public void afterTextChanged(Editable s) {
						// always place one character so all keyboards can send backspace
						if (s.length() < 1) {
							ignoreKey = true;
							mTextEdit.setText("0");
							mTextEdit.setSelection(mTextEdit.getText().length());
							ignoreKey = false;
							return;
						}
					}
				});
			}
		});
	}

	@Override
	public void onConfigurationChanged(Configuration config) {
		super.onConfigurationChanged(config);
		updateTheme(config);
	}

	protected void updateTheme(Configuration config) {
		boolean dark = (config.uiMode & Configuration.UI_MODE_NIGHT_MASK) == Configuration.UI_MODE_NIGHT_YES;
		setDarkMode(dark);
	}

	private ScaleGestureDetector mScaleDetector;

	@Override
	public boolean onTouchEvent(MotionEvent event) {
		this.mScaleDetector.onTouchEvent(event);
		return super.onTouchEvent(event);
	}

	class ScaleGestureListener extends ScaleGestureDetector.SimpleOnScaleGestureListener {
		@Override
		public boolean onScale(ScaleGestureDetector detector) {
			scaled(detector.getScaleFactor(), detector.getFocusX(), detector.getFocusY());
			return true;
		}
	}
}
