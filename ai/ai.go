package ai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ZhipuAIRequest 表示发送给智谱AI的请求结构
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

// ZhipuAIResponse 表示智谱AI的响应结构
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

// ZhipuAIClient 智谱AI客户端
type ZhipuAIClient struct {
	APIKey string
	Client *http.Client
}

// NewZhipuAIClient 创建新的智谱AI客户端
func NewZhipuAIClient(apiKey string) *ZhipuAIClient {
	return &ZhipuAIClient{
		APIKey: apiKey,
		Client: &http.Client{},
	}
}

// PostRequest 发送POST请求到智谱AI
func (c *ZhipuAIClient) PostRequest(request ZhipuAIRequest) (*ZhipuAIResponse, error) {
	// 智谱AI API端点
	url := "https://open.bigmodel.cn/api/paas/v4/chat/completions"

	// 将请求结构转换为JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	// 发送请求
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应JSON
	var response ZhipuAIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	// 检查API错误
	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s - %s", response.Error.Code, response.Error.Message)
	}

	return &response, nil
}

func (c *ZhipuAIClient) Translate(userMessage string, systemPrompt string) (*ZhipuAIResponse, error) {
	messages := []Message{}

	// 添加系统提示词（如果提供）
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// 添加用户消息
	messages = append(messages, Message{
		Role:    "user",
		Content: userMessage,
	})

	request := ZhipuAIRequest{
		Model:       "glm-4.5-flash",
		Messages:    messages,
		Temperature: 1,
		TopP:        0.9,
	}

	return c.PostRequest(request)
}

func (c *ZhipuAIClient) ImageToWords(userMessage string, imagePath string, systemPrompt string) (*ZhipuAIResponse, error) {
	messages := []Message{}

	// 添加系统提示词（如果提供）
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// 构建多模态内容
	content := []ContentItem{
		{
			Type: "text",
			Text: userMessage,
		},
	}

	var data []byte
	var mimeType string

	// 将本地图片转换为data URL
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %v", err)
	}
	defer file.Close()

	// 读取文件数据
	data, err = io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %v", err)
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(imagePath))

	// 根据文件类型确定MIME类型
	switch ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	case ".gif":
		mimeType = "image/gif"
	case ".webp":
		mimeType = "image/webp"
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}

	// 进行base64编码
	encoded := base64.StdEncoding.EncodeToString(data)

	// 构建data URL
	imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	// 添加图像到内容
	content = append(content, ContentItem{
		Type: "image_url",
		ImageURL: struct {
			URL string `json:"url"`
		}{
			URL: imageURL,
		},
	})

	// 添加用户消息
	messages = append(messages, Message{
		Role:    "user",
		Content: content,
	})

	request := ZhipuAIRequest{
		Model:       "glm-4v-flash",
		Messages:    messages,
		Temperature: 0.7,
		TopP:        0.9,
	}

	return c.PostRequest(request)
}

// ImageToWordsFromBytes 直接从图像字节数据提取文字
func (c *ZhipuAIClient) ImageToWordsFromBytes(userMessage string, imageData []byte, mimeType string, systemPrompt string) (*ZhipuAIResponse, error) {
	messages := []Message{}

	// 添加系统提示词（如果提供）
	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// 构建多模态内容
	content := []ContentItem{
		{
			Type: "text",
			Text: userMessage,
		},
	}

	// 进行base64编码
	encoded := base64.StdEncoding.EncodeToString(imageData)

	// 构建data URL
	imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)

	// 添加图像到内容
	content = append(content, ContentItem{
		Type: "image_url",
		ImageURL: struct {
			URL string `json:"url"`
		}{
			URL: imageURL,
		},
	})

	// 添加用户消息
	messages = append(messages, Message{
		Role:    "user",
		Content: content,
	})

	request := ZhipuAIRequest{
		Model:       "glm-4v-flash",
		Messages:    messages,
		Temperature: 0.7,
		TopP:        0.9,
	}

	return c.PostRequest(request)
}
