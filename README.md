# Translater 桌面翻译助手

Translater 是一个面向 Windows 平台的截图翻译工具：按下快捷键立即拉起截图框，后台自动完成 OCR、翻译并在原位置弹出半透明浮窗展示译文。默认提示词和配置已经内置，安装后只需填入兼容 OpenAI Chat Completions 的 API Key 即可使用。

> ⚠️ **使用限制**：本项目仅供学习和个人用途，禁止任何商业化使用。若在源代码基础上修改后发布，必须保留原作者标识与本说明。

## 功能特性

- 一键热键 (`Alt+T`) 截图，自动 OCR + 翻译。
- 翻译结果浮窗自动适配字体、支持暗色半透明背景。
- 支持自定义文字提取与翻译阶段的提示词，默认翻译提示词强制仅处理 `words` 字段，避免误译背景信息。
- 翻译与视觉模型/接口可分别配置，适配不同服务商或自建网关（兼容 OpenAI Chat Completions 协议）。
- 结果可自动复制到剪贴板，可选窗口置顶、完成 Toast 提醒。
- Wails + Vue 前端提供托盘和设置面板，配置项会持久化到用户目录。

## 架构总览

| 模块 | 位置 | 说明 |
| ---- | ---- | ---- |
| 应用入口 | `main.go` | 启动热键循环，初始化截图、AI、翻译服务。|
| AI 客户端 | `core/ai` | OpenAI 兼容客户端，可分别配置翻译/视觉模型与 Base URL。|
| 提示词 | `core/prompts` | 默认提取/翻译提示词常量，可被运行时覆盖。|
| 截图 | `core/screenshot` | 基于 `github.com/kbinani/screenshot` 进行屏幕捕获。|
| 翻译服务 | `core/translation` | 串联截图 → OCR → 翻译，并生成 UI 需要的数据。|
| 热键 | `core/hotkey` | Win32 阻塞消息循环，注册系统级热键。|
| Overlay UI | `core/ui/overlay` | Win32 窗口：绘制翻译结果、动态调节字体。|
| 桌面应用 | `frontend/app.go` | Wails 后端，与 Vue 设置面板通信。|
| 前端 UI | `frontend/frontend/` | Vue 3 + Tailwind 风格的设置页面与托盘。|

运行流程：

1. 程序启动后注册热键 `Alt+T`，并监听系统消息循环。
2. 用户触发热键后调用 `core/screenshot.Manager` 截取选区，OCR 提取文字。
3. `core/translation.Service` 使用默认提示词组装请求，通过 OpenAI 兼容接口调用翻译/视觉模型获取结果。
4. 结果保存到用户设置、复制到剪贴板，并通过 Win32 overlay 在屏幕上展示。
5. 前端 settings 面板通过 Wails RPC 读取/写入 `config.Settings`（`%AppData%/Translater/settings.json`）。

## 环境要求

- Windows 10 或更高版本（依赖 Win32 截图与热键 API）。
- Go 1.24+
- Node.js 18+ 与 npm（用于前端构建）。
- Wails CLI：`go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- 兼容 OpenAI Chat Completions 的 API Key（默认示例使用智谱开放平台）。
- Windows WebView2 Runtime（Windows 11 默认自带，Windows 10 可从微软下载）。

## 快速开始

1. **安装依赖**
   ```bash
   # 安装前端依赖
   cd frontend/frontend
   npm install
   cd ../../
   ```

2. **配置 API Key**
   - 在仓库根目录创建 `.env` 文件，写入：
     ```env
     API-KEY=sk-xxxxxxxxxxxxxxxx
     ```
     或者运行应用后，在设置面板填写「API Key」，程序会把密钥保存在 `%AppData%/Translater/settings.json`。
  - 可以直接在 [智谱开放平台](https://open.bigmodel.cn/) 创建密钥，或使用其他 OpenAI 兼容服务商，确保模型名称与 Base URL 在设置面板中正确配置。

3. **运行开发版本**
   - 纯 CLI 热键版本：`go run .`
   - 带桌面 UI：
     ```bash
     cd frontend
     wails dev
     ```
     默认会启动托盘 + 设置窗口，并带动 Go 后端。

4. **触发翻译**
   - 按 `Alt+T` 选择屏幕区域。
   - 等待浮窗显示译文，默认同时复制到剪贴板。

## 构建与发布

- 运行单元测试：`go test ./...`
- 静态检查：`go vet ./...`
- 构建桌面应用：
  ```bash
  cd frontend
  wails build -clean
  ```
  输出的可执行文件位于 `build/bin/`。
- 可选参数：
  - `-nsis` 生成 NSIS 安装包（需安装 NSIS）。
  - `-upx` 使用 UPX 压缩可执行文件（需安装 UPX）。


## 设置与提示词

- 支持以下开关：自动复制、窗口置顶、完成提醒、快捷键组合。
- 翻译/视觉的 API Key 与 Base URL 可分别配置，空值时自动回退到翻译接口设置。
- 「文字提取提示词」「翻译提示词」文本框默认不显示内置提示词；当文本框留空时，程序会回退到 `core/prompts/prompts.go` 中的默认值（翻译默认提示词已内置“只翻译 `words` 字段”的强制要求）。
- 「恢复默认提示词」「恢复默认接口与模型」按钮可快速回到官方默认配置。

## 配置存储

用户设置文件路径：`user/%AppData%/roaming/Translater/settings.json`（Windows）。

字段说明（部分）：
- `apiKeyOverride` / `visionApiKeyOverride`：界面保存的翻译/视觉 API Key。
- `apiBaseUrl` / `visionApiBaseUrl`：翻译/视觉接口 Base URL（OpenAI 兼容），空值时回退。
- `translateModel` / `visionModel`：翻译与视觉模型名称。
- `extractPrompt` / `translatePrompt`：自定义提示词，空值回退到默认文案。
- 其余字段对应面板的勾选项与主题。

删除该文件即可恢复所有默认设置（含热键、提示词等）。

## 排错指南

| 现象 | 可能原因 | 解决方案 |
| ---- | -------- | -------- |
| 启动提示找不到 API Key | `.env` / `env/` 缺失，或设置面板未保存密钥 | 按上文配置 `.env` / 在设置中输入 API Key |
| 截图后未显示浮窗 | Win32 Overlay 初始化失败 | 查看日志，确保未被安全软件拦截，或关闭「窗口置顶」重新测试 |
| 热键冲突 | 系统已有同快捷键 | 在设置面板修改热键组合 |
| 打包后包含 API Key | `settings.json` 或 `.env` 被打包 | 发布前清理本地配置，勿把敏感文件放入安装包 |

## 贡献指南

1. Fork & clone 仓库。
2. 安装依赖并运行 `go fmt ./...`、`npm run lint`（若有）。
3. 提交前确保 `go test ./...`、`wails build` 均通过。
4. 提交信息遵循项目约定的 Conventional Commits，必要时附带中文摘要。

## 致谢与版权

- 原作者：HandyWote。
- 项目使用的第三方库详见 `go.mod`、`package.json`。
- 任何二次分发请保留此 README 中的原作者信息与非商业声明。
