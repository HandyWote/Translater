package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"Translater/core/ai"
	"Translater/core/config"
	"Translater/core/hotkey"
	"Translater/core/screenshot"
	"Translater/core/translation"
	"Translater/core/ui/overlay"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	eventTranslationStarted  = "translation:started"
	eventTranslationProgress = "translation:progress"
	eventTranslationResult   = "translation:result"
	eventTranslationDelta    = "translation:delta"
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

	translationSvc        translation.Service
	screenshotMgr         *screenshot.Manager
	currentAPIKey         string
	currentBaseURL        string
	currentTranslateModel string
	currentVisionModel    string
	currentVisionAPIKey   string
	currentVisionBaseURL  string
	streamMutex           sync.Mutex
	streamActive          bool
	streamSource          string
	streamRect            overlay.Rect
	streamHasRect         bool
	streamOverlayVisible  bool
	screenshotLocker      sync.Mutex
	screenshotActive      bool
	screenshotDone        chan struct{}
	hotkeyMgr             *hotkey.Manager
	hotkeyMutex           sync.Mutex
	hotkeyLoopOnce        sync.Once
	hotkeyRegistered      bool
	hotkeyID              uintptr
	currentHotkeyCombo    string
	overlayMgr            *overlay.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		hotkeyID:   1,
		overlayMgr: overlay.NewManager(),
	}
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
	a.initSystemTray()
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

	for {
		a.screenshotLocker.Lock()
		if !a.screenshotActive {
			done := make(chan struct{})
			a.screenshotActive = true
			a.screenshotDone = done
			a.screenshotLocker.Unlock()

			if a.overlayMgr != nil {
				a.overlayMgr.Close()
			}

			go a.runScreenshotCapture(done)
			return nil
		}
		a.screenshotLocker.Unlock()

		if a.screenshotMgr == nil {
			return fmt.Errorf("截图服务未初始化")
		}
		a.screenshotMgr.CancelActiveCapture()
	}
}

func (a *App) runScreenshotCapture(done chan struct{}) {
	defer func() {
		a.screenshotLocker.Lock()
		if a.screenshotDone == done {
			a.screenshotActive = false
			a.screenshotDone = nil
		}
		a.screenshotLocker.Unlock()
		close(done)
		a.emit(eventTranslationIdle, nil)
	}()

	a.emit(eventTranslationStarted, map[string]string{"source": "screenshot"})
	a.emit(eventTranslationProgress, map[string]string{
		"stage":   "prepare",
		"message": "请按下鼠标左键拖拽选择需要翻译的区域，按 Esc 取消",
	})

	if a.screenshotMgr != nil {
		a.screenshotMgr.StartOnce()
	}
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
	OriginalText   string              `json:"originalText"`
	TranslatedText string              `json:"translatedText"`
	Source         string              `json:"source"`
	Timestamp      time.Time           `json:"timestamp"`
	DurationMs     int64               `json:"durationMs"`
	Bounds         *UIScreenshotBounds `json:"bounds,omitempty"`
}

