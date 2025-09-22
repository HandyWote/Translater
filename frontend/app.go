package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"Translater/ai"
	"Translater/config"
	"Translater/hotkey"
	"Translater/screenshot"
	"Translater/service"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventTranslationStarted  = "translation:started"
	eventTranslationProgress = "translation:progress"
	eventTranslationResult   = "translation:result"
	eventTranslationError    = "translation:error"
	eventTranslationIdle     = "translation:idle"
	eventTranslationCopied   = "translation:copied"
	eventSettingsTheme       = "settings:theme"
	eventSettingsUpdated     = "settings:updated"
	eventConfigMissingKey    = "config:missing_api_key"
	eventConfigReady         = "config:api_key_ready"
)

// App 提供给 Wails 的后端逻辑
type App struct {
	ctx context.Context

	settingsManager *config.SettingsManager
	settings        config.Settings

	translationSvc     service.TranslationService
	screenshotMgr      *screenshot.Manager
	currentAPIKey      string
	screenshotLocker   sync.Mutex
	screenshotActive   bool
	hotkeyMgr          *hotkey.Manager
	hotkeyMutex        sync.Mutex
	hotkeyLoopOnce     sync.Once
	hotkeyRegistered   bool
	hotkeyID           uintptr
	currentHotkeyCombo string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{hotkeyID: 1}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.initSettings(); err != nil {
		a.logError(fmt.Sprintf("加载配置失败: %v", err))
	}
	if err := a.ensureService(); err != nil {
		a.emit(eventConfigMissingKey, map[string]string{"message": err.Error()})
	} else {
		a.emit(eventConfigReady, map[string]string{"message": "翻译服务已就绪"})
	}
	// 根据配置调整窗口状态
	a.applyWindowPreferences()
}

// StartScreenshotTranslation 触发一次截图翻译流程
func (a *App) StartScreenshotTranslation() error {
	if err := a.ensureService(); err != nil {
		a.emit(eventTranslationError, map[string]string{
			"stage":   "init",
			"message": err.Error(),
		})
		return err
	}

	a.screenshotLocker.Lock()
	if a.screenshotActive {
		a.screenshotLocker.Unlock()
		return fmt.Errorf("已有截图任务正在进行")
	}
	a.screenshotActive = true
	a.screenshotLocker.Unlock()

	go func() {
		a.emit(eventTranslationStarted, map[string]string{"source": "screenshot"})
		a.emit(eventTranslationProgress, map[string]string{
			"stage":   "prepare",
			"message": "请按下鼠标左键拖拽选择需要翻译的区域，按 Esc 取消",
		})
		a.screenshotMgr.StartOnce()
		a.screenshotLocker.Lock()
		a.screenshotActive = false
		a.screenshotLocker.Unlock()
		a.emit(eventTranslationIdle, nil)
	}()

	return nil
}

// TranslateText 翻译纯文本
func (a *App) TranslateText(input string) (*UITranslationResult, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("请输入要翻译的内容")
	}

	if err := a.ensureService(); err != nil {
		a.emit(eventTranslationError, map[string]string{
			"stage":   "init",
			"message": err.Error(),
		})
		return nil, err
	}

	a.emit(eventTranslationStarted, map[string]string{"source": "manual"})
	result, err := a.translationSvc.TranslateText(trimmed)
	if err != nil {
		a.emit(eventTranslationError, map[string]string{
			"stage":   "translate",
			"message": err.Error(),
		})
		return nil, err
	}

	uiResult := &UITranslationResult{
		OriginalText:   result.OriginalText,
		TranslatedText: result.TranslatedText,
		Source:         "manual",
		Timestamp:      time.Now(),
		DurationMs:     result.ProcessingTime.Milliseconds(),
	}

	a.emit(eventTranslationResult, uiResult)
	a.postProcessTranslation(uiResult.TranslatedText)
	return uiResult, nil
}

// GetSettings 返回当前配置
func (a *App) GetSettings() (*SettingsDTO, error) {
	if a.settingsManager == nil {
		if err := a.initSettings(); err != nil {
			return nil, err
		}
	}

	dto := fromConfigSettings(a.settings)
	return &dto, nil
}

