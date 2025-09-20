package service

import (
	"Translater/ai"
	"Translater/prompts"
	"Translater/screenshot"
	"fmt"
)

// TranslationService 翻译服务接口
type TranslationService interface {
	ProcessScreenshot(startX, startY, endX, endY int) bool
}

// TranslationServiceImpl 翻译服务实现
type TranslationServiceImpl struct {
	AIClient *ai.ZhipuAIClient
}

// NewTranslationService 创建新的翻译服务
func NewTranslationService(aiClient *ai.ZhipuAIClient) TranslationService {
	return &TranslationServiceImpl{
		AIClient: aiClient,
	}
}

// ProcessScreenshot 处理截图：截图->提取文字->翻译
func (s *TranslationServiceImpl) ProcessScreenshot(startX, startY, endX, endY int) bool {
	fmt.Println("开始处理截图...")

	// 使用新的CaptureToBytes函数获取图像数据，不保存文件
	imageData, err := screenshot.CaptureToBytes(startX, startY, endX, endY)
	if err != nil {
		fmt.Printf("截图失败: %v\n", err)
		return false
	}

	fmt.Println("截图成功，开始提取文字...")

	// 使用ImageToWordsFromBytes函数提取文字
	extractResponse, err := s.AIClient.ImageToWords(prompts.ExtractPrompt, imageData, "image/png", "")
	if err != nil {
		fmt.Printf("文字提取失败: %v\n", err)
		return false
	}

	// 获取提取的文字内容
	extractedText := extractResponse.Choices[0].Message.Content
	var textStr string
	if str, ok := extractedText.(string); ok {
		textStr = str
	} else {
		fmt.Printf("提取的文字内容格式错误: %v\n", extractedText)
		return false
	}

	fmt.Printf("提取到的文字: %s\n", textStr)

	// 如果提取到了文字，则进行翻译
	if textStr != "" {
		fmt.Println("开始翻译...")

		// 使用Translate函数翻译成中文
		translateResponse, err := s.AIClient.Translate(textStr, prompts.TranslatePrompt)
		if err != nil {
			fmt.Printf("翻译失败: %v\n", err)
			return false
		}

		// 获取翻译结果
		translatedText := translateResponse.Choices[0].Message.Content
		fmt.Printf("翻译结果: %s\n", translatedText)

		// 这里可以添加将翻译结果显示给用户的逻辑，比如弹出窗口或复制到剪贴板
		fmt.Println("处理完成！")
	} else {
		fmt.Println("未提取到文字内容")
	}

	return true
}
