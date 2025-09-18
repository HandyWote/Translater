package main

import (
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/kbinani/screenshot"
)

func main() {
	fmt.Println("屏幕截图程序已启动...")
	fmt.Println("请在5秒内将鼠标移动到截图区域的一个角")

	// 等待用户准备
	time.Sleep(5 * time.Second)
	fmt.Println("5秒已过，请保持鼠标位置不变...")

	// 为了简化，我们使用全屏截图
	// 获取屏幕尺寸
	bounds := screenshot.GetDisplayBounds(0)
	
	fmt.Printf("屏幕尺寸: %dx%d\n", bounds.Dx(), bounds.Dy())

	// 获取全屏截图
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		fmt.Printf("截图失败: %v\n", err)
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("screenshot_%s.png", time.Now().Format("20060102_150405"))

	// 保存截图
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("创建文件失败: %v\n", err)
		return
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		fmt.Printf("保存截图失败: %v\n", err)
		return
	}

	fmt.Printf("全屏截图已保存为: %s\n", filename)
	fmt.Println("程序将退出...")
}
