//go:build windows

package screenshot

import (
	"fmt"
	"image"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

const (
	selectionWindowClassName = "TranslaterSelectionOverlayWindow"
	overlayBgColor           = 0x000000 // 深色蒙层背景
	overlayColorKey          = 0x010101 // 挖空区域使用的 colorkey
	lwaAlpha                 = 0x02
	lwaColorKey              = 0x01
	borderThickness          = 3
	overlayAlpha             = 120 // 0-255，越小越透明
)

var (
	selectionClassOnce      sync.Once
	selectionClassErr       error
	selectionClassNameUTF16 = syscall.StringToUTF16Ptr(selectionWindowClassName)
	user32Dll               = syscall.NewLazyDLL("user32.dll")
	gdi32Dll                = syscall.NewLazyDLL("gdi32.dll")
	procSetLayeredAttrs     = user32Dll.NewProc("SetLayeredWindowAttributes")
	procCreateSolidBrush    = gdi32Dll.NewProc("CreateSolidBrush")
	procFillRect            = user32Dll.NewProc("FillRect")
)

type selectionOverlayWindow struct {
	hwnd       win.HWND
	start      image.Point
	end        image.Point
	active     bool
	origin     image.Point
	prevStart  image.Point
	prevEnd    image.Point
	prevActive bool

	mu     sync.Mutex
	ready  chan error
	closed chan struct{}
}

func newSelectionPreview() selectionPreview {
	return &selectionOverlayWindow{}
}

func (ow *selectionOverlayWindow) Start() error {
	ow.mu.Lock()
	ow.ready = make(chan error, 1)
	ow.closed = make(chan struct{})
	ow.active = true // 启动即显示蒙层
	ow.mu.Unlock()

	go ow.loop()

	if err := <-ow.ready; err != nil {
		<-ow.closed
		fmt.Printf("选区预览启动失败: %v\n", err)
		return err
	}
	fmt.Println("选区预览窗口已启动")
	return nil
}

func (ow *selectionOverlayWindow) Update(startX, startY, currentX, currentY int, active bool) {
	ow.mu.Lock()
	ow.start = image.Pt(startX, startY)
	ow.end = image.Pt(currentX, currentY)
	ow.active = active
	if ow.start == ow.prevStart && ow.end == ow.prevEnd && ow.active == ow.prevActive {
		ow.mu.Unlock()
		return
	}
	ow.prevStart = ow.start
	ow.prevEnd = ow.end
	ow.prevActive = ow.active
	hwnd := ow.hwnd
	ow.mu.Unlock()

	if hwnd != 0 {
		win.InvalidateRect(hwnd, nil, true)
	}
}

func (ow *selectionOverlayWindow) Close() {
	ow.mu.Lock()
	hwnd := ow.hwnd
	closed := ow.closed
	ow.mu.Unlock()

	if hwnd != 0 {
		win.PostMessage(hwnd, win.WM_CLOSE, 0, 0)
	}
	if closed != nil {
		<-closed
	}
	fmt.Println("选区预览窗口已关闭")
}

func (ow *selectionOverlayWindow) loop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ensureSelectionWindowClass(); err != nil {
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

	ow.mu.Lock()
	ow.hwnd = 0
	closed := ow.closed
	ow.mu.Unlock()

	close(closed)
}

func (ow *selectionOverlayWindow) createWindow() error {
	left := win.GetSystemMetrics(win.SM_XVIRTUALSCREEN)
	top := win.GetSystemMetrics(win.SM_YVIRTUALSCREEN)
	width := win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
	height := win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	if width <= 0 || height <= 0 {
		width = win.GetSystemMetrics(win.SM_CXSCREEN)
		height = win.GetSystemMetrics(win.SM_CYSCREEN)
		left = 0
		top = 0
	}

	hwnd := win.CreateWindowEx(
		// 去掉 WS_EX_TRANSPARENT，阻止鼠标事件穿透到底层窗口
		win.WS_EX_LAYERED|win.WS_EX_TOPMOST|win.WS_EX_TOOLWINDOW|win.WS_EX_NOACTIVATE,
		selectionClassNameUTF16,
		nil,
		win.WS_POPUP,
		int32(left),
		int32(top),
		int32(width),
		int32(height),
		0,
		0,
		win.GetModuleHandle(nil),
		unsafe.Pointer(ow),
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx for selection overlay failed: %w", syscall.Errno(win.GetLastError()))
	}

	// 使用全局 alpha + colorkey（挖空时填充 colorkey 颜色），不会影响鼠标命中
	if err := setLayeredWindowAttributes(hwnd, overlayColorKey, overlayAlpha, lwaAlpha|lwaColorKey); err != nil {
		return err
	}

	ow.mu.Lock()
	ow.hwnd = hwnd
	ow.origin = image.Pt(int(left), int(top))
	ow.mu.Unlock()

	win.ShowWindow(hwnd, win.SW_SHOWNOACTIVATE)
	win.UpdateWindow(hwnd)
	fmt.Printf("选区预览窗口创建成功: origin=(%d,%d) size=%dx%d\n", left, top, width, height)
	return nil
}

func ensureSelectionWindowClass() error {
	selectionClassOnce.Do(func() {
		hInstance := win.GetModuleHandle(nil)
		var wc win.WNDCLASSEX
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		wc.Style = win.CS_HREDRAW | win.CS_VREDRAW
		wc.LpfnWndProc = syscall.NewCallback(selectionWndProc)
		wc.HInstance = hInstance
		wc.HCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_ARROW))
		wc.HbrBackground = 0
		wc.LpszClassName = selectionClassNameUTF16
		if atom := win.RegisterClassEx(&wc); atom == 0 {
			selectionClassErr = fmt.Errorf("RegisterClassEx for selection overlay failed: %w", syscall.Errno(win.GetLastError()))
			return
		}
	})
	return selectionClassErr
}

