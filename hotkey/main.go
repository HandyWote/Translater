package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	MOD_ALT = 0x0001
	VK_T    = 0x54
	WM_HOTKEY = 0x0312
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	registerHotKey   = user32.NewProc("RegisterHotKey")
	unregisterHotKey = user32.NewProc("UnregisterHotKey")
	getMessage       = user32.NewProc("GetMessageW")
	translateMessage = user32.NewProc("TranslateMessage")
	dispatchMessage  = user32.NewProc("DispatchMessageW")
)

type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

func main() {
	fmt.Println("热键监听程序已启动...")
	fmt.Println("按下Alt+T打印'ok'，按Ctrl+C退出程序")

	// 注册热键 Alt+T
	ret, _, err := registerHotKey.Call(
		0,           // HWND (0 = global)
		1,           // id
		MOD_ALT,     // fsModifiers
		VK_T,        // vk
	)
	if ret == 0 {
		fmt.Printf("注册热键失败: %v\n", err)
		return
	}
	defer unregisterHotKey.Call(0, 1)

	// 消息循环
	var msg MSG
	for {
		// 获取消息
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break // WM_QUIT
		}

		// 处理热键消息
		if msg.Message == WM_HOTKEY && msg.WParam == 1 {
			fmt.Println("ok")
		}

		// 翻译和分发消息
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}