#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

#include "reopen_darwin.h"

static BOOL handleReopen(id self, SEL _cmd, NSApplication *app, BOOL hasVisibleWindows) {
	[NSApp activateIgnoringOtherApps:YES];
	goReopen();
	return YES;
}

void InstallReopenHandler(void) {
	// NSApp's delegate (GLFW's) only exists once the driver has started, and AppKit objects belong to the main thread.
	dispatch_async(dispatch_get_main_queue(), ^{
		id delegate = [NSApp delegate];
		if (delegate == nil) {
			return;
		}
		Class cls = object_getClass(delegate);
		SEL sel = @selector(applicationShouldHandleReopen:hasVisibleWindows:);
		if (!class_addMethod(cls, sel, (IMP)handleReopen, "c@:@c")) {
			// The delegate already implements it: replace that.
			method_setImplementation(class_getInstanceMethod(cls, sel), (IMP)handleReopen);
		}
	});
}