// SaveSettings 更新配置
func (a *App) SaveSettings(payload SettingsDTO) (*SettingsDTO, error) {
	if a.settingsManager == nil {
		if err := a.initSettings(); err != nil {
			return nil, err
		}
	}

	settings := toConfigSettings(payload)
	if err := a.settingsManager.Save(settings); err != nil {
		return nil, err
	}

	fresh, err := a.settingsManager.Load()
	if err != nil {
		a.settings = settings
		a.logError(fmt.Sprintf("重新加载配置失败: %v", err))
	} else {
		a.settings = fresh
	}
	a.applyWindowPreferences()

	dto := fromConfigSettings(a.settings)
	a.emit(eventSettingsUpdated, dto)
	if theme := strings.TrimSpace(dto.Theme); theme != "" {
		a.emit(eventSettingsTheme, map[string]string{"theme": theme})
	}

	if err := a.ensureService(); err != nil {
		a.emit(eventConfigMissingKey, map[string]string{"message": err.Error()})
	} else {
		a.emit(eventConfigReady, map[string]string{"message": "翻译服务已更新"})
	}

	return &dto, nil
}

// UITranslationResult 用于前端展示
type UITranslationResult struct {
	OriginalText   string    `json:"originalText"`
	TranslatedText string    `json:"translatedText"`
	Source         string    `json:"source"`
	Timestamp      time.Time `json:"timestamp"`
	DurationMs     int64     `json:"durationMs"`
}

// SettingsDTO 前端-后端交互的配置载体
type SettingsDTO struct {
	APIKeyOverride      string `json:"apiKeyOverride"`
	TargetLanguage      string `json:"targetLanguage"`
	AutoCopyResult      bool   `json:"autoCopyResult"`
	KeepWindowOnTop     bool   `json:"keepWindowOnTop"`
	Theme               string `json:"theme"`
	ShowToastOnComplete bool   `json:"showToastOnComplete"`
	HotkeyCombination   string `json:"hotkeyCombination"`
}

func (a *App) initSettings() error {
	if a.settingsManager != nil {
		return nil
	}
	manager, err := config.NewSettingsManager("Translater")
	if err != nil {
		return err
	}
	a.settingsManager = manager
	settings, err := manager.Load()
	if err != nil {
		return err
	}
	a.settings = settings
	return nil
}

func (a *App) ensureService() error {
	if err := a.initSettings(); err != nil {
		return err
	}

	apiKey, err := a.resolveAPIKey()
	if err != nil {
		a.disableHotkey()
		return err
	}

	if a.translationSvc == nil || apiKey != a.currentAPIKey {
		a.translationSvc = service.NewTranslationService(ai.NewZhipuAIClient(apiKey))
		a.currentAPIKey = apiKey
	}

	if a.screenshotMgr == nil {
		a.screenshotMgr = screenshot.NewManager()
		a.screenshotMgr.SetCaptureHandler(a.handleScreenshotCapture)
	} else {
		// 确保 handler 持续引用最新的 service
		a.screenshotMgr.SetCaptureHandler(a.handleScreenshotCapture)
	}

	if err := a.ensureHotkeyListener(); err != nil {
		a.logError(fmt.Sprintf("热键初始化失败: %v", err))
	}

	return nil
}

func (a *App) ensureHotkeyListener() error {
	combo := strings.TrimSpace(a.settings.HotkeyCombination)
	if combo == "" {
		combo = config.DefaultSettings().HotkeyCombination
	}

	modifiers, key, err := hotkey.ParseCombination(combo)
	if err != nil {
		a.disableHotkey()
		return err
	}

	canonical := hotkey.FormatCombination(modifiers, key)

	a.hotkeyMutex.Lock()
	defer a.hotkeyMutex.Unlock()

	if a.hotkeyMgr == nil {
		a.hotkeyMgr = hotkey.NewManager()
		if a.hotkeyID == 0 {
			a.hotkeyID = 1
		}
	}

	if a.hotkeyRegistered {
		if strings.EqualFold(a.currentHotkeyCombo, canonical) {
			return nil
		}
		a.hotkeyMgr.Unregister(a.hotkeyID)
		a.hotkeyRegistered = false
	}

	if err := a.hotkeyMgr.Register(a.hotkeyID, modifiers, key, func() {
		go a.handleHotkeyTrigger()
	}); err != nil {
		return err
	}

	a.hotkeyRegistered = true
	a.currentHotkeyCombo = canonical
	a.hotkeyLoopOnce.Do(func() {
		go a.hotkeyMgr.Start()
	})

	return nil
}

func (a *App) disableHotkey() {
	a.hotkeyMutex.Lock()
	defer a.hotkeyMutex.Unlock()

	if a.hotkeyMgr != nil && a.hotkeyRegistered {
		a.hotkeyMgr.Unregister(a.hotkeyID)
		a.hotkeyRegistered = false
	}
	a.currentHotkeyCombo = ""
}

