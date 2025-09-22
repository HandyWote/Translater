package main

import (
	"fmt"
	"log"

	"Translater/ai"
	"Translater/config"
	"Translater/hotkey"
	"Translater/screenshot"
	"Translater/service"
)

func main() {
	// 创建API密钥读取器
	envFiles := []string{".env", "env"}
	apiKeyReader := config.NewFileAPIKeyReader(envFiles)

	// 读取API密钥
	apiKey, err := apiKeyReader.ReadAPIKey()
	if err != nil {
		log.Fatal("Could not load API key from any source")
	}
	fmt.Println("Successfully read API key from file")

	// 创建热键管理器
	hotkeyManager := hotkey.NewManager()

	// 创建截图管理器（只创建一次）
	screenshotManager := screenshot.NewManager()

	// 创建AI客户端
	aiClient := ai.NewZhipuAIClient(apiKey)

	// 创建翻译服务
	translationService := service.NewTranslationService(aiClient)

	// 设置截图处理函数
	screenshotManager.SetCaptureHandler(func(startX, startY, endX, endY int) bool {
		// 使用翻译服务处理截图
		return translationService.ProcessScreenshot(startX, startY, endX, endY)
	})

	// 注册热键，当触发时启动截图
	if err := hotkeyManager.Register(1, hotkey.MOD_ALT, hotkey.VK_T, func() {
		fmt.Println("热键触发，启动截图...")
		// 在新的 goroutine 中启动截图，避免阻塞热键监听
		go screenshotManager.StartOnce()
	}); err != nil {
		log.Fatalf("注册热键失败: %v", err)
	}

	// 启动热键监听
	hotkeyManager.Start()
}
