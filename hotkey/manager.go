//go:build windows
// +build windows

package hotkey

import (
	"syscall"
)

// NewHotkeyManager creates a new hotkey manager
func NewHotkeyManager() *HotkeyManager {
	return &HotkeyManager{
		hotkeys: make(map[int]*Hotkey),
		quit:    make(chan bool),
	}
}

// Register registers a global hotkey
func (hm *HotkeyManager) Register(id int, modifiers uint32, key uint32) error {
	hotkey := &Hotkey{
		ID:        id,
		Modifiers: modifiers,
		Key:       key,
	}
	
	hm.hotkeys[id] = hotkey
	
	// 如果hwnd已经创建，则立即注册热键
	if hm.hwnd != 0 {
		return registerHotKeyWindows(hm.hwnd, id, modifiers, key)
	}
	
	return nil
}

// Unregister unregisters a global hotkey
func (hm *HotkeyManager) Unregister(id int) error {
	delete(hm.hotkeys, id)
	
	// 如果hwnd已经创建，则立即注销热键
	if hm.hwnd != 0 {
		return unregisterHotKeyWindows(hm.hwnd, id)
	}
	
	return nil
}

// Start starts the hotkey message loop
func (hm *HotkeyManager) Start(callback func(int)) error {
	hm.callback = callback
	
	// 创建隐藏窗口
	if err := hm.createHiddenWindow(); err != nil {
		return err
	}
	
	// 注册所有热键
	for _, hotkey := range hm.hotkeys {
		if err := registerHotKeyWindows(hm.hwnd, hotkey.ID, hotkey.Modifiers, hotkey.Key); err != nil {
			return err
		}
	}
	
	// 启动消息循环
	go hm.messageLoop()
	
	return nil
}

// Stop stops the hotkey message loop
func (hm *HotkeyManager) Stop() {
	hm.quit <- true
}