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

	// 解析主 API Key（视觉 API Key 优先）
	mainKey := strings.TrimSpace(settings.VisionAPIKeyOverride)
	if mainKey == "" {
		// 向后兼容：回退到 apiKeyOverride
		mainKey = strings.TrimSpace(settings.APIKeyOverride)
	}
	if mainKey == "" {
		// 最后尝试从文件读取
		var err error
		mainKey, err = apiKeyReader.ReadAPIKey()
		if err != nil || mainKey == "" {
			log.Fatal("需要配置视觉 API Key (visionApiKeyOverride) 或在 .env 文件中设置")
		}
		fmt.Println("Successfully read API key from file")
	} else {
		if strings.TrimSpace(settings.VisionAPIKeyOverride) != "" {
			fmt.Println("Using vision API key from settings")
		} else {
			fmt.Println("Using API key from settings override")
		}
	}

	// 解析翻译 API Key
	var translateKey string
	if settings.UseVisionForTranslation {
		// 视觉直出模式：翻译也用主 key
		translateKey = mainKey
		fmt.Println("Vision-only mode: using vision API key for translation")
	} else {
		// 文本模型模式：翻译 key 可选，留空则回退到主 key
		translateKey = strings.TrimSpace(settings.APIKeyOverride)
		if translateKey == "" {
			translateKey = mainKey
			fmt.Println("Text model mode: translation API key falls back to vision API key")
		} else {
			fmt.Println("Text model mode: using separate translation API key")
		}
	}

	// 视觉 API 配置
	visionAPIKey := mainKey
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
		APIKey:         translateKey,
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
			SourceLanguage:          settings.SourceLanguage,
			TargetLanguage:          settings.TargetLanguage,
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
