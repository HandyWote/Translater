package hotkey

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	MOD_WIN     = 0x0008

	VK_T = 0x54

	WM_HOTKEY = 0x0312
	WM_APP    = 0x8000

	commandMessage = WM_APP + 1
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	registerHotKey     = user32.NewProc("RegisterHotKey")
	unregisterHotKey   = user32.NewProc("UnregisterHotKey")
	getMessage         = user32.NewProc("GetMessageW")
	translateMessage   = user32.NewProc("TranslateMessage")
	dispatchMessage    = user32.NewProc("DispatchMessageW")
	postThreadMessage  = user32.NewProc("PostThreadMessageW")
	getCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")
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

type registerCommand struct {
	id      uintptr
	mod     uintptr
	vk      uintptr
	handler HotkeyHandler
	resp    chan error
}

type unregisterCommand struct {
	id   uintptr
	resp chan error
}

// Manager 热键管理器
type Manager struct {
	mu       sync.RWMutex
	handlers map[uintptr]HotkeyHandler

	cmdCh     chan interface{}
	ready     chan struct{}
	done      chan struct{}
	startOnce sync.Once
	threadID  uint32
}

// NewManager 创建新的热键管理器
func NewManager() *Manager {
	return &Manager{
		handlers: make(map[uintptr]HotkeyHandler),
	}
}

func (m *Manager) ensureLoop() {
	m.startOnce.Do(func() {
		m.cmdCh = make(chan interface{}, 16)
		m.ready = make(chan struct{})
		m.done = make(chan struct{})
		go m.loop()
		<-m.ready
	})
}

// Start 启动热键监听
func (m *Manager) Start() {
	m.ensureLoop()
	fmt.Println("热键监听程序已启动...")
	fmt.Println("按Ctrl+C退出程序")
	<-m.done
}

// Register 注册热键
func (m *Manager) Register(id uintptr, mod uintptr, vk uintptr, handler HotkeyHandler) error {
	if handler == nil {
		return fmt.Errorf("注册热键失败: handler 不能为空")
	}
	m.ensureLoop()

	resp := make(chan error, 1)
	m.cmdCh <- registerCommand{id: id, mod: mod, vk: vk, handler: handler, resp: resp}
	m.wakeLoop()
	return <-resp
}

// Unregister 注销热键
func (m *Manager) Unregister(id uintptr) {
	m.ensureLoop()
	resp := make(chan error, 1)
	m.cmdCh <- unregisterCommand{id: id, resp: resp}
	m.wakeLoop()
	<-resp
}

func (m *Manager) wakeLoop() {
	if m.threadID == 0 {
		return
	}
	postThreadMessage.Call(uintptr(m.threadID), commandMessage, 0, 0)
}

func (m *Manager) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	tid, _, _ := getCurrentThreadID.Call()
	m.threadID = uint32(tid)
	close(m.ready)

	var msg MSG
	for {
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret == 0 {
			break // WM_QUIT
		}

		if msg.Message == commandMessage {
			m.processPendingCommands()
			continue
		}

		if msg.Message == WM_HOTKEY {
			m.mu.RLock()
			handler := m.handlers[msg.WParam]
			m.mu.RUnlock()
			if handler != nil {
				handler()
			}
			continue
		}

		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	close(m.done)
}

func (m *Manager) processPendingCommands() {
	for {
		select {
		case cmd := <-m.cmdCh:
			switch c := cmd.(type) {
			case registerCommand:
				c.resp <- m.registerOnThread(c)
			case unregisterCommand:
				unregisterHotKey.Call(0, c.id)
				m.mu.Lock()
				delete(m.handlers, c.id)
				m.mu.Unlock()
				c.resp <- nil
			}
		default:
			return
		}
	}
}

func (m *Manager) registerOnThread(cmd registerCommand) error {
	if cmd.handler == nil {
		return fmt.Errorf("handler 不能为空")
	}

	tryRegister := func() (uintptr, error) {
		ret, _, err := registerHotKey.Call(0, cmd.id, cmd.mod, cmd.vk)
		if ret == 0 {
			return ret, err
		}
		return ret, nil
	}

	if ret, err := tryRegister(); ret == 0 {
		if errno, ok := err.(syscall.Errno); ok && errno == 1409 {
			unregisterHotKey.Call(0, cmd.id)
			time.Sleep(20 * time.Millisecond)
			if ret, err = tryRegister(); ret == 0 {
				return fmt.Errorf("注册热键失败: %v", err)
			}
		} else {
			return fmt.Errorf("注册热键失败: %v", err)
		}
	}

	m.mu.Lock()
	m.handlers[cmd.id] = cmd.handler
	m.mu.Unlock()
	return nil
}
