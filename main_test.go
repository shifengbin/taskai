package main

import "testing"

func TestApplicationOptionsKeepWebViewDropEnabledForNativeFileDrop(t *testing.T) {
	options := applicationOptions(&App{})
	if options.DragAndDrop == nil {
		t.Fatal("应用选项未配置文件拖放")
	}
	if !options.DragAndDrop.EnableFileDrop {
		t.Fatal("应用选项未启用原生文件拖放")
	}
	if options.DragAndDrop.DisableWebViewDrop {
		t.Fatal("启用原生文件拖放时不能禁用 WebView 拖放")
	}
}
