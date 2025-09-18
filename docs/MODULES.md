# 各模块详细设计

## 1. 全局快捷键监听模块 (hotkey)

### 功能描述
负责注册和监听全局快捷键，当用户按下预设快捷键时触发翻译流程。

### 接口设计
```go
// Hotkey represents a global hotkey
type Hotkey struct {
    ID        int
    Modifiers uint32
    Key       uint32
}

// HotkeyManager manages global hotkeys
type HotkeyManager struct {
    hotkeys map[int]*Hotkey
    hwnd    syscall.Handle
    callback func(int)
}

// NewHotkeyManager creates a new hotkey manager
func NewHotkeyManager() *HotkeyManager

// Register registers a global hotkey
func (hm *HotkeyManager) Register(id int, modifiers uint32, key uint32) error

// Unregister unregisters a global hotkey
func (hm *HotkeyManager) Unregister(id int) error

// Start starts the hotkey message loop
func (hm *HotkeyManager) Start(callback func(int)) error
```

### 实现细节
- 使用Windows API `RegisterHotKey`和`UnregisterHotKey`注册和注销热键
- 创建隐藏窗口接收WM_HOTKEY消息
- 通过消息循环处理热键事件

## 2. 屏幕区域选择模块 (selection)

### 功能描述
提供屏幕区域选择功能，用户可以通过拖拽选择需要翻译的区域。

### 接口设计
```go
// SelectionHandler handles screen region selection
type SelectionHandler struct {
    // 配置和状态
}

// NewSelectionHandler creates a new selection handler
func NewSelectionHandler() *SelectionHandler

// StartSelection starts the region selection process
func (sh *SelectionHandler) StartSelection() (x, y, width, height int, err error)

// CaptureRegion captures the specified screen region
func (sh *SelectionHandler) CaptureRegion(x, y, width, height int) ([]byte, error)
```

### 实现细节
- 创建全屏透明覆盖窗口
- 监听鼠标事件实现区域选择
- 使用Windows API进行屏幕截图

## 3. 屏幕捕获服务模块 (capture)

### 功能描述
实现屏幕截图功能，支持全屏和区域截图。

### 接口设计
```go
// ScreenCapture handles screen capturing
type ScreenCapture struct {
    // 配置和状态
}

// NewScreenCapture creates a new screen capture service
func NewScreenCapture() *ScreenCapture

// CaptureScreen captures the entire screen
func (sc *ScreenCapture) CaptureScreen() ([]byte, error)

// CaptureRegion captures a specific region of the screen
func (sc *ScreenCapture) CaptureRegion(x, y, width, height int) ([]byte, error)
```

### 实现细节
- 使用Windows GDI或DirectX进行屏幕截图
- 支持多种图像格式输出（PNG, JPEG等）
- 优化性能，只捕获变化区域

## 4. OCR识别服务模块 (ocr)

### 功能描述
集成OCR引擎进行文字识别，处理图像并提取文字内容。

### 接口设计
```go
// OCRService handles OCR recognition
type OCRService struct {
    client *gosseract.Client
}

// NewOCRService creates a new OCR service
func NewOCRService() *OCRService

// RecognizeText recognizes text from image data
func (os *OCRService) RecognizeText(imgData []byte) (string, error)

// RecognizeTextWithPosition recognizes text with position information
func (os *OCRService) RecognizeTextWithPosition(imgData []byte) ([]TextBlock, error)
```

### 实现细节
- 集成Tesseract OCR引擎
- 支持多种语言识别
- 提供文字位置信息
- 图像预处理优化识别准确率

## 5. 翻译处理服务模块 (translate)

### 功能描述
使用ai技术实现智能翻译，语义翻译

### 实现细节
- 支持openai格式的api
- 实现翻译结果缓存
- 处理API调用限制和错误重试
- 支持批量翻译

## 6. 浮动窗口显示模块 (ui)

### 功能描述
显示翻译结果的透明浮动窗口，支持自定义样式和位置。

### 接口设计
```go
// OverlayWindow represents a floating overlay window
type OverlayWindow struct {
    hwnd syscall.Handle
    // 窗口属性
}

// NewOverlayWindow creates a new overlay window
func NewOverlayWindow() *OverlayWindow

// ShowTranslation shows the translation result
func (ow *OverlayWindow) ShowTranslation(text string, x, y int) error

// Hide hides the overlay window
func (ow *OverlayWindow) Hide() error

// SetPosition sets the window position
func (ow *OverlayWindow) SetPosition(x, y int) error

// SetSize sets the window size
func (ow *OverlayWindow) SetSize(width, height int) error
```

### 实现细节
- 创建透明可穿透的浮动窗口
- 支持自定义样式（字体、颜色、透明度等）
- 实现自动换行和滚动
- 支持鼠标穿透和置顶显示

## 7. 系统托盘模块 (tray)

### 功能描述
程序后台运行时的系统托盘图标，提供程序控制菜单。

### 接口设计
```go
// TrayIcon represents a system tray icon
type TrayIcon struct {
    // 托盘图标属性
}

// NewTrayIcon creates a new tray icon
func NewTrayIcon() *TrayIcon

// Show shows the tray icon
func (ti *TrayIcon) Show() error

// Hide hides the tray icon
func (ti *TrayIcon) Hide() error

// SetMenu sets the context menu
func (ti *TrayIcon) SetMenu(menu *Menu) error
```

### 实现细节
- 使用Windows API创建系统托盘图标
- 实现右键菜单功能
- 支持图标动画和提示信息

## 8. 配置管理模块 (config)

### 功能描述
管理用户配置和偏好设置，保存和加载配置信息。

### 接口设计
```go
// Config represents application configuration
type Config struct {
    Hotkey          HotkeyConfig
    OCR             OCRConfig
    Translation     TranslationConfig
    UI              UIConfig
    General         GeneralConfig
}

// HotkeyConfig represents hotkey configuration
type HotkeyConfig struct {
    Modifiers uint32
    Key       uint32
}

// OCRConfig represents OCR configuration
type OCRConfig struct {
    Language string
    Engine   string
}

// TranslationConfig represents translation configuration
type TranslationConfig struct {
    SourceLanguage string
    TargetLanguage string
    Service        string
    APIKeys        map[string]string
}

// ConfigManager manages application configuration
type ConfigManager struct {
    config *Config
    path   string
}

// NewConfigManager creates a new config manager
func NewConfigManager(configPath string) *ConfigManager

// Load loads configuration from file
func (cm *ConfigManager) Load() error

// Save saves configuration to file
func (cm *ConfigManager) Save() error

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config

// SetConfig sets the configuration
func (cm *ConfigManager) SetConfig(config *Config) error
```

### 实现细节
- 使用JSON格式存储配置
- 支持配置热重载
- 提供默认配置
- 实现配置验证