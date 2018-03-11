#import <Cocoa/Cocoa.h>
#import "cursor.h"

GMDCursors cursors;

void initMacCursor() {
	cursors.arrow = [NSCursor arrowCursor];
	cursors.resizeUp = [NSCursor resizeUpCursor];
	cursors.resizeRight = [NSCursor resizeRightCursor];
	cursors.resizeDown = [NSCursor resizeDownCursor];
	cursors.resizeLeft = [NSCursor resizeLeftCursor];
	cursors.resizeLeftRight = [NSCursor resizeLeftRightCursor];
	cursors.resizeUpDown = [NSCursor resizeUpDownCursor];
	cursors.pointingHand = [NSCursor pointingHandCursor];
	cursors.crosshair = [NSCursor crosshairCursor];
	cursors.IBeam = [NSCursor IBeamCursor];
	cursors.openHand = [NSCursor openHandCursor];
	cursors.closedHand = [NSCursor closedHandCursor];
	cursors.operationNotAllowed = [NSCursor operationNotAllowedCursor];
}

void setCursor(void* c) {
	if (c != nil) {
		[((NSCursor*)c) set];
	}
}
