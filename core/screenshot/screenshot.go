package screenshot

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"
	"time"

	"github.com/kbinani/screenshot"
	hook "github.com/robotn/gohook"
)

// CaptureHandler 截图处理函数类型
type CaptureHandler func(ctx context.Context, startX, startY, endX, endY int) bool

// Manager 截图管理器
type Manager struct {
	mu        sync.Mutex
	onCapture CaptureHandler
	cancel    context.CancelFunc
	done      chan struct{}
}

// NewManager 创建新的截图管理器
func NewManager() *Manager {
	return &Manager{}
}

// SetCaptureHandler 设置截图处理函数
func (m *Manager) SetCaptureHandler(handler CaptureHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onCapture = handler
}

// CancelActiveCapture 主动终止正在进行的截图任务（如果存在）
func (m *Manager) CancelActiveCapture() {
	m.mu.Lock()
	cancel := m.cancel
	done := m.done
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// StartOnce 执行一次截图操作
func (m *Manager) StartOnce() {
	// 确保不会存在遗留的截图任务
	m.CancelActiveCapture()

	fmt.Println("开始监听鼠标事件...")
	fmt.Println("请按下鼠标左键并拖拽，然后释放来选择截图区域")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	preview := newSelectionPreview()
	if err := preview.Start(); err != nil {
		fmt.Printf("选区预览启动失败: %v\n", err)
		preview = nil
	}

	m.mu.Lock()
	m.cancel = cancel
	m.done = done
	m.mu.Unlock()

	defer func() {
		cancel()
		m.mu.Lock()
		m.cancel = nil
		if m.done == done {
			m.done = nil
		}
		m.mu.Unlock()
		close(done)
		if preview != nil {
			preview.Close()
		}
	}()

	evChan := hook.Start()
	defer hook.End()

	var startX, startY, endX, endY int
	mousePressed := false
	var translationStarted bool
	var translationDone chan bool

	for {
		select {
		case <-ctx.Done():
			fmt.Println("截图任务已被取消")
			fmt.Println("截图监听已结束，等待下次热键触发...")
			return
		case handled, ok := <-translationDone:
			if ok {
				if handled {
					fmt.Println("截图完成，等待下次热键触发...")
				}
				fmt.Println("截图监听已结束，等待下次热键触发...")
				return
			}
		case ev, ok := <-evChan:
			if !ok {
				fmt.Println("事件通道已关闭，退出截图监听")
				fmt.Println("截图监听已结束，等待下次热键触发...")
				return
			}

			// 翻译阶段仅关注取消事件，避免额外干扰
			if translationStarted {
				if ev.Kind == 4 && ev.Keycode == hook.Keycode["esc"] {
					fmt.Println("翻译已被取消")
					cancel()
				}
				continue
			}

			switch ev.Kind {
			case hook.MouseDown: // 鼠标按下
				if ev.Button == hook.MouseMap["left"] {
					startX, startY = int(ev.X), int(ev.Y)
					mousePressed = true
					if preview != nil {
						preview.Update(startX, startY, startX, startY, true)
					}
					fmt.Printf("鼠标按下: (%d, %d)\n", startX, startY)
				}
			case hook.MouseHold, hook.MouseUp: // 鼠标释放（兼容旧逻辑）
				if ev.Button == hook.MouseMap["left"] && mousePressed {
					endX, endY = int(ev.X), int(ev.Y)
					mousePressed = false
					if preview != nil {
						preview.Update(startX, startY, endX, endY, false)
						preview.Close()
						preview = nil
					}
					fmt.Printf("鼠标释放: (%d, %d)\n", endX, endY)

					handler := m.getCaptureHandler()
					if handler != nil {
						translationDone = make(chan bool, 1)
						translationStarted = true
						go func() {
							defer close(translationDone)
							translationDone <- handler(ctx, startX, startY, endX, endY)
						}()
					} else {
						fmt.Println("未设置截图处理函数，直接退出")
						fmt.Println("截图监听已结束，等待下次热键触发...")
						return
					}
				}
			case hook.MouseMove, hook.MouseDrag:
				if mousePressed && preview != nil {
					preview.Update(startX, startY, int(ev.X), int(ev.Y), true)
				}
			case 4: // 按下Esc键取消截图
				if ev.Keycode == hook.Keycode["esc"] {
					fmt.Println("截图已取消")
					fmt.Println("截图监听已结束，等待下次热键触发...")
					return
				}
			default:
				// 忽略其他事件
			}
		}
	}
}

func (m *Manager) getCaptureHandler() CaptureHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.onCapture
}

// Capture 截取指定区域的屏幕并保存
func Capture(startX, startY, endX, endY int) bool {
	// 确保坐标是正确的（左上到右下）
	if startX > endX {
		startX, endX = endX, startX
	}
	if startY > endY {
		startY, endY = endY, startY
	}

	// 创建一个 Rectangle 对象
	rect := image.Rect(startX, startY, endX, endY)

	// 截图
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		fmt.Printf("截图失败: %v\n", err)
		return false
	}

	// 保存图片
	filename := fmt.Sprintf("screenshot_%d.png", time.Now().Unix())
	if saveImage(img, filename) {
		return true
	} else {
		return false
	}
}

// saveImage 保存图片
func saveImage(img image.Image, filename string) bool {
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return false
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		fmt.Printf("保存图片失败: %v\n", err)
		return false
	}

	fmt.Printf("图片已保存: %s\n", filename)
	return true
}

// CaptureToBytes 截取指定区域的屏幕并返回图像字节数据
func CaptureToBytes(startX, startY, endX, endY int) ([]byte, error) {
	// 确保坐标是正确的（左上到右下）
	if startX > endX {
		startX, endX = endX, startX
	}
	if startY > endY {
		startY, endY = endY, startY
	}

	// 创建一个 Rectangle 对象
	rect := image.Rect(startX, startY, endX, endY)

	// 截图
	img, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %v", err)
	}

	// 将图像编码为字节数据
	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return nil, fmt.Errorf("编码图像失败: %v", err)
	}

	return buf.Bytes(), nil
}
