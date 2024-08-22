// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

#include "_cgo_export.h"
#include <pthread.h>
#include <stdio.h>
#include <sys/utsname.h>

#import <UIKit/UIKit.h>
#import <MobileCoreServices/MobileCoreServices.h>
#import <UserNotifications/UserNotifications.h>

struct utsname sysInfo;

@interface GoAppAppController : UIViewController<UIContentContainer>
@end

@interface GoAppView : UIView
@end

@interface GoInputView : UITextField<UITextFieldDelegate>
@end

@interface GoAppAppDelegate : UIResponder<UIApplicationDelegate>
@property (strong, nonatomic) UIWindow *window;
@property (strong, nonatomic) GoAppAppController *controller;
@end

@implementation GoAppAppDelegate
- (BOOL)application:(UIApplication *)application didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
    int scale = 1;
    if ([[UIScreen mainScreen] respondsToSelector:@selector(displayLinkWithTarget:selector:)]) {
		scale = (int)[UIScreen mainScreen].scale; // either 1.0, 2.0, or 3.0.
	}
    CGSize size = [UIScreen mainScreen].nativeBounds.size;
    setDisplayMetrics((int)size.width, (int)size.height, scale);

	lifecycleAlive();
	self.window = [[UIWindow alloc] initWithFrame:[[UIScreen mainScreen] bounds]];
	self.controller = [[GoAppAppController alloc] initWithNibName:nil bundle:nil];
	self.window.rootViewController = self.controller;
	[self.window makeKeyAndVisible];

    // update insets once key window is set
	UIInterfaceOrientation orientation = [[UIApplication sharedApplication] statusBarOrientation];
	updateConfig((int)size.width, (int)size.height, orientation);

	UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
	center.delegate = self;

	return YES;
}

- (void)applicationDidBecomeActive:(UIApplication * )application {
	lifecycleFocused();
}

- (void)applicationWillResignActive:(UIApplication *)application {
	lifecycleVisible();
}

- (void)applicationDidEnterBackground:(UIApplication *)application {
	lifecycleAlive();
}

- (void)applicationWillTerminate:(UIApplication *)application {
	lifecycleDead();
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
	completionHandler(UNNotificationPresentationOptionAlert);
}
@end

@interface GoAppAppController ()
@property (strong, nonatomic) GoInputView *inputView;
@end

@implementation GoAppAppController
// TODO(kai): figure out what to do with this
// - (void)viewWillAppear:(BOOL)animated
// {
//	// TODO: replace by swapping out GLKViewController for a UIVIewController.
//	[super viewWillAppear:animated];
//	self.paused = YES;
//}

- (void) loadView {
	self.view = [[GoAppView alloc] initWithFrame:CGRectMake(0, 0, 400, 400)];
}

- (void)viewDidLoad {
	[super viewDidLoad];
	
	self.inputView = [[GoInputView alloc] initWithFrame:CGRectMake(0, 0, 0, 0)];
	self.inputView.delegate = self.inputView;
	self.inputView.autocapitalizationType = UITextAutocapitalizationTypeNone;
	self.inputView.autocorrectionType = UITextAutocorrectionTypeNo;
	[self.view addSubview:self.inputView];

	self.view.contentScaleFactor = UIScreen.mainScreen.nativeScale;
	self.view.multipleTouchEnabled = true;
	self.view.userInteractionEnabled = YES;
	
	// TODO(kai): figure out to what to do with this
	// self.paused = YES;
	// self.resumeOnDidBecomeActive = NO;
	// self.preferredFramesPerSecond = 0;

	int scale = 1;
	if ([[UIScreen mainScreen] respondsToSelector:@selector(displayLinkWithTarget:selector:)]) {
		scale = (int)[UIScreen mainScreen].scale; // either 1.0, 2.0, or 3.0.
	}
	setScreen(scale);

	CGSize size = [UIScreen mainScreen].nativeBounds.size;
	UIInterfaceOrientation orientation = [[UIApplication sharedApplication] statusBarOrientation];
	updateConfig((int)size.width, (int)size.height, orientation);

    setWindowPtr([self.view layer]);
	
    CADisplayLink* displayLink = [CADisplayLink displayLinkWithTarget:self selector:@selector(render:)];
    [displayLink addToRunLoop:[NSRunLoop currentRunLoop] forMode:NSDefaultRunLoopMode];

	UIPinchGestureRecognizer* pinchGesture = [[UIPinchGestureRecognizer alloc] init];
	pinchGesture.delegate = self;
	[pinchGesture addTarget:self action: @selector(onPinch:)];
	[self.view addGestureRecognizer:pinchGesture];
}

- (void)viewWillTransitionToSize:(CGSize)ptSize withTransitionCoordinator:(id<UIViewControllerTransitionCoordinator>)coordinator {
	[coordinator animateAlongsideTransition:^(id<UIViewControllerTransitionCoordinatorContext> context) {
		// TODO(crawshaw): come up with a plan to handle animations.
	} completion:^(id<UIViewControllerTransitionCoordinatorContext> context) {
		UIInterfaceOrientation orientation = [[UIApplication sharedApplication] statusBarOrientation];
		CGSize size = [UIScreen mainScreen].nativeBounds.size;
		updateConfig((int)size.width, (int)size.height, orientation);
	}];
}

