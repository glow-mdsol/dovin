package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#include <stdlib.h>
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

static NSWindow   *g_window   = nil;
static WKWebView  *g_webview  = nil;

@interface DovinDelegate : NSObject <NSWindowDelegate>
@end
@implementation DovinDelegate
- (BOOL)windowShouldClose:(NSWindow *)sender {
    [sender orderOut:nil];
    return NO;
}
@end

static DovinDelegate *g_delegate = nil;

void dovin_show(const char *url) {
    NSString *urlStr = [[NSString alloc] initWithUTF8String:url];
    dispatch_async(dispatch_get_main_queue(), ^{
        if (g_window == nil) {
            NSRect frame = NSMakeRect(0, 0, 420, 600);
            NSWindowStyleMask style = NSWindowStyleMaskTitled |
                                      NSWindowStyleMaskClosable |
                                      NSWindowStyleMaskMiniaturizable;
            g_window = [[NSWindow alloc] initWithContentRect:frame
                                                   styleMask:style
                                                     backing:NSBackingStoreBuffered
                                                       defer:NO];
            [g_window setTitle:@"Dovin"];
            [g_window center];
            g_delegate = [[DovinDelegate alloc] init];
            [g_window setDelegate:g_delegate];

            WKWebViewConfiguration *cfg = [[WKWebViewConfiguration alloc] init];
            g_webview = [[WKWebView alloc]
                initWithFrame:[[g_window contentView] bounds]
                configuration:cfg];
            [g_webview setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
            [[g_window contentView] addSubview:g_webview];
        }

        NSURLRequest *req = [NSURLRequest requestWithURL:[NSURL URLWithString:urlStr]];
        [g_webview loadRequest:req];
        [g_window setLevel:NSFloatingWindowLevel];
        [NSApp activateIgnoringOtherApps:YES];
        [g_window orderFrontRegardless];
        [g_window makeKeyWindow];
    });
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

func openWebview(port int, fragment string) {
	url := fmt.Sprintf("http://localhost:%d/%s", port, fragment)
	cs := C.CString(url)
	defer C.free(unsafe.Pointer(cs))
	C.dovin_show(cs)
}
