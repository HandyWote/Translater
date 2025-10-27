//go:build windows

package overlay

import (
	"fmt"
	"runtime"
	"strings"
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
	textFitPadding  = 4
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
	procShowScrollBar              = user32.NewProc("ShowScrollBar")
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

// Update updates text of the current overlay window.
func (m *Manager) Update(text string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.current == nil {
		if strings.TrimSpace(text) == "" {
			return nil
		}
		return fmt.Errorf("overlay window is not active")
	}
	return m.current.UpdateText(text)
}

type overlayWindow struct {
	baseRect  Rect
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
	scrollPos      int32
	viewHeight     int32
	contentHeight  int32
}

func newOverlayWindow(text string, rect Rect) (*overlayWindow, error) {
	ow := &overlayWindow{
		baseRect:  rect,
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

func (ow *overlayWindow) UpdateText(text string) error {
	if ow == nil {
		return fmt.Errorf("overlay window is nil")
	}
	if ow.hwnd == 0 {
		return fmt.Errorf("overlay window not initialized")
	}

	ow.textUTF16 = append(utf16.Encode([]rune(text)), 0)
	ow.fontRectWidth = 0
	ow.fontRectHeight = 0
	ow.contentHeight = 0
	ow.updateLayout()
	win.InvalidateRect(ow.hwnd, nil, true)
	win.UpdateWindow(ow.hwnd)
	return nil
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
		win.WS_POPUP|win.WS_VSCROLL,
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
	ow.updateLayout()
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
	case win.WM_VSCROLL:
		ow.onVScroll(wParam, lParam)
		return 0
	case win.WM_MOUSEWHEEL:
		ow.onMouseWheel(wParam)
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
		drawRect := inner
		drawRect.Top -= ow.scrollPos
		drawRect.Bottom -= ow.scrollPos
		var previous win.HGDIOBJ
		if font != 0 {
			textHeight := ow.measureTextHeight(hdc, font, int(inner.Right-inner.Left))
			availableHeight := int(drawRect.Bottom - drawRect.Top)

			if textHeight > 0 && availableHeight > textHeight {
				offset := int32((availableHeight - textHeight) / 2)
				if offset > 0 {
					drawRect.Top += offset
					bottom := drawRect.Top + int32(textHeight)
					if bottom < drawRect.Bottom {
						drawRect.Bottom = bottom
					}
				}
			}

			// 确保有足够的绘制空间（防止文字被截断）
			effectiveHeight := drawRect.Bottom - drawRect.Top
			if textHeight > int(effectiveHeight) {
				drawRect.Bottom = drawRect.Top + int32(textHeight)
			}

			previous = win.SelectObject(hdc, win.HGDIOBJ(font))
			defer win.SelectObject(hdc, previous)
		}
		win.DrawTextEx(hdc, &ow.textUTF16[0], -1, &drawRect, win.DT_LEFT|win.DT_WORDBREAK|win.DT_NOPREFIX, nil)
	}
}

func (ow *overlayWindow) updateLayout() {
	if ow.hwnd == 0 {
		return
	}
	hdc := win.GetDC(ow.hwnd)
	if hdc == 0 {
		return
	}
	defer win.ReleaseDC(ow.hwnd, hdc)

	pad := int32(windowPadding)
	maxOuterWidth := int32(rectSafeDimension(ow.baseRect.Width))
	maxOuterHeight := int32(rectSafeDimension(ow.baseRect.Height))
	innerMaxWidth := maxOuterWidth - pad*2
	innerMaxHeight := maxOuterHeight - pad*2
	if innerMaxWidth <= 0 {
		innerMaxWidth = 1
	}
	if innerMaxHeight <= 0 {
		innerMaxHeight = 1
	}

	maxTarget := win.RECT{Left: 0, Top: 0, Right: innerMaxWidth, Bottom: innerMaxHeight}
	font := ow.ensureFittingFont(hdc, &maxTarget)

	innerWidth := innerMaxWidth
	if font != 0 {
		if width := int32(ow.measureTextSingleLine(hdc, font)); width > 0 {
			innerWidth = width
			if innerWidth > innerMaxWidth {
				innerWidth = innerMaxWidth
			}
		}
	}
	if innerWidth <= 0 {
		innerWidth = innerMaxWidth
	}

	_, textHeight := ow.measureText(hdc, font, int(innerWidth))
	contentHeight := int32(textHeight)
	if contentHeight <= 0 {
		contentHeight = innerMaxHeight
	}

	// 检查是否需要滚动条
	needScroll := contentHeight > innerMaxHeight

	// 如果需要滚动条，用减去滚动条宽度后的宽度重新测量
	scrollbarWidth := int32(17) // Windows 标准滚动条宽度
	if needScroll {
		innerWidthWithScrollbar := innerWidth - scrollbarWidth
		if innerWidthWithScrollbar < 1 {
			innerWidthWithScrollbar = 1
		}

		// 用新宽度重新测量
		_, remeasuredHeight := ow.measureText(hdc, font, int(innerWidthWithScrollbar))
		if remeasuredHeight > 0 {
			contentHeight = int32(remeasuredHeight)
		}
	}

	innerHeight := contentHeight
	lineHeight := ow.lineHeight(hdc, font)
	if lineHeight <= 0 {
		lineHeight = 1
	}

	if innerHeight < innerMaxHeight {
		allowance := innerMaxHeight - innerHeight
		extra := lineHeight
		if extra > allowance {
			extra = allowance
		}
		innerHeight += extra
	}
	if innerHeight > innerMaxHeight {
		innerHeight = innerMaxHeight
	}

	if needScroll {
		innerHeight = innerMaxHeight
	}

	outWidth := innerWidth + pad*2
	if outWidth > maxOuterWidth {
		outWidth = maxOuterWidth
	}
	outHeight := innerHeight + pad*2
	if outHeight > maxOuterHeight {
		outHeight = maxOuterHeight
	}

	flags := uint32(win.SWP_NOZORDER | win.SWP_NOACTIVATE)
	win.SetWindowPos(ow.hwnd, 0, int32(ow.baseRect.Left), int32(ow.baseRect.Top), outWidth, outHeight, flags)
	ow.rect.Width = int(outWidth)
	ow.rect.Height = int(outHeight)
	ow.viewHeight = innerHeight
	ow.contentHeight = contentHeight

	if needScroll {
		ow.enableScrollbar()
	} else {
		ow.disableScrollbar()
		ow.scrollPos = 0
	}
}

func rectSafeDimension(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func (ow *overlayWindow) enableScrollbar() {
	var si win.SCROLLINFO
	si.CbSize = uint32(unsafe.Sizeof(si))
	si.FMask = win.SIF_RANGE | win.SIF_PAGE | win.SIF_POS
	si.NMin = 0
	si.NMax = ow.contentHeight
	if si.NMax < 0 {
		si.NMax = 0
	}
	si.NPage = uint32(ow.viewHeight)
	if si.NPage == 0 {
		si.NPage = 1
	}
	if ow.scrollPos > ow.contentHeight-ow.viewHeight {
		ow.scrollPos = maxInt32(0, ow.contentHeight-ow.viewHeight)
	}
	si.NPos = ow.scrollPos
	win.SetScrollInfo(ow.hwnd, win.SB_VERT, &si, true)
	showScrollBar(ow.hwnd, win.SB_VERT, true)
}

func (ow *overlayWindow) disableScrollbar() {
	var si win.SCROLLINFO
	si.CbSize = uint32(unsafe.Sizeof(si))
	si.FMask = win.SIF_RANGE | win.SIF_PAGE | win.SIF_POS
	win.SetScrollInfo(ow.hwnd, win.SB_VERT, &si, true)
	showScrollBar(ow.hwnd, win.SB_VERT, false)
}

func (ow *overlayWindow) measureTextSingleLine(hdc win.HDC, font win.HFONT) int {
	if font == 0 {
		return 0
	}
	prev := win.SelectObject(hdc, win.HGDIOBJ(font))
	defer win.SelectObject(hdc, prev)

	rect := win.RECT{Left: 0, Top: 0, Right: 0, Bottom: 0}
	flags := uint32(win.DT_LEFT | win.DT_NOPREFIX | win.DT_SINGLELINE | win.DT_CALCRECT)
	win.DrawTextEx(hdc, &ow.textUTF16[0], -1, &rect, flags, nil)
	return int(rect.Right - rect.Left)
}

func (ow *overlayWindow) lineHeight(hdc win.HDC, font win.HFONT) int32 {
	if font == 0 {
		return 0
	}
	prev := win.SelectObject(hdc, win.HGDIOBJ(font))
	defer win.SelectObject(hdc, prev)
	var tm win.TEXTMETRIC
	if win.GetTextMetrics(hdc, &tm) {
		return int32(tm.TmHeight)
	}
	return 0
}

func (ow *overlayWindow) onVScroll(wParam, lParam uintptr) {
	_ = lParam
	var si win.SCROLLINFO
	si.CbSize = uint32(unsafe.Sizeof(si))
	si.FMask = win.SIF_ALL
	if !win.GetScrollInfo(ow.hwnd, win.SB_VERT, &si) {
		return
	}
	pos := int32(si.NPos)
	switch lowWord(uint32(wParam)) {
	case win.SB_LINEUP:
		pos -= ow.scrollStep()
	case win.SB_LINEDOWN:
		pos += ow.scrollStep()
	case win.SB_PAGEUP:
		pos -= ow.viewHeight
	case win.SB_PAGEDOWN:
		pos += ow.viewHeight
	case win.SB_THUMBPOSITION, win.SB_THUMBTRACK:
		pos = int32(si.NTrackPos)
	default:
		return
	}
	ow.applyScrollPos(pos)
}

func (ow *overlayWindow) onMouseWheel(wParam uintptr) {
	delta := int16(highWord(uint32(wParam)))
	if delta == 0 {
		return
	}
	step := ow.scrollStep()
	if step == 0 {
		step = 20
	}
	pos := ow.scrollPos - int32(delta)/120*step
	ow.applyScrollPos(pos)
}

func (ow *overlayWindow) scrollStep() int32 {
	if ow.viewHeight > 0 {
		return maxInt32(10, ow.viewHeight/10)
	}
	return 10
}

func (ow *overlayWindow) applyScrollPos(pos int32) {
	if ow.viewHeight <= 0 {
		return
	}
	maxPos := ow.contentHeight - ow.viewHeight
	if maxPos < 0 {
		maxPos = 0
	}
	if pos < 0 {
		pos = 0
	}
	if pos > maxPos {
		pos = maxPos
	}
	ow.scrollPos = pos

	var si win.SCROLLINFO
	si.CbSize = uint32(unsafe.Sizeof(si))
	si.FMask = win.SIF_POS
	si.NPos = pos
	win.SetScrollInfo(ow.hwnd, win.SB_VERT, &si, true)
	win.InvalidateRect(ow.hwnd, nil, true)
}

func lowWord(v uint32) uint16 {
	return uint16(v & 0xFFFF)
}

func highWord(v uint32) uint16 {
	return uint16((v >> 16) & 0xFFFF)
}

func maxInt32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func showScrollBar(hwnd win.HWND, bar int32, show bool) {
	if procShowScrollBar == nil {
		return
	}
	visible := uintptr(0)
	if show {
		visible = 1
	}
	procShowScrollBar.Call(uintptr(hwnd), uintptr(bar), visible)
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

func (ow *overlayWindow) measureText(hdc win.HDC, font win.HFONT, width int) (int, int) {
	if font == 0 || width <= 0 {
		return 0, 0
	}
	prev := win.SelectObject(hdc, win.HGDIOBJ(font))
	defer win.SelectObject(hdc, prev)

	calcRect := win.RECT{Left: 0, Top: 0, Right: int32(width), Bottom: 0}
	flags := uint32(win.DT_LEFT | win.DT_WORDBREAK | win.DT_NOPREFIX | win.DT_CALCRECT)
	win.DrawTextEx(hdc, &ow.textUTF16[0], -1, &calcRect, flags, nil)
	requiredWidth := int(calcRect.Right - calcRect.Left)
	requiredHeight := int(calcRect.Bottom - calcRect.Top)
	return requiredWidth, requiredHeight
}

func (ow *overlayWindow) measureTextHeight(hdc win.HDC, font win.HFONT, width int) int {
	_, height := ow.measureText(hdc, font, width)
	return height
}

func (ow *overlayWindow) textFits(hdc win.HDC, font win.HFONT, width, height int) bool {
	requiredWidth, requiredHeight := ow.measureText(hdc, font, width)
	if requiredWidth == 0 && requiredHeight == 0 {
		return false
	}
	allowedWidth := width
	if allowedWidth > textFitPadding {
		allowedWidth -= textFitPadding
	}
	allowedHeight := height
	if allowedHeight > textFitPadding {
		allowedHeight -= textFitPadding
	}
	return requiredWidth <= allowedWidth && requiredHeight <= allowedHeight
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
