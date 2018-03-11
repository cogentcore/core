typedef struct GMDCursors_ {
	/* these should be NSCursor* but #import <Cocoa/Cocoa.h> doesn't
	** seem to work within a .h file. `go build` hits 25,000 errors parsing
	** system headers if you try, hence the void* hackery. */
	void* arrow;
	void* resizeUp;
	void* resizeRight;
	void* resizeDown;
	void* resizeLeft;
	void* resizeLeftRight;
	void* resizeUpDown;
	void* pointingHand;
	void* crosshair;
	void* IBeam;
	void* openHand;
	void* closedHand;
	void* operationNotAllowed;
} GMDCursors;

extern GMDCursors cursors;

void initMacCursor();

void setCursor(void* c);
