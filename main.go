package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"Translater/hotkey"
	"Translater/screenshot"
	"Translater/ai"
)

// readAPIKeyFromFile 直接从文件读取API密钥
func readAPIKeyFromFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "API-KEY") {
			parts := strings.Split(line, "=")
			if len(parts) >= 2 {
				apiKey := strings.TrimSpace(parts[1])
				// 移除引号
				apiKey = strings.Trim(apiKey, "\"")
				return apiKey, nil
			}
		}
	}

	return "", fmt.Errorf("API-KEY not found in file")
}

func main() {
	// 尝试从不同位置的文件读取API密钥
	envFiles := []string{".env", "env"}
	var apiKey string
	var err error
	
	for _, envFile := range envFiles {
		apiKey, err = readAPIKeyFromFile(envFile)
		if err == nil {
			fmt.Printf("Successfully read API key from %s file\n", envFile)
			break
		}
	}
	
	if apiKey == "" {
		log.Fatal("Could not load API key from any source")
	}

	// 创建热键管理器
	hotkeyManager := hotkey.NewManager()

	// 创建截图管理器（只创建一次）
	screenshotManager := screenshot.NewManager()

	// 创建AI客户端
	aiClient := ai.NewZhipuAIClient(apiKey)

	// 设置截图处理函数
	screenshotManager.SetCaptureHandler(func(startX, startY, endX, endY int) bool {
		fmt.Println("开始处理截图...")
		
		// 使用新的CaptureToBytes函数获取图像数据，不保存文件
		imageData, err := screenshot.CaptureToBytes(startX, startY, endX, endY)
		if err != nil {
			fmt.Printf("截图失败: %v\n", err)
			return false
		}
		
		fmt.Println("截图成功，开始提取文字...")
		
		// 使用ImageToWordsFromBytes函数提取文字
		extractPrompt := "请提取这张图片中的所有文字内容，只返回文字，不要添加任何其他说明。"
		extractResponse, err := aiClient.ImageToWordsFromBytes(extractPrompt, imageData, "image/png", "")
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
			translatePrompt := "请将以下文本翻译成中文，保持原文的格式和结构："
			translateResponse, err := aiClient.Translate(textStr, translatePrompt)
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
