package main

import (
	"fmt"
	"Translater/hotkey"
	"Translater/screenshot"
)

func main() {
	// 创建热键管理器
	hotkeyManager := hotkey.NewManager()

	// 创建截图管理器（只创建一次）
	screenshotManager := screenshot.NewManager()

	// 设置截图处理函数
	screenshotManager.SetCaptureHandler(func(startX, startY, endX, endY int) bool {
		return screenshot.Capture(startX, startY, endX, endY)
	})

	// 注册热键，当触发时启动截图
	hotkeyManager.Register(1, hotkey.MOD_ALT, hotkey.VK_T, func() {
		fmt.Println("热键触发，启动截图...")
		// 在新的 goroutine 中启动截图，避免阻塞热键监听
		go screenshotManager.StartOnce()
	})

	// 启动热键监听
	hotkeyManager.Start()
}