//go:nocheckptr
func selectionWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_NCCREATE:
		cs := (*win.CREATESTRUCT)(unsafe.Pointer(lParam)) //nolint:unsafeptr // Win32 回调传入指针
		win.SetWindowLongPtr(hwnd, win.GWLP_USERDATA, cs.CreateParams)
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	ow := (*selectionOverlayWindow)(unsafe.Pointer(win.GetWindowLongPtr(hwnd, win.GWLP_USERDATA))) //nolint:unsafeptr // 从窗口数据恢复指针
	if ow == nil {
		return win.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case win.WM_PAINT:
		ow.onPaint(hwnd)
		return 0
	case win.WM_LBUTTONDOWN:
		win.SetCapture(hwnd)
		x := int(int16(lowWord(uint32(lParam))))
		y := int(int16(highWord(uint32(lParam))))
		fmt.Printf("选区预览捕获鼠标: 按下(%d,%d)\n", x, y)
		return 0
	case win.WM_MOUSEMOVE:
		// 阻断事件向下传递，避免影响底层窗口
		return 0
	case win.WM_LBUTTONUP:
		win.ReleaseCapture()
		x := int(int16(lowWord(uint32(lParam))))
		y := int(int16(highWord(uint32(lParam))))
		fmt.Printf("选区预览释放鼠标: 抬起(%d,%d)\n", x, y)
		return 0
	case win.WM_SETCURSOR:
		win.SetCursor(win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_CROSS)))
		return 1
	case win.WM_NCHITTEST:
		return win.HTCLIENT
	case win.WM_DESTROY:
		win.PostQuitMessage(0)
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (ow *selectionOverlayWindow) onPaint(hwnd win.HWND) {
	var ps win.PAINTSTRUCT
	hdc := win.BeginPaint(hwnd, &ps)
	if hdc == 0 {
		return
	}
	defer win.EndPaint(hwnd, &ps)

	var rect win.RECT
	win.GetClientRect(hwnd, &rect)

	// 背景画刷（半透明遮罩）
	bgBrush, err := createSolidBrush(overlayBgColor)
	if err != nil || bgBrush == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(bgBrush))

	ow.mu.Lock()
	active := ow.active
	start := ow.start
	end := ow.end
	origin := ow.origin
	ow.mu.Unlock()

	if !active {
		fillRect(hdc, &rect, bgBrush)
		return
	}

	left := minInt(start.X, end.X) - origin.X
	right := maxInt(start.X, end.X) - origin.X
	top := minInt(start.Y, end.Y) - origin.Y
	bottom := maxInt(start.Y, end.Y) - origin.Y

	width := right - left
	height := bottom - top
	if width < 5 || height < 5 {
		// 没有有效选区时，全屏蒙层
		fillRect(hdc, &rect, bgBrush)
		return
	}

	// 先铺满背景，再用 colorkey 挖空选区区域
	fillRect(hdc, &rect, bgBrush)

	holeBrush, err := createSolidBrush(overlayColorKey)
	if err != nil || holeBrush == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(holeBrush))

	sel := win.RECT{
		Left:   int32(left),
		Top:    int32(top),
		Right:  int32(right),
		Bottom: int32(bottom),
	}
	fillRect(hdc, &sel, holeBrush)

	frameBrush, err := createSolidBrush(win.RGB(0, 170, 255))
	if err != nil || frameBrush == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(frameBrush))

	// 手工绘制矩形边框（FrameRect 在 lxn/win 中未导出）
	drawRectEdges(hdc, &sel, frameBrush, borderThickness)
}

func setLayeredWindowAttributes(hwnd win.HWND, colorKey uint32, alpha byte, flags uint32) error {
	r1, _, err := procSetLayeredAttrs.Call(
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

func fillRect(hdc win.HDC, rect *win.RECT, brush win.HBRUSH) {
	procFillRect.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(rect)),
		uintptr(brush),
	)
}

func drawRectEdges(hdc win.HDC, rect *win.RECT, brush win.HBRUSH, thickness int32) {
	if thickness < 1 {
		thickness = 1
	}
	width := rect.Right - rect.Left
	height := rect.Bottom - rect.Top
	if width <= 0 || height <= 0 {
		return
	}
	if thickness*2 > width {
		thickness = width / 2
	}
	if thickness*2 > height {
		thickness = height / 2
	}
	if thickness < 1 {
		thickness = 1
	}

	// 顶部
	top := win.RECT{Left: rect.Left, Top: rect.Top, Right: rect.Right, Bottom: rect.Top + thickness}
	fillRect(hdc, &top, brush)
	// 底部
	bottom := win.RECT{Left: rect.Left, Top: rect.Bottom - thickness, Right: rect.Right, Bottom: rect.Bottom}
	fillRect(hdc, &bottom, brush)
	// 左侧
	left := win.RECT{Left: rect.Left, Top: rect.Top + thickness, Right: rect.Left + thickness, Bottom: rect.Bottom - thickness}
	fillRect(hdc, &left, brush)
	// 右侧
	right := win.RECT{Left: rect.Right - thickness, Top: rect.Top + thickness, Right: rect.Right, Bottom: rect.Bottom - thickness}
	fillRect(hdc, &right, brush)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func lowWord(v uint32) uint16 {
	return uint16(v & 0xFFFF)
}

func highWord(v uint32) uint16 {
	return uint16((v >> 16) & 0xFFFF)
}
