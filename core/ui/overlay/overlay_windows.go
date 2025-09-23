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
	vkEscape        = 0x1B
	escHotkeyID     = 0x4F52
)

var (
	classOnce                      sync.Once
	classErr                       error
	classNameUTF16                 = syscall.StringToUTF16Ptr(windowClassName)
	user32                         = syscall.NewLazyDLL("user32.dll")
	gdi32                          = syscall.NewLazyDLL("gdi32.dll")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	procCreateSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	procFillRect                   = user32.NewProc("FillRect")
	procRegisterHotKey             = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey           = user32.NewProc("UnregisterHotKey")
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

	font           win.HFONT
	fontRectWidth  int32
	fontRectHeight int32
	fontPointSize  int
	escHotkey      bool
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
	ow.registerEscapeHotkey()
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
		ow.cleanup()
		win.PostQuitMessage(0)
		return 0
	case win.WM_HOTKEY:
		if uintptr(wParam) == escHotkeyID {
			win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
			return 0
		}
	case win.WM_KEYDOWN, win.WM_SYSKEYDOWN:
		if wParam == vkEscape {
			win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
			return 0
		}
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
		font := ow.ensureFittingFont(hdc, &inner)
		var previous win.HGDIOBJ
		if font != 0 {
			previous = win.SelectObject(hdc, win.HGDIOBJ(font))
			defer win.SelectObject(hdc, previous)
		}
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
	procFillRect.Call(
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

func (ow *overlayWindow) registerEscapeHotkey() {
	if procRegisterHotKey == nil {
		return
	}
	r1, _, err := procRegisterHotKey.Call(
		uintptr(ow.hwnd),
		escHotkeyID,
		0,
		vkEscape,
	)
	if r1 == 0 {
		if err != syscall.Errno(0) {
			fmt.Printf("RegisterHotKey ESC failed: %v\n", err)
		}
		return
	}
	ow.escHotkey = true
}

func (ow *overlayWindow) ensureFittingFont(hdc win.HDC, target *win.RECT) win.HFONT {
	availableWidth := target.Right - target.Left
	availableHeight := target.Bottom - target.Top
	if availableWidth <= 0 || availableHeight <= 0 {
		return 0
	}

	if len(ow.textUTF16) == 0 {
		return 0
	}

	if ow.font != 0 && ow.fontRectWidth == availableWidth && ow.fontRectHeight == availableHeight {
		return ow.font
	}

	if ow.font != 0 {
		win.DeleteObject(win.HGDIOBJ(ow.font))
		ow.font = 0
	}

	logPixelsY := win.GetDeviceCaps(hdc, win.LOGPIXELSY)
	if logPixelsY <= 0 {
		logPixelsY = 96
	}

	minPoint := 8
	maxPoint := maxInt(minPoint, int(availableHeight)*72/int(logPixelsY))
	if maxPoint > 400 {
		maxPoint = 400
	}

	var (
		bestFont win.HFONT
		bestSize int
	)

	for minPoint <= maxPoint {
		mid := (minPoint + maxPoint) / 2
		font := createFontForPoint(logPixelsY, mid)
		if font == 0 {
			break
		}
		if ow.textFits(hdc, font, int(availableWidth), int(availableHeight)) {
			if bestFont != 0 {
				win.DeleteObject(win.HGDIOBJ(bestFont))
			}
			bestFont = font
			bestSize = mid
			minPoint = mid + 1
		} else {
			win.DeleteObject(win.HGDIOBJ(font))
			maxPoint = mid - 1
		}
	}

	if bestFont == 0 {
		bestFont = createFontForPoint(logPixelsY, minPoint)
		bestSize = minPoint
	}

	ow.font = bestFont
	ow.fontRectWidth = availableWidth
	ow.fontRectHeight = availableHeight
	ow.fontPointSize = bestSize
	return ow.font
}

func (ow *overlayWindow) textFits(hdc win.HDC, font win.HFONT, width, height int) bool {
	prev := win.SelectObject(hdc, win.HGDIOBJ(font))
	defer win.SelectObject(hdc, prev)

	calcRect := win.RECT{Left: 0, Top: 0, Right: int32(width), Bottom: 0}
	flags := uint32(win.DT_LEFT | win.DT_WORDBREAK | win.DT_NOPREFIX | win.DT_CALCRECT)
	win.DrawTextEx(hdc, &ow.textUTF16[0], -1, &calcRect, flags, nil)
	requiredWidth := calcRect.Right - calcRect.Left
	requiredHeight := calcRect.Bottom - calcRect.Top
	return int(requiredWidth) <= width && int(requiredHeight) <= height
}

func createFontForPoint(logPixelsY int32, pointSize int) win.HFONT {
	if pointSize < 1 {
		pointSize = 1
	}

	height := -int32(pointSize * int(logPixelsY) / 72)
	if height == 0 {
		height = -1
	}

	var lf win.LOGFONT
	lf.LfHeight = height
	lf.LfWeight = win.FW_NORMAL
	lf.LfCharSet = win.DEFAULT_CHARSET
	lf.LfOutPrecision = win.OUT_DEFAULT_PRECIS
	lf.LfClipPrecision = win.CLIP_DEFAULT_PRECIS
	lf.LfQuality = win.CLEARTYPE_QUALITY
	lf.LfPitchAndFamily = win.DEFAULT_PITCH | win.FF_DONTCARE
	face := syscall.StringToUTF16("Microsoft YaHei")
	copy(lf.LfFaceName[:], face)

	return win.CreateFontIndirect(&lf)
}

func (ow *overlayWindow) cleanup() {
	if ow.font != 0 {
		win.DeleteObject(win.HGDIOBJ(ow.font))
		ow.font = 0
	}
	ow.fontRectWidth = 0
	ow.fontRectHeight = 0
	ow.fontPointSize = 0
	if ow.escHotkey {
		procUnregisterHotKey.Call(uintptr(ow.hwnd), escHotkeyID)
		ow.escHotkey = false
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
