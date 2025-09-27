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
}

// ServiceImpl 翻译服务实现
type ServiceImpl struct {
	AIClient        *ai.Client
	extractPrompt   string
	translatePrompt string
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
func NewService(aiClient *ai.Client, extractPrompt, translatePrompt string) Service {
	return &ServiceImpl{
		AIClient:        aiClient,
		extractPrompt:   normalisePrompt(extractPrompt, prompts.DefaultExtractPrompt),
		translatePrompt: normalisePrompt(translatePrompt, prompts.DefaultTranslatePrompt),
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

	// OCR 阶段
	extractResponse, err := s.AIClient.ImageToWords(s.extractPrompt, imageData, "image/png", "")
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

	result := &ScreenshotTranslationResult{
		ExtractedText:   extractedText,
		ExtractPrompt:   s.extractPrompt,
		TranslatePrompt: s.translatePrompt,
		Bounds:          bounds,
	}

	if strings.TrimSpace(extractedText) == "" {
		result.ProcessingTime = time.Since(started)
		return result, nil
	}

	// 翻译阶段
	translateResponse, err := s.AIClient.Translate(extractedText, s.translatePrompt)
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
	translateResponse, err := s.AIClient.Translate(input, s.translatePrompt)
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
		TranslatePrompt: s.translatePrompt,
		ProcessingTime:  time.Since(started),
	}, nil
}

// UpdatePrompts 允许在运行时刷新提示词配置。
func (s *ServiceImpl) UpdatePrompts(extract, translate string) {
	s.extractPrompt = normalisePrompt(extract, prompts.DefaultExtractPrompt)
	s.translatePrompt = normalisePrompt(translate, prompts.DefaultTranslatePrompt)
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