- (void) render:(CADisplayLink*)displayLink {
   // [self.view display]; // todo: seems unnecessary?
}

#define TOUCH_TYPE_BEGIN 0 // system.TouchStart
#define TOUCH_TYPE_MOVE  1 // touch.TouchMove
#define TOUCH_TYPE_END   2 // touch.TouchEnd

static void sendTouches(int change, NSSet* touches) {
	CGFloat scale = [UIScreen mainScreen].nativeScale;
	for (UITouch* touch in touches) {
		CGPoint p = [touch locationInView:touch.view];
		sendTouch((GoUintptr)touch, (GoUintptr)change, p.x*scale, p.y*scale);
	}
}

- (void)touchesBegan:(NSSet*)touches withEvent:(UIEvent*)event {
	sendTouches(TOUCH_TYPE_BEGIN, touches);
}

- (void)touchesMoved:(NSSet*)touches withEvent:(UIEvent*)event {
	sendTouches(TOUCH_TYPE_MOVE, touches);
}

- (void)touchesEnded:(NSSet*)touches withEvent:(UIEvent*)event {
	sendTouches(TOUCH_TYPE_END, touches);
}

- (void)touchesCanceled:(NSSet*)touches withEvent:(UIEvent*)event {
    sendTouches(TOUCH_TYPE_END, touches);
}

- (void) traitCollectionDidChange: (UITraitCollection *) previousTraitCollection {
    [super traitCollectionDidChange: previousTraitCollection];

	UIInterfaceOrientation orientation = [[UIApplication sharedApplication] statusBarOrientation];
	CGSize size = [UIScreen mainScreen].nativeBounds.size;
	updateConfig((int)size.width, (int)size.height, orientation);
}

- (void) onPinch: (UIPinchGestureRecognizer *)pinchRecognizer {
	// if (gestureRecognizer.state == .began) {
		[self becomeFirstResponder];
		// self.viewForReset = gestureRecognizer.view;
		CGFloat scale = pinchRecognizer.scale;
		CGPoint location = [pinchRecognizer locationInView:pinchRecognizer.view];
		scaled(scale, location.x, location.y);
	// }
}

@end

#pragma mark -
#pragma mark GoAppView

@implementation GoAppView

/** Returns a Metal-compatible layer. */
+(Class) layerClass { return [CAMetalLayer class]; }

@end

@implementation GoInputView

- (BOOL)canBecomeFirstResponder {
    return YES;
}

- (void)deleteBackward {
    keyboardDelete();
}

-(BOOL)textField:(UITextField *)textField shouldChangeCharactersInRange:(NSRange)range replacementString:(NSString *)string {
    keyboardTyped([string UTF8String]);
    return NO;
}

@end

void runApp(void) {
	char * argv[] = {};
	@autoreleasepool {
		UIApplicationMain(0, argv, nil, NSStringFromClass([GoAppAppDelegate class]));
	}
}

uint64_t threadID() {
	uint64_t id;
	if (pthread_threadid_np(pthread_self(), &id)) {
		abort();
	}
	return id;
}

UIEdgeInsets getDevicePadding() {
    if (@available(iOS 11.0, *)) {
        UIWindow *window = UIApplication.sharedApplication.keyWindow;

        return window.safeAreaInsets;
    }

    return UIEdgeInsetsZero;
}

bool isDark() {
    UIViewController *rootVC = [[[[UIApplication sharedApplication] delegate] window] rootViewController];
    return rootVC.traitCollection.userInterfaceStyle == UIUserInterfaceStyleDark;
}

void showKeyboard(int keyboardType) {
    GoAppAppDelegate *appDelegate = (GoAppAppDelegate *)[[UIApplication sharedApplication] delegate];
    GoInputView *view = appDelegate.controller.inputView;

    dispatch_async(dispatch_get_main_queue(), ^{
        switch (keyboardType)
        {
			case 1:
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
            case 2:
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDefault];
                break;
            case 3:
                [view setKeyboardType:UIKeyboardTypeDecimalPad];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
			case 4:
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
			case 5:
                [view setKeyboardType:UIKeyboardTypeEmailAddress];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
			case 6:
                [view setKeyboardType:UIKeyboardTypePhonePad];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
			case 7:
                [view setKeyboardType:UIKeyboardTypeURL];
                [view setReturnKeyType:UIReturnKeyDone];
                break;
            default:
                NSLog(@"unknown keyboard type, use default");
                [view setKeyboardType:UIKeyboardTypeDefault];
                [view setReturnKeyType:UIReturnKeyDefault];
                break;
        }
        // refresh settings if keyboard is already open
        [view reloadInputViews];

        BOOL ret = [view becomeFirstResponder];
    });
}

void hideKeyboard() {
    GoAppAppDelegate *appDelegate = (GoAppAppDelegate *)[[UIApplication sharedApplication] delegate];
    GoInputView *view = appDelegate.controller.inputView;

    dispatch_async(dispatch_get_main_queue(), ^{
        [view resignFirstResponder];
    });
}

void setClipboardContent(char *content) {
    NSString *value = [NSString stringWithUTF8String:content];
    [[UIPasteboard generalPasteboard] setString:value];
}

char *getClipboardContent() {
    NSString *str = [[UIPasteboard generalPasteboard] string];

    return [str UTF8String];
}
