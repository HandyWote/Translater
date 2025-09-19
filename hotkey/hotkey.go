package hotkey

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	MOD_ALT   = 0x0001
	VK_T      = 0x54
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

// HotkeyHandler 热键处理函数类型
type HotkeyHandler func()

// Manager 热键管理器
type Manager struct {
	handlers map[uintptr]HotkeyHandler
}

// NewManager 创建新的热键管理器
func NewManager() *Manager {
	return &Manager{
		handlers: make(map[uintptr]HotkeyHandler),
	}
}

// Register 注册热键
func (m *Manager) Register(id uintptr, mod uintptr, vk uintptr, handler HotkeyHandler) error {
	ret, _, err := registerHotKey.Call(
		0,   // HWND (0 = global)
		id,  // id
		mod, // fsModifiers
		vk,  // vk
	)
	if ret == 0 {
		return fmt.Errorf("注册热键失败: %v", err)
	}
	m.handlers[id] = handler
	return nil
}

// Unregister 注销热键
func (m *Manager) Unregister(id uintptr) {
	unregisterHotKey.Call(0, id)
	delete(m.handlers, id)
}

// Start 启动热键监听
func (m *Manager) Start() {
	fmt.Println("热键监听程序已启动...")
	fmt.Println("按Ctrl+C退出程序")

	// 消息循环
	var msg MSG
	for {
		// 获取消息
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break // WM_QUIT
		}

		// 处理热键消息
		if msg.Message == WM_HOTKEY {
			if handler, exists := m.handlers[msg.WParam]; exists {
				handler()
			}
		}

		// 翻译和分发消息
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}