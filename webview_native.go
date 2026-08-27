package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework WebKit
#include <stdlib.h>
#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

static NSWindow   *g_window   = nil;
static WKWebView  *g_webview  = nil;

@interface DovinDelegate : NSObject <NSWindowDelegate, WKNavigationDelegate, WKUIDelegate>
@end
@implementation DovinDelegate
- (BOOL)windowShouldClose:(NSWindow *)sender {
    [sender orderOut:nil];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    return NO;
}
- (void)windowDidResignKey:(NSNotification *)notification {
    [g_window orderOut:nil];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}
- (void)webView:(WKWebView *)webView didFinishNavigation:(WKNavigation *)navigation {
    [webView evaluateJavaScript:@"document.querySelector('input[type=text],input:not([type])')?.focus()"
              completionHandler:nil];
}
- (void)webView:(WKWebView *)webView
    runJavaScriptAlertPanelWithMessage:(NSString *)message
                      initiatedByFrame:(WKFrameInfo *)frame
                     completionHandler:(void (^)(void))completionHandler {
    NSAlert *alert = [[NSAlert alloc] init];
    [alert setMessageText:message];
    [alert runModal];
    completionHandler();
}
- (void)webView:(WKWebView *)webView
    runJavaScriptConfirmPanelWithMessage:(NSString *)message
                        initiatedByFrame:(WKFrameInfo *)frame
                       completionHandler:(void (^)(BOOL))completionHandler {
    NSAlert *alert = [[NSAlert alloc] init];
    [alert setMessageText:message];
    [alert addButtonWithTitle:@"OK"];
    [alert addButtonWithTitle:@"Cancel"];
    completionHandler([alert runModal] == NSAlertFirstButtonReturn);
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
            [g_webview setNavigationDelegate:g_delegate];
            [g_webview setUIDelegate:g_delegate];
            [g_webview setAutoresizingMask:NSViewWidthSizable | NSViewHeightSizable];
            [[g_window contentView] addSubview:g_webview];
        }

        NSURLRequest *req = [NSURLRequest requestWithURL:[NSURL URLWithString:urlStr]];
        [g_webview loadRequest:req];
        [g_window setLevel:NSFloatingWindowLevel];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        [NSApp activateIgnoringOtherApps:YES];
        [g_window orderFrontRegardless];
        [g_window makeKeyWindow];
        [g_window makeFirstResponder:g_webview];
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
