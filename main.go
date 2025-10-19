package main

import (
	"fmt"
	"log"
	"strings"

	"Translater/core/ai"
	"Translater/core/config"
	"Translater/core/hotkey"
	"Translater/core/screenshot"
	"Translater/core/translation"
)

func main() {
	// 创建API密钥读取器
	envFiles := []string{".env", "env"}
	apiKeyReader := config.NewFileAPIKeyReader(envFiles)

	settings := config.DefaultSettings()
	if manager, err := config.NewSettingsManager("Translater"); err != nil {
		log.Printf("failed to resolve settings path: %v", err)
	} else if loaded, err := manager.Load(); err != nil {
		log.Printf("failed to load settings, using defaults: %v", err)
	} else {
		settings = loaded
	}

	apiKey := strings.TrimSpace(settings.APIKeyOverride)
	if apiKey != "" {
		fmt.Println("Using API key from settings override")
	} else {
		// 读取API密钥
		var err error
		apiKey, err = apiKeyReader.ReadAPIKey()
		if err != nil {
			log.Fatal("Could not load API key from any source")
		}
		fmt.Println("Successfully read API key from file")
	}

	if apiKey == "" {
		log.Fatal("API key is empty")
	}

	visionAPIKey := strings.TrimSpace(settings.VisionAPIKeyOverride)
	if visionAPIKey == "" {
		visionAPIKey = apiKey
	}
	visionBaseURL := strings.TrimSpace(settings.VisionAPIBaseURL)
	if visionBaseURL == "" {
		visionBaseURL = settings.APIBaseURL
	}

	// 创建热键管理器
	hotkeyManager := hotkey.NewManager()

	// 创建截图管理器（只创建一次）
	screenshotManager := screenshot.NewManager()

	// 创建AI客户端
	aiClient := ai.NewClient(ai.ClientConfig{
		APIKey:         apiKey,
		BaseURL:        settings.APIBaseURL,
		TranslateModel: settings.TranslateModel,
		VisionModel:    settings.VisionModel,
		VisionAPIKey:   visionAPIKey,
		VisionBaseURL:  visionBaseURL,
	})

	// 创建翻译服务
	translationService := translation.NewService(
		aiClient,
		settings.ExtractPrompt,
		settings.TranslatePrompt,
		translation.Options{
			Stream:                  settings.EnableStreamOutput,
			UseVisionForTranslation: settings.UseVisionForTranslation,
		},
	)

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