func (a *App) handleHotkeyTrigger() {
	a.screenshotLocker.Lock()
	if a.screenshotActive {
		a.screenshotLocker.Unlock()
		return
	}
	a.screenshotLocker.Unlock()

	if err := a.StartScreenshotTranslation(); err != nil {
		a.logError(fmt.Sprintf("热键触发截图失败: %v", err))
	}
}

func (a *App) handleScreenshotCapture(startX, startY, endX, endY int) bool {
	a.emit(eventTranslationProgress, map[string]string{
		"stage":   "ocr",
		"message": "正在识别文字…",
	})

	result, err := a.translationSvc.ProcessScreenshotDetailed(startX, startY, endX, endY)
	if err != nil {
		a.emit(eventTranslationError, map[string]string{
			"stage":   "screenshot",
			"message": err.Error(),
		})
		return false
	}

	uiResult := &UITranslationResult{
		OriginalText:   result.ExtractedText,
		TranslatedText: result.TranslatedText,
		Source:         "screenshot",
		Timestamp:      time.Now(),
		DurationMs:     result.ProcessingTime.Milliseconds(),
	}

	if strings.TrimSpace(result.ExtractedText) == "" {
		a.emit(eventTranslationProgress, map[string]string{
			"stage":   "ocr",
			"message": "未检测到文字内容",
		})
		a.emit(eventTranslationIdle, nil)
		return false
	}

	a.emit(eventTranslationProgress, map[string]string{
		"stage":   "translate",
		"message": "正在翻译…",
	})

	if strings.TrimSpace(result.TranslatedText) == "" {
		a.emit(eventTranslationError, map[string]string{
			"stage":   "translate",
			"message": "翻译结果为空",
		})
		return false
	}

	a.emit(eventTranslationResult, uiResult)
	a.postProcessTranslation(uiResult.TranslatedText)
	return true
}

func (a *App) postProcessTranslation(translated string) {
	if !a.settings.AutoCopyResult || strings.TrimSpace(translated) == "" {
		return
	}
	if a.ctx == nil {
		return
	}
	if err := runtime.ClipboardSetText(a.ctx, translated); err != nil {
		a.logError(fmt.Sprintf("复制翻译结果失败: %v", err))
		return
	}
	a.emit(eventTranslationCopied, map[string]string{"message": "翻译结果已复制到剪贴板"})
}

func (a *App) applyWindowPreferences() {
	if a.ctx == nil {
		return
	}
	runtime.WindowSetAlwaysOnTop(a.ctx, a.settings.KeepWindowOnTop)
}

func (a *App) resolveAPIKey() (string, error) {
	if key := strings.TrimSpace(a.settings.APIKeyOverride); key != "" {
		return key, nil
	}
	reader := config.NewFileAPIKeyReader([]string{".env", "env", "../.env", "../env"})
	return reader.ReadAPIKey()
}

func (a *App) emit(event string, payload interface{}) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, event, payload)
}

func (a *App) logError(message string) {
	if a.ctx == nil {
		return
	}
	runtime.LogError(a.ctx, message)
}

func fromConfigSettings(settings config.Settings) SettingsDTO {
	return SettingsDTO{
		APIKeyOverride:      settings.APIKeyOverride,
		TargetLanguage:      settings.TargetLanguage,
		AutoCopyResult:      settings.AutoCopyResult,
		KeepWindowOnTop:     settings.KeepWindowOnTop,
		Theme:               settings.Theme,
		ShowToastOnComplete: settings.ShowToastOnComplete,
		HotkeyCombination:   settings.HotkeyCombination,
	}
}

func toConfigSettings(dto SettingsDTO) config.Settings {
	settings := config.DefaultSettings()
	settings.APIKeyOverride = strings.TrimSpace(dto.APIKeyOverride)
	if strings.TrimSpace(dto.TargetLanguage) != "" {
		settings.TargetLanguage = dto.TargetLanguage
	}
	settings.AutoCopyResult = dto.AutoCopyResult
	settings.KeepWindowOnTop = dto.KeepWindowOnTop
	if strings.TrimSpace(dto.Theme) != "" {
		settings.Theme = dto.Theme
	}
	settings.ShowToastOnComplete = dto.ShowToastOnComplete
	combo := strings.TrimSpace(dto.HotkeyCombination)
	if combo == "" {
		settings.HotkeyCombination = config.DefaultSettings().HotkeyCombination
	} else {
		if normalized, err := hotkey.NormalizeCombination(combo); err == nil {
			settings.HotkeyCombination = normalized
		} else {
			settings.HotkeyCombination = config.DefaultSettings().HotkeyCombination
		}
	}
	return settings
}
