package ai

import (
	"bufio"
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

type streamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []streamChoice `json:"choices"`
	Usage   *Usage         `json:"usage,omitempty"`
	Error   *APIError      `json:"error,omitempty"`
}

type streamChoice struct {
	Index        int         `json:"index"`
	Delta        streamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type streamDelta struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
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

func (c *Client) stream(request ZhipuAIRequest, target endpoint, onDelta func(string)) (*ZhipuAIResponse, error) {
	url := c.chatCompletionsURL(target.base)
	request.Stream = true

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var (
		builder      strings.Builder
		final        ZhipuAIResponse
		finishReason string
		sawChunk     bool
		dataBuffer   strings.Builder
	)

	processData := func(data string) error {
		if strings.TrimSpace(data) == "" {
			return nil
		}
		if data == "[DONE]" {
			return io.EOF
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return fmt.Errorf("failed to unmarshal stream chunk: %v (raw: %s)", err, data)
		}
		if chunk.Error != nil {
			return fmt.Errorf("API error: %s - %s", chunk.Error.Code, chunk.Error.Message)
		}
		if chunk.ID != "" && final.ID == "" {
			final.ID = chunk.ID
		}
		if chunk.Object != "" && final.Object == "" {
			final.Object = chunk.Object
		}
		if chunk.Created != 0 && final.Created == 0 {
			final.Created = chunk.Created
		}
		if chunk.Usage != nil {
			final.Usage = *chunk.Usage
		}

		for _, choice := range chunk.Choices {
			text := streamContentToString(choice.Delta.Content)
			if text != "" {
				builder.WriteString(text)
				if onDelta != nil {
					onDelta(builder.String())
				}
			}
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
		}

		sawChunk = true
		return nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRight(line, "\r")

		if strings.HasPrefix(line, ":") {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			segment := strings.TrimSpace(line[len("data:"):])
			if segment == "[DONE]" {
				if err := processData(segment); err == io.EOF {
					break
				}
				break
			}
			if dataBuffer.Len() > 0 {
				dataBuffer.WriteByte('\n')
			}
			dataBuffer.WriteString(segment)
			continue
		}

		if strings.TrimSpace(line) == "" {
			if dataBuffer.Len() == 0 {
				continue
			}
			payload := dataBuffer.String()
			dataBuffer.Reset()
			if err := processData(payload); err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			continue
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read stream: %v", err)
	}

	if dataBuffer.Len() > 0 {
		if err := processData(dataBuffer.String()); err != nil && err != io.EOF {
			return nil, err
		}
	}

	if !sawChunk {
		return nil, fmt.Errorf("stream response empty")
	}

	final.Choices = []Choice{
		{
			Index: 0,
			Message: Message{
				Role:    "assistant",
				Content: builder.String(),
			},
			FinishReason: finishReason,
		},
	}

	return &final, nil
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

// TranslateStream 以流式方式发送文本消息并回调增量内容
func (c *Client) TranslateStream(userMessage string, systemPrompt string, onDelta func(string)) (*ZhipuAIResponse, error) {
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

	return c.stream(request, c.translate, onDelta)
}

// ImageToWords 直接从图像字节数据提取文字
func (c *Client) ImageToWords(userMessage string, imageData []byte, mimeType string, systemPrompt string) (*ZhipuAIResponse, error) {
	request := ZhipuAIRequest{
		Model:       c.vision.model,
		Messages:    c.buildVisionMessages(userMessage, imageData, mimeType, systemPrompt),
		Temperature: 0.7,
		TopP:        0.9,
	}

	return c.post(request, c.vision)
}

// ImageToTranslation 使用视觉模型直接生成翻译结果
func (c *Client) ImageToTranslation(userMessage string, imageData []byte, mimeType string, systemPrompt string) (*ZhipuAIResponse, error) {
	request := ZhipuAIRequest{
		Model:       c.vision.model,
		Messages:    c.buildVisionMessages(userMessage, imageData, mimeType, systemPrompt),
		Temperature: 0.7,
		TopP:        0.9,
	}

	return c.post(request, c.vision)
}

// ImageToTranslationStream 使用视觉模型流式输出翻译结果
func (c *Client) ImageToTranslationStream(userMessage string, imageData []byte, mimeType string, systemPrompt string, onDelta func(string)) (*ZhipuAIResponse, error) {
	request := ZhipuAIRequest{
		Model:       c.vision.model,
		Messages:    c.buildVisionMessages(userMessage, imageData, mimeType, systemPrompt),
		Temperature: 0.7,
		TopP:        0.9,
	}

	return c.stream(request, c.vision, onDelta)
}

func (c *Client) buildVisionMessages(userMessage string, imageData []byte, mimeType string, systemPrompt string) []Message {
	messages := []Message{}

	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	content := []ContentItem{}
	if strings.TrimSpace(userMessage) != "" {
		content = append(content, ContentItem{
			Type: "text",
			Text: userMessage,
		})
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

	return messages
}

func streamContentToString(content interface{}) string {
	switch value := content.(type) {
	case string:
		return value
	case []interface{}:
		var builder strings.Builder
		for _, item := range value {
			switch piece := item.(type) {
			case map[string]interface{}:
				if pieceType, ok := piece["type"].(string); ok && pieceType == "text" {
					if text, ok := piece["text"].(string); ok {
						builder.WriteString(text)
					}
				}
			}
		}
		return builder.String()
	default:
		return ""
	}
}