// UIScreenshotBounds 将截图范围暴露给前端用于定位浮窗
type UIScreenshotBounds struct {
	StartX int `json:"startX"`
	StartY int `json:"startY"`
	EndX   int `json:"endX"`
	EndY   int `json:"endY"`
	Left   int `json:"left"`
	Top    int `json:"top"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SettingsDTO 前端-后端交互的配置载体
type SettingsDTO struct {
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
	SourceLanguage          string `json:"sourceLanguage"`
	TargetLanguage          string `json:"targetLanguage"`
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

	// 使用新的 API Key 解析逻辑
	mainKey, translateKey, err := a.resolveAPIKeys()
	if err != nil {
		a.disableHotkey()
		return err
	}

	baseURL := ai.NormalizeBaseURL(a.settings.APIBaseURL)
	translateModel := strings.TrimSpace(a.settings.TranslateModel)
	if translateModel == "" {
		translateModel = ai.DefaultTranslateModel
	}
	visionModel := strings.TrimSpace(a.settings.VisionModel)
	if visionModel == "" {
		visionModel = ai.DefaultVisionModel
	}

	// 视觉 API 配置（使用主 key）
	visionAPIKey := mainKey
	visionBaseURL := strings.TrimSpace(a.settings.VisionAPIBaseURL)
	if visionBaseURL == "" {
		visionBaseURL = baseURL
	} else {
		visionBaseURL = ai.NormalizeBaseURL(visionBaseURL)
	}

	options := translation.Options{
		Stream:                  a.settings.EnableStreamOutput,
		UseVisionForTranslation: a.settings.UseVisionForTranslation,
		SourceLanguage:          a.settings.SourceLanguage,
		TargetLanguage:          a.settings.TargetLanguage,
	}

	if a.translationSvc == nil || translateKey != a.currentAPIKey || baseURL != a.currentBaseURL || translateModel != a.currentTranslateModel || visionModel != a.currentVisionModel || visionAPIKey != a.currentVisionAPIKey || visionBaseURL != a.currentVisionBaseURL {
		a.translationSvc = translation.NewService(
			ai.NewClient(ai.ClientConfig{
				APIKey:         translateKey,
				BaseURL:        baseURL,
				TranslateModel: translateModel,
				VisionModel:    visionModel,
				VisionAPIKey:   visionAPIKey,
				VisionBaseURL:  visionBaseURL,
			}),
			a.settings.ExtractPrompt,
			a.settings.TranslatePrompt,
			options,
		)
		a.currentAPIKey = translateKey
		a.currentBaseURL = baseURL
		a.currentTranslateModel = translateModel
		a.currentVisionModel = visionModel
		a.currentVisionAPIKey = visionAPIKey
		a.currentVisionBaseURL = visionBaseURL
	}

	if a.translationSvc != nil {
		a.translationSvc.UpdatePrompts(a.settings.ExtractPrompt, a.settings.TranslatePrompt)
		a.translationSvc.UpdateOptions(options)
		a.translationSvc.SetStreamHandler(a.handleStreamDelta)
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
	if err := a.StartScreenshotTranslation(); err != nil {
		a.logError(fmt.Sprintf("热键触发截图失败: %v", err))
	}
}

func (a *App) computeOverlayRect(startX, startY, endX, endY int) overlay.Rect {
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
	return overlay.Rect{
		Left:   left,
		Top:    top,
		Width:  width,
		Height: height,
	}
}

func (a *App) handleScreenshotCapture(startX, startY, endX, endY int) bool {
	streamEnabled := a.settings.EnableStreamOutput
	shouldCleanup := false
	if streamEnabled {
		rect := a.computeOverlayRect(startX, startY, endX, endY)
		a.beginStream("screenshot", &rect)
		shouldCleanup = true
		// 翻译完成后只清理流状态，不关闭overlay窗口
		// overlay窗口由用户按ESC键手动关闭
		defer func() {
			if shouldCleanup {
				a.endStream(false)
			}
		}()
	}

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
		// 不自动关闭overlay，让用户可以看到错误信息并手动关闭
		return false
	}


	uiResult := &UITranslationResult{
		OriginalText:   result.ExtractedText,
		TranslatedText: result.TranslatedText,
		Source:         "screenshot",
		Timestamp:      time.Now(),
		DurationMs:     result.ProcessingTime.Milliseconds(),
		Bounds: &UIScreenshotBounds{
			StartX: result.Bounds.StartX,
			StartY: result.Bounds.StartY,
			EndX:   result.Bounds.EndX,
			EndY:   result.Bounds.EndY,
			Left:   result.Bounds.Left,
			Top:    result.Bounds.Top,
			Width:  result.Bounds.Width,
			Height: result.Bounds.Height,
		},
	}

	// 检查是否有有效内容（ExtractedText 或 TranslatedText 至少有一个非空）
	// 在 useVisionForTranslation 模式下，ExtractedText 可能为空但 TranslatedText 有内容
	hasExtractedText := strings.TrimSpace(result.ExtractedText) != ""
	hasTranslatedText := strings.TrimSpace(result.TranslatedText) != ""

	if !hasExtractedText && !hasTranslatedText {
		a.logError("⚠️ [后端] OCR 和翻译结果都为空")
		a.emit(eventTranslationProgress, map[string]string{
			"stage":   "ocr",
			"message": "未检测到文字内容",
		})
		a.emit(eventTranslationIdle, nil)
		// 不自动关闭overlay，让用户手动关闭
		return false
	}

	if !hasTranslatedText {
		a.logError("⚠️ [后端] 翻译结果为空（但有提取的文本），发送 translation:error 事件")
		a.emit(eventTranslationError, map[string]string{
			"stage":   "translate",
			"message": "翻译结果为空",
		})
		// 不自动关闭overlay，让用户手动关闭
		return false
	}

	a.emit(eventTranslationProgress, map[string]string{
		"stage":   "translate",
		"message": "翻译完成",
	})

	if streamEnabled {
		a.endStream(false)
		shouldCleanup = false
	}

	// 调试日志：打印即将发送的结果
	preview := uiResult.TranslatedText
	if len(preview) > 100 {
		preview = preview[:100]
	}
	a.logError(fmt.Sprintf("🚀 [后端] 准备发送 translation:result, translatedText 长度: %d, 内容: %s",
		len(uiResult.TranslatedText), preview))

	a.emit(eventTranslationResult, uiResult)
	a.postProcessTranslation(uiResult.TranslatedText)

	// 显示或更新overlay窗口，窗口会一直保持显示直到用户按ESC关闭
	if a.overlayMgr != nil && uiResult.Bounds != nil {
		rect := overlay.Rect{
			Left:   uiResult.Bounds.Left,
			Top:    uiResult.Bounds.Top,
			Width:  uiResult.Bounds.Width,
			Height: uiResult.Bounds.Height,
		}
		if streamEnabled && a.isStreamOverlayActive() {
			// 流式模式下，更新已存在的overlay
			if err := a.overlayMgr.Update(uiResult.TranslatedText); err != nil {
				a.logError(fmt.Sprintf("更新流式翻译浮窗失败: %v", err))
				// 如果更新失败，尝试重新显示
				if err := a.overlayMgr.Show(uiResult.TranslatedText, rect); err != nil {
					a.logError(fmt.Sprintf("展示翻译浮窗失败: %v", err))
				}
			}
		} else {
			// 非流式模式或overlay未激活，直接显示新窗口
			if err := a.overlayMgr.Show(uiResult.TranslatedText, rect); err != nil {
				a.logError(fmt.Sprintf("展示翻译浮窗失败: %v", err))
			}
		}
	}

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

func (a *App) beginStream(source string, rect *overlay.Rect) {
	a.streamMutex.Lock()
	defer a.streamMutex.Unlock()

	a.streamActive = true
	a.streamSource = source
	if rect != nil {
		a.streamRect = *rect
		a.streamHasRect = true
	} else {
		a.streamRect = overlay.Rect{}
		a.streamHasRect = false
	}
	a.streamOverlayVisible = false
}

func (a *App) endStream(closeOverlay bool) {
	a.streamMutex.Lock()
	wasActive := a.streamActive
	a.streamActive = false
	a.streamSource = ""
	a.streamHasRect = false
	// 只有在需要关闭overlay时才重置可见状态
	if closeOverlay {
		a.streamOverlayVisible = false
	}
	a.streamMutex.Unlock()

	// 如果需要关闭overlay且之前有活动的流，则关闭窗口
	if closeOverlay && wasActive && a.overlayMgr != nil {
		a.overlayMgr.Close()
	}
}

func (a *App) handleStreamDelta(stage string, content string) {
	payload := map[string]string{
		"stage":   stage,
		"content": content,
	}

	a.streamMutex.Lock()
	source := a.streamSource
	hasRect := a.streamHasRect
	rect := a.streamRect
	overlayVisible := a.streamOverlayVisible
	active := a.streamActive
	a.streamMutex.Unlock()

	if !active {
		return
	}

	if source != "" {
		payload["source"] = source
	}
	a.emit(eventTranslationDelta, payload)

	if source != "screenshot" || !hasRect || a.overlayMgr == nil {
		return
	}

	if strings.TrimSpace(content) == "" {
		return
	}

	if !overlayVisible {
		if err := a.overlayMgr.Show(content, rect); err != nil {
			a.logError(fmt.Sprintf("展示流式翻译浮窗失败: %v", err))
			return
		}
		a.streamMutex.Lock()
		a.streamOverlayVisible = true
		a.streamMutex.Unlock()
		return
	}

	if err := a.overlayMgr.Update(content); err != nil {
		a.logError(fmt.Sprintf("更新流式翻译浮窗失败: %v", err))
		if err := a.overlayMgr.Show(content, rect); err != nil {
			a.logError(fmt.Sprintf("展示流式翻译浮窗失败: %v", err))
			return
		}
		a.streamMutex.Lock()
		a.streamOverlayVisible = true
		a.streamMutex.Unlock()
	}
}

func (a *App) isStreamOverlayActive() bool {
	a.streamMutex.Lock()
	defer a.streamMutex.Unlock()
	return a.streamOverlayVisible
}

func (a *App) applyWindowPreferences() {
	if a.ctx == nil {
		return
	}
	runtime.WindowSetAlwaysOnTop(a.ctx, a.settings.KeepWindowOnTop)
}

func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
}

func (a *App) quitApplication() {
	a.teardownSystemTray()
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.teardownSystemTray()
	if a.overlayMgr != nil {
		a.overlayMgr.Close()
	}
}

// resolveAPIKeys 根据 useVisionForTranslation 设置解析主 API Key 和翻译 API Key
// 主 API Key 优先从 visionApiKeyOverride 读取（向前兼容），翻译 API Key 根据模式决定
func (a *App) resolveAPIKeys() (mainKey string, translateKey string, err error) {
	// 1. 解析主 API Key（视觉 API Key 优先）
	mainKey = strings.TrimSpace(a.settings.VisionAPIKeyOverride)
	if mainKey == "" {
		// 向后兼容：回退到 apiKeyOverride
		mainKey = strings.TrimSpace(a.settings.APIKeyOverride)
	}
	if mainKey == "" {
		// 最后尝试从文件读取
		reader := config.NewFileAPIKeyReader([]string{".env", "env", "../.env", "../env"})
		mainKey, err = reader.ReadAPIKey()
		if err != nil || mainKey == "" {
			return "", "", fmt.Errorf("需要配置视觉 API Key (visionApiKeyOverride) 或在 .env 文件中设置")
		}
	}

	// 2. 解析翻译 API Key
	if a.settings.UseVisionForTranslation {
		// 视觉直出模式：翻译也用主 key
		translateKey = mainKey
	} else {
		// 文本模型模式：翻译 key 可选，留空则回退到主 key
		translateKey = strings.TrimSpace(a.settings.APIKeyOverride)
		if translateKey == "" {
			translateKey = mainKey
		}
	}

	return mainKey, translateKey, nil
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
		APIKeyOverride:          settings.APIKeyOverride,
		AutoCopyResult:          settings.AutoCopyResult,
		KeepWindowOnTop:         settings.KeepWindowOnTop,
		Theme:                   settings.Theme,
		ShowToastOnComplete:     settings.ShowToastOnComplete,
		EnableStreamOutput:      settings.EnableStreamOutput,
		HotkeyCombination:       settings.HotkeyCombination,
		ExtractPrompt:           settings.ExtractPrompt,
		TranslatePrompt:         settings.TranslatePrompt,
		APIBaseURL:              settings.APIBaseURL,
		TranslateModel:          settings.TranslateModel,
		VisionModel:             settings.VisionModel,
		VisionAPIBaseURL:        settings.VisionAPIBaseURL,
		VisionAPIKeyOverride:    settings.VisionAPIKeyOverride,
		UseVisionForTranslation: settings.UseVisionForTranslation,
		SourceLanguage:          settings.SourceLanguage,
		TargetLanguage:          settings.TargetLanguage,
	}
}

func toConfigSettings(dto SettingsDTO) config.Settings {
	settings := config.DefaultSettings()
	settings.APIKeyOverride = strings.TrimSpace(dto.APIKeyOverride)
	settings.AutoCopyResult = dto.AutoCopyResult
	settings.KeepWindowOnTop = dto.KeepWindowOnTop
	if strings.TrimSpace(dto.Theme) != "" {
		settings.Theme = dto.Theme
	}
	settings.ShowToastOnComplete = dto.ShowToastOnComplete
	settings.EnableStreamOutput = dto.EnableStreamOutput
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
	settings.ExtractPrompt = strings.TrimSpace(dto.ExtractPrompt)
	settings.TranslatePrompt = strings.TrimSpace(dto.TranslatePrompt)
	settings.APIBaseURL = strings.TrimSpace(dto.APIBaseURL)
	settings.TranslateModel = strings.TrimSpace(dto.TranslateModel)
	settings.VisionModel = strings.TrimSpace(dto.VisionModel)
	settings.VisionAPIBaseURL = strings.TrimSpace(dto.VisionAPIBaseURL)
	settings.VisionAPIKeyOverride = strings.TrimSpace(dto.VisionAPIKeyOverride)
	settings.UseVisionForTranslation = dto.UseVisionForTranslation
	settings.SourceLanguage = strings.TrimSpace(dto.SourceLanguage)
	settings.TargetLanguage = strings.TrimSpace(dto.TargetLanguage)
	return settings
}
