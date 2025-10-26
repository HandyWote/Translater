package translation

import (
	"fmt"
	"strings"
	"time"

	"Translater/core/ai"
	"Translater/core/prompts"
	"Translater/core/screenshot"
)

// Service 翻译服务接口
type Service interface {
	ProcessScreenshot(startX, startY, endX, endY int) bool
	ProcessScreenshotDetailed(startX, startY, endX, endY int) (*ScreenshotTranslationResult, error)
	TranslateText(input string) (*TextTranslationResult, error)
	UpdatePrompts(extract, translate string)
	UpdateOptions(opts Options)
	SetStreamHandler(handler StreamHandler)
}

// ServiceImpl 翻译服务实现
type ServiceImpl struct {
	AIClient        *ai.Client
	extractPrompt   string
	translatePrompt string
	options         Options
	streamHandler   StreamHandler
}

// StreamHandler 用于接收翻译过程中的流式文本
type StreamHandler func(stage string, content string)

// Options 控制翻译服务行为
type Options struct {
	Stream                  bool
	UseVisionForTranslation bool
	SourceLanguage          string
	TargetLanguage          string
}

// ScreenshotTranslationResult 包含一次截图翻译的详情
type ScreenshotTranslationResult struct {
	ExtractedText   string
	TranslatedText  string
	ExtractPrompt   string
	TranslatePrompt string
	ProcessingTime  time.Duration
	Bounds          ScreenshotBounds
}

