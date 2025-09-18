//go:build windows
// +build windows

package hotkey

import (
	"syscall"
	"unsafe"
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	registerHotKey   = user32.NewProc("RegisterHotKey")
	unregisterHotKey = user32.NewProc("UnregisterHotKey")
)

// registerHotKeyWindows registers a hotkey with Windows API
func registerHotKeyWindows(hwnd syscall.Handle, id int, modifiers uint32, key uint32) error {
	ret, _, err := registerHotKey.Call(
		uintptr(hwnd),
		uintptr(id),
		uintptr(modifiers),
		uintptr(key),
	)
	if ret == 0 {
		return err
	}
	return nil
}

// unregisterHotKeyWindows unregisters a hotkey with Windows API
func unregisterHotKeyWindows(hwnd syscall.Handle, id int) error {
	ret, _, err := unregisterHotKey.Call(
		uintptr(hwnd),
		uintptr(id),
	)
	if ret == 0 {
		return err
	}
	return nil
}