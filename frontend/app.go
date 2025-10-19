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
	streamMutex          sync.Mutex
	streamActive         bool
	streamSource         string
	streamRect           overlay.Rect
	streamHasRect        bool
	streamOverlayVisible bool
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

// TranslateText 翻译纯文本
func (a *App) TranslateText(input string) (*UITranslationResult, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, fmt.Errorf("请输入要翻译的内容")
	}

	streamEnabled := a.settings.EnableStreamOutput
	var streamSuccess bool
	if streamEnabled {
		a.beginStream("manual", nil)
		defer func() {
			a.endStream(!streamSuccess)
		}()
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
	streamSuccess = true
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
	APIKeyOverride       string `json:"apiKeyOverride"`
	AutoCopyResult       bool   `json:"autoCopyResult"`
	KeepWindowOnTop      bool   `json:"keepWindowOnTop"`
	Theme                string `json:"theme"`
	ShowToastOnComplete  bool   `json:"showToastOnComplete"`
	EnableStreamOutput   bool   `json:"enableStreamOutput"`
	HotkeyCombination    string `json:"hotkeyCombination"`
	ExtractPrompt        string `json:"extractPrompt"`
	TranslatePrompt      string `json:"translatePrompt"`
	APIBaseURL           string `json:"apiBaseUrl"`
	TranslateModel       string `json:"translateModel"`
	VisionModel          string `json:"visionModel"`
	VisionAPIBaseURL     string `json:"visionApiBaseUrl"`
	VisionAPIKeyOverride string `json:"visionApiKeyOverride"`
	UseVisionForTranslation bool `json:"useVisionForTranslation"`
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

	baseURL := ai.NormalizeBaseURL(a.settings.APIBaseURL)
	translateModel := strings.TrimSpace(a.settings.TranslateModel)
	if translateModel == "" {
		translateModel = ai.DefaultTranslateModel
	}
	visionModel := strings.TrimSpace(a.settings.VisionModel)
	if visionModel == "" {
		visionModel = ai.DefaultVisionModel
	}
	visionAPIKey := strings.TrimSpace(a.settings.VisionAPIKeyOverride)
	if visionAPIKey == "" {
		visionAPIKey = apiKey
	}
	visionBaseURL := strings.TrimSpace(a.settings.VisionAPIBaseURL)
	if visionBaseURL == "" {
		visionBaseURL = baseURL
	} else {
		visionBaseURL = ai.NormalizeBaseURL(visionBaseURL)
	}

	options := translation.Options{
		Stream:                a.settings.EnableStreamOutput,
		UseVisionForTranslation: a.settings.UseVisionForTranslation,
	}

	if a.translationSvc == nil || apiKey != a.currentAPIKey || baseURL != a.currentBaseURL || translateModel != a.currentTranslateModel || visionModel != a.currentVisionModel || visionAPIKey != a.currentVisionAPIKey || visionBaseURL != a.currentVisionBaseURL {
		a.translationSvc = translation.NewService(
			ai.NewClient(ai.ClientConfig{
				APIKey:         apiKey,
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
		a.currentAPIKey = apiKey
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
	var streamSuccess bool
	if streamEnabled {
		rect := a.computeOverlayRect(startX, startY, endX, endY)
		a.beginStream("screenshot", &rect)
		defer func() {
			a.endStream(!streamSuccess)
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
		if a.overlayMgr != nil {
			a.overlayMgr.Close()
		}
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

	if strings.TrimSpace(result.ExtractedText) == "" {
		a.emit(eventTranslationProgress, map[string]string{
			"stage":   "ocr",
			"message": "未检测到文字内容",
		})
		a.emit(eventTranslationIdle, nil)
		if a.overlayMgr != nil {
			a.overlayMgr.Close()
		}
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
		if a.overlayMgr != nil {
			a.overlayMgr.Close()
		}
		return false
	}

	a.emit(eventTranslationResult, uiResult)
	a.postProcessTranslation(uiResult.TranslatedText)
	if a.overlayMgr != nil && uiResult.Bounds != nil {
		rect := overlay.Rect{
			Left:   uiResult.Bounds.Left,
			Top:    uiResult.Bounds.Top,
			Width:  uiResult.Bounds.Width,
			Height: uiResult.Bounds.Height,
		}
		if streamEnabled && a.isStreamOverlayActive() {
			if err := a.overlayMgr.Update(uiResult.TranslatedText); err != nil {
				a.logError(fmt.Sprintf("更新流式翻译浮窗失败: %v", err))
				if err := a.overlayMgr.Show(uiResult.TranslatedText, rect); err != nil {
					a.logError(fmt.Sprintf("展示翻译浮窗失败: %v", err))
				}
			}
		} else {
			if err := a.overlayMgr.Show(uiResult.TranslatedText, rect); err != nil {
				a.logError(fmt.Sprintf("展示翻译浮窗失败: %v", err))
			}
		}
	}
	streamSuccess = true
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
	a.streamOverlayVisible = false
	a.streamMutex.Unlock()

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
	a.streamMutex.Unlock()

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
		APIKeyOverride:       settings.APIKeyOverride,
		AutoCopyResult:       settings.AutoCopyResult,
		KeepWindowOnTop:      settings.KeepWindowOnTop,
		Theme:                settings.Theme,
		ShowToastOnComplete:  settings.ShowToastOnComplete,
		EnableStreamOutput:   settings.EnableStreamOutput,
		HotkeyCombination:    settings.HotkeyCombination,
		ExtractPrompt:        settings.ExtractPrompt,
		TranslatePrompt:      settings.TranslatePrompt,
		APIBaseURL:           settings.APIBaseURL,
		TranslateModel:       settings.TranslateModel,
		VisionModel:          settings.VisionModel,
		VisionAPIBaseURL:     settings.VisionAPIBaseURL,
		VisionAPIKeyOverride: settings.VisionAPIKeyOverride,
		UseVisionForTranslation: settings.UseVisionForTranslation,
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
	return settings
}
