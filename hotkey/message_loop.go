//go:build windows
// +build windows

package hotkey

import (
	"syscall"
	"unsafe"
)

var (
	moduser32    = syscall.NewLazyDLL("user32.dll")
	createWindow = moduser32.NewProc("CreateWindowExW")
	destroyWindow = moduser32.NewProc("DestroyWindow")
	getMessage   = moduser32.NewProc("GetMessageW")
	translateMessage = moduser32.NewProc("TranslateMessage")
	dispatchMessage = moduser32.NewProc("DispatchMessageW")
	defWindowProc = moduser32.NewProc("DefWindowProcW")
)

const (
	wmHotkey = 0x0312
)

// createHiddenWindow creates a hidden window for receiving hotkey messages
func (hm *HotkeyManager) createHiddenWindow() error {
	// 注册窗口类
	wcname, err := syscall.UTF16PtrFromString("HotkeyWindow")
	if err != nil {
		return err
	}
	
	wc := &syscall.WndClassEx{
		ClsExtra:    0,
		WndExtra:    0,
		Instance:    0,
		Icon:        0,
		IconSm:      0,
		Cursor:      0,
		Background:  0,
		MenuName:    nil,
		ClassName:   wcname,
		WndProc:     syscall.NewCallback(hm.wndProc),
		Style:       0,
	}
	
	_, err = syscall.RegisterClassEx(wc)
	if err != nil {
		return err
	}
	
	// 创建隐藏窗口
	hwnd, err := syscall.CreateWindowEx(
		0,
		wcname,
		wcname,
		0,
		0,
		0,
		0,
		0,
		syscall.HWND_MESSAGE,
		0,
		0,
		0,
	)
	if err != nil {
		return err
	}
	
	hm.hwnd = hwnd
	return nil
}

// messageLoop starts the Windows message loop
func (hm *HotkeyManager) messageLoop() {
	var msg syscall.Msg
	for {
		select {
		case <-hm.quit:
			return
		default:
			ret, _, _ := getMessage.Call(
				uintptr(unsafe.Pointer(&msg)),
				0,
				0,
				0,
			)
			if ret == 0 {
				return
			}
			
			translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

// wndProc handles window messages
func (hm *HotkeyManager) wndProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case wmHotkey:
		if hm.callback != nil {
			hm.callback(int(wparam))
		}
		return 0
	}
	
	// 调用默认窗口过程
	ret, _, _ := defWindowProc.Call(
		uintptr(hwnd),
		uintptr(msg),
		wparam,
		lparam,
	)
	return ret
}