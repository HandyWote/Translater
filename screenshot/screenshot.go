package screenshot

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"time"

	"github.com/kbinani/screenshot"
	hook "github.com/robotn/gohook"
)

// CaptureHandler 截图处理函数类型
type CaptureHandler func(startX, startY, endX, endY int) bool

// Manager 截图管理器
type Manager struct {
	onCapture CaptureHandler
}

// NewManager 创建新的截图管理器
func NewManager() *Manager {
	return &Manager{}
}

// SetCaptureHandler 设置截图处理函数
func (m *Manager) SetCaptureHandler(handler CaptureHandler) {
	m.onCapture = handler
}

// StartOnce 执行一次截图操作
func (m *Manager) StartOnce() {
	// 监听鼠标事件
	fmt.Println("开始监听鼠标事件...")
	fmt.Println("请按下鼠标左键并拖拽，然后释放来选择截图区域")

	// 用于存储鼠标按下时的坐标
	var startX, startY, endX, endY int

	// 启动事件监听
	evChan := hook.Start()
	defer hook.End()

	// 用于跟踪鼠标是否被按下
	mousePressed := false

	// 使用一个标志来控制循环退出
	shouldExit := false

	for ev := range evChan {
		if shouldExit {
			break
		}
		
		switch ev.Kind {
		case 7: // 鼠标按下
			if ev.Button == hook.MouseMap["left"] {
				startX, startY = int(ev.X), int(ev.Y)
				mousePressed = true
				fmt.Printf("鼠标按下: (%d, %d)\n", startX, startY)
			}
		case 8: // 鼠标释放
			if ev.Button == hook.MouseMap["left"] && mousePressed {
				endX, endY = int(ev.X), int(ev.Y)
				mousePressed = false
				fmt.Printf("鼠标释放: (%d, %d)\n", endX, endY)
				
				// 调用截图处理函数
				if m.onCapture != nil {
					m.onCapture(startX, startY, endX, endY)
				}
				
				// 完成截图后退出本次监听
				fmt.Println("截图完成，等待下次热键触发...")
				shouldExit = true
			}
		case 4: //按下Esc键取消截图
			if ev.Keycode == hook.Keycode["esc"] {
				fmt.Println("截图已取消")
				shouldExit = true
			}
		case 10: // 其他事件
			// 忽略其他事件
		}
	}
	
	// 确保在退出前关闭事件监听
	fmt.Println("截图监听已结束，等待下次热键触发...")
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