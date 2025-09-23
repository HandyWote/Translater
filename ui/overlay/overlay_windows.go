//go:build windows

package overlay

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lxn/win"
)

const (
	windowClassName = "TranslaterOverlayWindow"
	windowPadding   = 14
	lwaAlpha        = 0x02
)

var (
	classOnce                      sync.Once
	classErr                       error
	classNameUTF16                 = syscall.StringToUTF16Ptr(windowClassName)
	user32                         = syscall.NewLazyDLL("user32.dll")
	gdi32                          = syscall.NewLazyDLL("gdi32.dll")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procCreateSolidBrush           = gdi32.NewProc("CreateSolidBrush")
)

// Manager coordinates overlay window lifecycle on Windows.
type Manager struct {
	mu      sync.Mutex
	current *overlayWindow
}

// NewManager creates a new Windows overlay manager.
func NewManager() *Manager {
	return &Manager{}
}

// Show ensures only one overlay window exists and displays the provided text.
func (m *Manager) Show(text string, rect Rect) error {
	if len(text) == 0 {
		m.Close()
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil {
		m.current.Close()
		m.current = nil
	}

	ow, err := newOverlayWindow(text, rect)
	if err != nil {
		return err
	}
	m.current = ow
	return nil
}

// Close closes the current overlay window if any.
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current != nil {
		m.current.Close()
		m.current = nil
	}
}

type overlayWindow struct {
	rect      Rect
	textUTF16 []uint16

	hwnd   win.HWND
	ready  chan error
	closed chan struct{}
}

func newOverlayWindow(text string, rect Rect) (*overlayWindow, error) {
	ow := &overlayWindow{
		rect:      rect,
		textUTF16: append(utf16.Encode([]rune(text)), 0),
		ready:     make(chan error, 1),
		closed:    make(chan struct{}),
	}

	go ow.loop()

	if err := <-ow.ready; err != nil {
		<-ow.closed
		return nil, err
	}

	return ow, nil
}

func (ow *overlayWindow) Close() {
	if ow == nil {
		return
	}
	if ow.hwnd != 0 {
		win.PostMessage(ow.hwnd, win.WM_CLOSE, 0, 0)
	}
	<-ow.closed
}

func (ow *overlayWindow) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ensureWindowClass(); err != nil {
		ow.ready <- err
		close(ow.ready)
		close(ow.closed)
		return
	}

	if err := ow.createWindow(); err != nil {
		ow.ready <- err
		close(ow.ready)
		close(ow.closed)
		return
	}

	ow.ready <- nil
	close(ow.ready)

	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 || ret == -1 {
			break
		}
		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)
	}

	ow.hwnd = 0
	close(ow.closed)
}

func (ow *overlayWindow) createWindow() error {
	width := int32(rectSafeDimension(ow.rect.Width))
	height := int32(rectSafeDimension(ow.rect.Height))

	hwnd := win.CreateWindowEx(
		win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|win.WS_EX_LAYERED|win.WS_EX_NOACTIVATE,
		classNameUTF16,
		nil,
		win.WS_POPUP,
		int32(ow.rect.Left),
		int32(ow.rect.Top),
		width,
		height,
		0,
		0,
		win.GetModuleHandle(nil),
		unsafe.Pointer(ow),
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx failed: %w", syscall.Errno(win.GetLastError()))
	}

	ow.hwnd = hwnd

	if err := setLayeredWindowAttributes(hwnd, 0, 240, lwaAlpha); err != nil {
		return err
	}
	win.ShowWindow(hwnd, win.SW_SHOWNOACTIVATE)
	win.UpdateWindow(hwnd)
	return nil
}

func ensureWindowClass() error {
	classOnce.Do(func() {
		hInstance := win.GetModuleHandle(nil)
		var wc win.WNDCLASSEX
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		wc.Style = win.CS_HREDRAW | win.CS_VREDRAW
		wc.LpfnWndProc = syscall.NewCallback(overlayWndProc)
		wc.HInstance = hInstance
		wc.HCursor = win.LoadCursor(0, (*uint16)(unsafe.Pointer(uintptr(win.IDC_ARROW))))
		wc.HbrBackground = 0
		wc.LpszClassName = classNameUTF16
		if atom := win.RegisterClassEx(&wc); atom == 0 {
			classErr = fmt.Errorf("RegisterClassEx failed: %w", syscall.Errno(win.GetLastError()))
			return
		}
	})
	return classErr
}

func overlayWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_NCCREATE:
		cs := (*win.CREATESTRUCT)(unsafe.Pointer(lParam))
		win.SetWindowLongPtr(hwnd, win.GWLP_USERDATA, cs.CreateParams)
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	ow := (*overlayWindow)(unsafe.Pointer(win.GetWindowLongPtr(hwnd, win.GWLP_USERDATA)))
	if ow == nil {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case win.WM_PAINT:
		ow.onPaint(hwnd)
		return 0
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	case win.WM_ERASEBKGND:
		return 1
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (ow *overlayWindow) onPaint(hwnd win.HWND) {
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	if hdc == 0 {
		return
	}
	defer win.EndPaint(hwnd, &ps)

	var rect win.RECT
	win.GetClientRect(hwnd, &rect)

	bgBrush, err := createSolidBrush(win.RGB(20, 24, 32))
	if err != nil {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(bgBrush))
	fillRect(hdc, &rect, bgBrush)

	inner := rect
	pad := int32(windowPadding)
	if inner.Right-inner.Left > pad*2 {
		inner.Left += pad
		inner.Right -= pad
	}
	if inner.Bottom-inner.Top > pad*2 {
		inner.Top += pad
		inner.Bottom -= pad
	}

	win.SetBkMode(hdc, win.TRANSPARENT)
	win.SetTextColor(hdc, win.RGB(240, 247, 255))
	if len(ow.textUTF16) > 0 {
		win.DrawTextEx(hdc, &ow.textUTF16[0], -1, &inner, win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX, nil)
	}
}

func rectSafeDimension(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func fillRect(hdc win.HDC, rect *win.RECT, brush win.HBRUSH) {
	user32.NewProc("FillRect").Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(rect)),
		uintptr(brush),
	)
}

func setLayeredWindowAttributes(hwnd win.HWND, colorKey uint32, alpha byte, flags uint32) error {
	r1, _, err := procSetLayeredWindowAttributes.Call(
		uintptr(hwnd),
		uintptr(colorKey),
		uintptr(alpha),
		uintptr(flags),
	)
	if r1 == 0 {
		if err == syscall.Errno(0) {
			return fmt.Errorf("SetLayeredWindowAttributes failed")
		}
		return err
	}
	return nil
}

func createSolidBrush(color win.COLORREF) (win.HBRUSH, error) {
	r1, _, err := procCreateSolidBrush.Call(uintptr(color))
	if r1 == 0 {
		if err == syscall.Errno(0) {
			return 0, fmt.Errorf("CreateSolidBrush failed")
		}
		return 0, err
	}
	return win.HBRUSH(r1), nil
}