// ScreenshotBounds 描述一次截图对应的屏幕区域
type ScreenshotBounds struct {
	StartX int `json:"startX"`
	StartY int `json:"startY"`
	EndX   int `json:"endX"`
	EndY   int `json:"endY"`
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// TextTranslationResult 记录纯文本翻译的结果
type TextTranslationResult struct {
	OriginalText    string
	TranslatedText  string
	TranslatePrompt string
	ProcessingTime  time.Duration
}

// NewService 创建新的翻译服务
func NewService(aiClient *ai.Client, extractPrompt, translatePrompt string, opts Options) Service {
	return &ServiceImpl{
		AIClient:        aiClient,
		extractPrompt:   normalisePrompt(extractPrompt, prompts.DefaultExtractPrompt),
		translatePrompt: normalisePrompt(translatePrompt, prompts.DefaultTranslatePrompt),
		options:         opts,
	}
}

func normalisePrompt(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return fallback
}

func newScreenshotBounds(startX, startY, endX, endY int) ScreenshotBounds {
	left := startX
	right := endX
	if left > right {
		left, right = right, left
	}
	top := startY
	bottom := endY
	if top > bottom {
		top, bottom = bottom, top
	}
	width := right - left
	height := bottom - top
	if width <= 0 {
		width = 1
	}
	if height <= 0 {
		height = 1
	}

	return ScreenshotBounds{
		StartX: startX,
		StartY: startY,
		EndX:   endX,
		EndY:   endY,
		Left:   left,
		Top:    top,
		Width:  width,
		Height: height,
	}
}

// ProcessScreenshot 处理截图：截图->提取文字->翻译
func (s *ServiceImpl) ProcessScreenshot(startX, startY, endX, endY int) bool {
	fmt.Println("开始处理截图...")

	result, err := s.ProcessScreenshotDetailed(startX, startY, endX, endY)
	if err != nil {
		fmt.Printf("截图处理失败: %v\n", err)
		return false
	}

	fmt.Printf("提取到的文字: %s\n", result.ExtractedText)
	if result.TranslatedText != "" {
		fmt.Printf("翻译结果: %s\n", result.TranslatedText)
		fmt.Println("处理完成！")
	} else {
		fmt.Println("未提取到文字内容")
	}

	return true
}

// ProcessScreenshotDetailed 执行截图、OCR、翻译并返回完整结果
func (s *ServiceImpl) ProcessScreenshotDetailed(startX, startY, endX, endY int) (*ScreenshotTranslationResult, error) {
	if s.AIClient == nil {
		return nil, fmt.Errorf("AI client 未初始化")
	}

	started := time.Now()
	bounds := newScreenshotBounds(startX, startY, endX, endY)

	imageData, err := screenshot.CaptureToBytes(startX, startY, endX, endY)
	if err != nil {
		return nil, fmt.Errorf("截图失败: %w", err)
	}

	// 创建提示词变量
	vars := prompts.PromptVariables{
		SourceLanguage:          s.options.SourceLanguage,
		TargetLanguage:          s.options.TargetLanguage,
		UseVisionForTranslation: s.options.UseVisionForTranslation,
	}

	// 处理动态提示词
	processedExtractPrompt := prompts.ProcessExtractPrompt(s.extractPrompt, vars)
	processedTranslatePrompt := prompts.ProcessTranslatePrompt(s.translatePrompt, vars)

	result := &ScreenshotTranslationResult{
		ExtractPrompt:   processedExtractPrompt,
		TranslatePrompt: processedTranslatePrompt,
		Bounds:          bounds,
	}

	streamEnabled := s.options.Stream && s.streamHandler != nil

	// 视觉直出翻译模式
	if s.options.UseVisionForTranslation {
		directPrompt := prompts.BuildVisionDirectTranslationPrompt(vars)
		if streamEnabled {
			translateResponse, err := s.AIClient.ImageToTranslationStream(
				directPrompt,
				imageData,
				"image/png",
				"",
				func(text string) {
					s.emitStream("translate", text)
				},
			)
			if err != nil {
				return nil, fmt.Errorf("视觉直出翻译失败: %w", err)
			}
			if len(translateResponse.Choices) == 0 {
				return nil, fmt.Errorf("视觉直出翻译结果为空")
			}
			translatedText, err := messageContentToString(translateResponse.Choices[0].Message.Content)
			if err != nil {
				return nil, fmt.Errorf("翻译内容解析失败: %w", err)
			}
			result.TranslatedText = translatedText
		} else {
			translateResponse, err := s.AIClient.ImageToTranslation(
				directPrompt,
				imageData,
				"image/png",
				"",
			)
			if err != nil {
				return nil, fmt.Errorf("视觉直出翻译失败: %w", err)
			}
			if len(translateResponse.Choices) == 0 {
				return nil, fmt.Errorf("视觉直出翻译结果为空")
			}
			translatedText, err := messageContentToString(translateResponse.Choices[0].Message.Content)
			if err != nil {
				return nil, fmt.Errorf("翻译内容解析失败: %w", err)
			}
			result.TranslatedText = translatedText
		}
	} else {
		// 传统模式：先提取，再翻译
		
		// OCR 阶段
		extractResponse, err := s.AIClient.ImageToWords(processedExtractPrompt, imageData, "image/png", "")
		if err != nil {
			return nil, fmt.Errorf("文字提取失败: %w", err)
		}

		if len(extractResponse.Choices) == 0 {
			return nil, fmt.Errorf("文字提取结果为空")
		}

		extractedText, err := messageContentToString(extractResponse.Choices[0].Message.Content)
		if err != nil {
			return nil, fmt.Errorf("提取内容解析失败: %w", err)
		}

		result.ExtractedText = extractedText

		if strings.TrimSpace(extractedText) == "" {
			result.ProcessingTime = time.Since(started)
			return result, nil
		}

		// 翻译阶段
		var translateResponse *ai.ZhipuAIResponse
		if streamEnabled {
			translateResponse, err = s.AIClient.TranslateStream(
				extractedText,
				processedTranslatePrompt,
				func(text string) {
					s.emitStream("translate", text)
				},
			)
		} else {
			translateResponse, err = s.AIClient.Translate(extractedText, processedTranslatePrompt)
		}
		if err != nil {
			return nil, fmt.Errorf("翻译失败: %w", err)
		}

		if len(translateResponse.Choices) == 0 {
			return nil, fmt.Errorf("翻译结果为空")
		}

		translatedText, err := messageContentToString(translateResponse.Choices[0].Message.Content)
		if err != nil {
			return nil, fmt.Errorf("翻译内容解析失败: %w", err)
		}

		result.TranslatedText = translatedText
	}

	result.ProcessingTime = time.Since(started)
	return result, nil
}

// TranslateText 翻译纯文本
func (s *ServiceImpl) TranslateText(input string) (*TextTranslationResult, error) {
	if s.AIClient == nil {
		return nil, fmt.Errorf("AI client 未初始化")
	}

	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("翻译内容不能为空")
	}

	started := time.Now()
	streamEnabled := s.options.Stream && s.streamHandler != nil

	// 创建提示词变量
	vars := prompts.PromptVariables{
		SourceLanguage:          s.options.SourceLanguage,
		TargetLanguage:          s.options.TargetLanguage,
		UseVisionForTranslation: s.options.UseVisionForTranslation,
	}

	// 处理动态提示词
	processedTranslatePrompt := prompts.ProcessTranslatePrompt(s.translatePrompt, vars)

	var translateResponse *ai.ZhipuAIResponse
	var err error
	if streamEnabled {
		translateResponse, err = s.AIClient.TranslateStream(
			input,
			processedTranslatePrompt,
			func(text string) {
				s.emitStream("translate", text)
			},
		)
	} else {
		translateResponse, err = s.AIClient.Translate(input, processedTranslatePrompt)
	}
	if err != nil {
		return nil, fmt.Errorf("翻译失败: %w", err)
	}

	if len(translateResponse.Choices) == 0 {
		return nil, fmt.Errorf("翻译结果为空")
	}

	translatedText, err := messageContentToString(translateResponse.Choices[0].Message.Content)
	if err != nil {
		return nil, fmt.Errorf("翻译内容解析失败: %w", err)
	}

	return &TextTranslationResult{
		OriginalText:    input,
		TranslatedText:  translatedText,
		TranslatePrompt: processedTranslatePrompt,
		ProcessingTime:  time.Since(started),
	}, nil
}

