//go:build windows
// +build windows

package hotkey

import (
	"syscall"
)

// Hotkey represents a global hotkey
type Hotkey struct {
	ID        int
	Modifiers uint32
	Key       uint32
}

// HotkeyManager manages global hotkeys
type HotkeyManager struct {
	hotkeys  map[int]*Hotkey
	hwnd     syscall.Handle
	callback func(int)
	quit     chan bool
}