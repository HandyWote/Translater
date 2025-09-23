package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// APIKeyReader 用于读取API密钥的接口
type APIKeyReader interface {
	ReadAPIKey() (string, error)
}

// FileAPIKeyReader 从文件读取API密钥的实现
type FileAPIKeyReader struct {
	EnvFiles []string
}

// NewFileAPIKeyReader 创建新的文件API密钥读取器
func NewFileAPIKeyReader(envFiles []string) *FileAPIKeyReader {
	return &FileAPIKeyReader{
		EnvFiles: envFiles,
	}
}

// ReadAPIKeyFromFile 直接从文件读取API密钥
func (r *FileAPIKeyReader) ReadAPIKeyFromFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "API-KEY") {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) >= 2 {
			apiKey := strings.TrimSpace(parts[1])
			return apiKey, nil
		}
	}

	return "", fmt.Errorf("API-KEY not found in file")
}

// ReadAPIKey 从配置的文件中读取API密钥
func (r *FileAPIKeyReader) ReadAPIKey() (string, error) {
	var apiKey string
	var err error

	for _, envFile := range r.EnvFiles {
		apiKey, err = r.ReadAPIKeyFromFile(envFile)
		if err == nil {
			return apiKey, nil
		}
	}

	return "", fmt.Errorf("could not load API key from any source")
}