// UpdatePrompts 允许在运行时刷新提示词配置。
func (s *ServiceImpl) UpdatePrompts(extract, translate string) {
	s.extractPrompt = normalisePrompt(extract, prompts.DefaultExtractPrompt)
	s.translatePrompt = normalisePrompt(translate, prompts.DefaultTranslatePrompt)
}

// UpdateOptions 更新服务运行参数
func (s *ServiceImpl) UpdateOptions(opts Options) {
	s.options = opts
}

// SetStreamHandler 配置流式输出回调
func (s *ServiceImpl) SetStreamHandler(handler StreamHandler) {
	s.streamHandler = handler
}

func (s *ServiceImpl) emitStream(stage, content string) {
	if !s.options.Stream || s.streamHandler == nil {
		return
	}
	s.streamHandler(stage, content)
}

// buildVisionTranslationMessage 保留此函数以保持向后兼容性，但不再使用
// 已被新的视觉直出翻译模式替代
func (s *ServiceImpl) buildVisionTranslationMessage(extractedText string) string {
	var builder strings.Builder
	builder.WriteString(s.translatePrompt)

	trimmed := strings.TrimSpace(extractedText)
	if trimmed != "" {
		builder.WriteString("\n\n以下是图像解析得到的原文 JSON：\n")
		builder.WriteString(trimmed)
	}
	builder.WriteString("\n\n请结合图像内容直接输出翻译后的文字，不要包含任何额外说明。")
	return builder.String()
}

func messageContentToString(content interface{}) (string, error) {
	switch value := content.(type) {
	case string:
		return value, nil
	case []byte:
		return string(value), nil
	case fmt.Stringer:
		return value.String(), nil
	default:
		return "", fmt.Errorf("不支持的消息内容类型: %T", content)
	}
}
