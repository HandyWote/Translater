package main

import (
	"Translater/hotkey"
	"Translater/screenshot"
)

func main() {
	// 创建热键管理器
	hotkeyManager := hotkey.NewManager()

	// 创建截图管理器
	screenshotManager := screenshot.NewManager()

	// 设置截图处理函数
	screenshotManager.SetCaptureHandler(func(startX, startY, endX, endY int) bool {
		return screenshot.Capture(startX, startY, endX, endY)
	})

	// 注册热键，当触发时启动截图
	hotkeyManager.Register(1, hotkey.MOD_ALT, hotkey.VK_T, func() {
		// 当热键被触发时，启动截图监听
		go screenshotManager.Start()
	})

	// 启动热键监听
	hotkeyManager.Start()
}