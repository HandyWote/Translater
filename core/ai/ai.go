package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// DefaultBaseURL 是默认的 OpenAI 兼容接口地址
	DefaultBaseURL = "https://open.bigmodel.cn/api/paas/v4"
	// DefaultTranslateModel 是默认的文本翻译模型
	DefaultTranslateModel = "glm-4.5-flash"
	// DefaultVisionModel 是默认的视觉理解模型
	DefaultVisionModel = "glm-4v-flash"
)

// ClientConfig 描述创建客户端的必要信息
type ClientConfig struct {
	APIKey         string
	BaseURL        string
	TranslateModel string
	VisionModel    string
	HTTPClient     *http.Client
	VisionAPIKey   string
	VisionBaseURL  string
}

type endpoint struct {
	apiKey string
	base   string
	model  string
}

// Client 表示一个兼容 OpenAI Chat Completions 的客户端
type Client struct {
	translate  endpoint
	vision     endpoint
	httpClient *http.Client
}

// NormalizeBaseURL 处理用户输入的 BaseURL，保证格式统一
func NormalizeBaseURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		trimmed = DefaultBaseURL
	}
	return strings.TrimRight(trimmed, "/")
}

func normalizeModel(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func normalizeBaseURLOrFallback(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return strings.TrimRight(trimmed, "/")
}

// NewClient 创建一个新的通用 AI 客户端
func NewClient(cfg ClientConfig) *Client {
	baseURL := NormalizeBaseURL(cfg.BaseURL)
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	translateAPIKey := strings.TrimSpace(cfg.APIKey)
	visionAPIKey := strings.TrimSpace(cfg.VisionAPIKey)
	if visionAPIKey == "" {
		visionAPIKey = translateAPIKey
	}

	visionBase := normalizeBaseURLOrFallback(cfg.VisionBaseURL, baseURL)

	return &Client{
		translate: endpoint{
			apiKey: translateAPIKey,
			base:   baseURL,
			model:  normalizeModel(cfg.TranslateModel, DefaultTranslateModel),
		},
		vision: endpoint{
			apiKey: visionAPIKey,
			base:   visionBase,
			model:  normalizeModel(cfg.VisionModel, DefaultVisionModel),
		},
		httpClient: httpClient,
	}
}

// NewZhipuAIClient 保留旧接口，默认指向智谱开放平台
func NewZhipuAIClient(apiKey string) *Client {
	return NewClient(ClientConfig{APIKey: apiKey})
}

// BaseURL 返回归一化的 BaseURL
func (c *Client) BaseURL() string {
	return c.translate.base
}

// TranslateModel 返回归一化后的翻译模型名称
func (c *Client) TranslateModel() string {
	return c.translate.model
}

// VisionModel 返回归一化后的视觉模型名称
func (c *Client) VisionModel() string {
	return c.vision.model
}

// VisionBaseURL 返回视觉模型使用的接口地址
func (c *Client) VisionBaseURL() string {
	return c.vision.base
}

func (c *Client) chatCompletionsURL(base string) string {
	return base + "/chat/completions"
}

// ZhipuAIRequest 表示发送给兼容接口的请求结构
// 保留原命名以避免大规模重构
// TODO: 未来可考虑重命名为 ChatCompletionRequest
type ZhipuAIRequest struct {
	Model       string                 `json:"model"`
	Messages    []Message              `json:"messages"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	TopP        float64                `json:"top_p,omitempty"`
	OtherParams map[string]interface{} `json:"other_params,omitempty"`
}

// ContentItem 表示消息内容项，可以是文本或图像
type ContentItem struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

// Message 表示对话消息结构
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // 可以是string（文本）或[]ContentItem（多模态）
}

// ZhipuAIResponse 表示兼容接口的响应结构
type ZhipuAIResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice 表示响应中的选择项
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 表示token使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError 表示API错误
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (c *Client) post(request ZhipuAIRequest, target endpoint) (*ZhipuAIResponse, error) {
	url := c.chatCompletionsURL(target.base)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if target.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+target.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response ZhipuAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
	}

	return &response, nil
}

// Translate 发送文本消息至聊天接口
func (c *Client) Translate(userMessage string, systemPrompt string) (*ZhipuAIResponse, error) {
	messages := []Message{}

	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	messages = append(messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	request := ZhipuAIRequest{
		Model:       c.translate.model,
		Messages:    messages,
		Temperature: 1,
		TopP:        0.9,
	}

	return c.post(request, c.translate)
}

// ImageToWords 直接从图像字节数据提取文字
func (c *Client) ImageToWords(userMessage string, imageData []byte, mimeType string, systemPrompt string) (*ZhipuAIResponse, error) {
	messages := []Message{}

	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	content := []ContentItem{
		{
			Type: "text",
			Text: userMessage,
		},
	}

	encoded := base64.StdEncoding.EncodeToString(imageData)
	imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	content = append(content, ContentItem{
		Type: "image_url",
		ImageURL: struct {
			URL string `json:"url"`
		}{
			URL: imageURL,
		},
	})

	messages = append(messages, Message{
		Role:    "user",
		Content: content,
	})

	request := ZhipuAIRequest{
		Model:       c.vision.model,
		Messages:    messages,
		Temperature: 0.7,
		TopP:        0.9,
	}

	return c.post(request, c.vision)
}
