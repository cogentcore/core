//
//  gmd.h
//  gomacdraw
//
//  Created by John Asmuth on 5/9/11.
//  Copyright 2011 Rutgers University. All rights reserved.
//

enum GMDErrorCodes {
    GMDNoError = 0,
    GMDLoadNibError = -1,
};

enum GMDEventCodes {
    GMDNoop = 0,
    GMDMouseDown = 1,
    GMDMouseUp = 2,
    GMDMouseDragged = 3,
    GMDMouseMoved = 4,
    GMDMouseEntered = 5,
    GMDMouseExited = 6,
    GMDKeyDown = 7,
    GMDKeyUp = 8,
    //GMDKeyPress = 9,
    GMDResize = 10,
    GMDClose = 11,
    GMDKeyFocus = 12, // got keyboard focus
    GMDMainFocus = 13, // became "main" window
    GMDMagnify = 14,
    GMDRotate = 15,
    GMDScroll = 16,
    GMDMouseWheel = 17,
};

typedef void* GMDWindow;
typedef void* GMDImage;

typedef struct {
    int kind;
    int data[5];
} GMDEvent;

int initMacDraw();
void releaseMacDraw();

void NSAppRun();
void NSAppStop();

int isMainThread();
void taskReady();

GMDWindow openWindow();
int closeWindow(GMDWindow gmdw);

void showWindow(GMDWindow gmdw);
void hideWindow(GMDWindow gmdw);

void setWindowTitle(GMDWindow gmdw, char* title);
void setWindowSize(GMDWindow gmdw, int width, int height);
void getWindowSize(GMDWindow gmdw, int* width, int* height);

GMDEvent getNextEvent(GMDWindow gmdw);

GMDImage getWindowScreen(GMDWindow gmdw);
void flushWindowScreen(GMDWindow gmdw);

void setScreenPixel(GMDImage screen, int x, int y, int r, int g, int b, int a);
void getScreenSize(GMDImage screen, int* width, int* height);

void setScreenData(GMDImage screen, void* data);
