package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#include <dispatch/dispatch.h>
#include <stdint.h>

extern void goDispatchBridge(uintptr_t handle);

static void run_on_main_sync(uintptr_t handle) {
	dispatch_sync(dispatch_get_main_queue(), ^{
		goDispatchBridge(handle);
	});
}
*/
import "C"
import "runtime/cgo"

//export goDispatchBridge
func goDispatchBridge(handle C.uintptr_t) {
	h := cgo.Handle(uintptr(handle))
	fn := h.Value().(func())
	h.Delete()
	fn()
}

// runOnMain schedules fn on the macOS main thread via GCD and blocks until it returns.
// Required because both NSStatusItem (systray) and WKWebView (webview) must run on the main thread.
func runOnMain(fn func()) {
	h := cgo.NewHandle(fn)
	C.run_on_main_sync(C.uintptr_t(h))
}
