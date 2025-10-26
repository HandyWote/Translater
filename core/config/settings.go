package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"Translater/core/ai"
	"Translater/core/prompts"
)

// Settings 保存桌面端可配置项
type Settings struct {
	APIKeyOverride          string `json:"apiKeyOverride"`
	AutoCopyResult          bool   `json:"autoCopyResult"`
	KeepWindowOnTop         bool   `json:"keepWindowOnTop"`
	Theme                   string `json:"theme"`
	ShowToastOnComplete     bool   `json:"showToastOnComplete"`
	EnableStreamOutput      bool   `json:"enableStreamOutput"`
	HotkeyCombination       string `json:"hotkeyCombination"`
	ExtractPrompt           string `json:"extractPrompt"`
	TranslatePrompt         string `json:"translatePrompt"`
	APIBaseURL              string `json:"apiBaseUrl"`
	TranslateModel          string `json:"translateModel"`
	VisionModel             string `json:"visionModel"`
	VisionAPIBaseURL        string `json:"visionApiBaseUrl"`
	VisionAPIKeyOverride    string `json:"visionApiKeyOverride"`
	UseVisionForTranslation bool   `json:"useVisionForTranslation"`
}

// DefaultSettings 返回默认配置
func DefaultSettings() Settings {
	return Settings{
		AutoCopyResult:          true,
		KeepWindowOnTop:         false,
		Theme:                   "system",
		ShowToastOnComplete:     true,
		EnableStreamOutput:      true,
		HotkeyCombination:       "Alt+T",
		ExtractPrompt:           prompts.DefaultExtractPrompt,
		TranslatePrompt:         prompts.DefaultTranslatePrompt,
		APIBaseURL:              ai.DefaultBaseURL,
		TranslateModel:          ai.DefaultTranslateModel,
		VisionModel:             ai.DefaultVisionModel,
		VisionAPIBaseURL:        ai.DefaultBaseURL,
		UseVisionForTranslation: true,
	}
}

// SettingsManager 管理配置文件的读写
type SettingsManager struct {
	path string
	mu   sync.RWMutex
}

// NewSettingsManager 创建配置管理器，配置存储于用户配置目录下
func NewSettingsManager(appName string) (*SettingsManager, error) {
	path, err := resolveSettingsPath(appName)
	if err != nil {
		return nil, err
	}
	return &SettingsManager{path: path}, nil
}

// Path 返回配置文件路径
func (m *SettingsManager) Path() string {
	return m.path
}

// Load 读取配置，不存在时返回默认配置
func (m *SettingsManager) Load() (Settings, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	defaults := DefaultSettings()
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults, nil
		}
		return defaults, err
	}

	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaults, err
	}

	applySettingsDefaults(&settings)
	return settings, nil
}

// Save 持久化配置
func (m *SettingsManager) Save(settings Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return err
	}

	applySettingsDefaults(&settings)

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.path, data, 0o600)
}

func resolveSettingsPath(appName string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, appName)
	return filepath.Join(appDir, "settings.json"), nil
}

func applySettingsDefaults(settings *Settings) {
	defaults := DefaultSettings()
	settings.APIKeyOverride = strings.TrimSpace(settings.APIKeyOverride)
	if settings.Theme == "" {
		settings.Theme = defaults.Theme
	}
	if strings.TrimSpace(settings.HotkeyCombination) == "" {
		settings.HotkeyCombination = defaults.HotkeyCombination
	}
	if strings.TrimSpace(settings.ExtractPrompt) == "" {
		settings.ExtractPrompt = defaults.ExtractPrompt
	}
	if strings.TrimSpace(settings.TranslatePrompt) == "" {
		settings.TranslatePrompt = defaults.TranslatePrompt
	}
	settings.APIBaseURL = ai.NormalizeBaseURL(settings.APIBaseURL)
	settings.TranslateModel = strings.TrimSpace(settings.TranslateModel)
	if settings.TranslateModel == "" {
		settings.TranslateModel = defaults.TranslateModel
	}
	settings.VisionModel = strings.TrimSpace(settings.VisionModel)
	if settings.VisionModel == "" {
		settings.VisionModel = defaults.VisionModel
	}
	settings.VisionAPIBaseURL = strings.TrimSpace(settings.VisionAPIBaseURL)
	if settings.VisionAPIBaseURL == "" {
		settings.VisionAPIBaseURL = settings.APIBaseURL
	} else {
		settings.VisionAPIBaseURL = ai.NormalizeBaseURL(settings.VisionAPIBaseURL)
	}
	settings.VisionAPIKeyOverride = strings.TrimSpace(settings.VisionAPIKeyOverride)
}
