package prompts

// 默认提示词常量，供配置和服务在用户未自定义时使用。
const (
	DefaultExtractPrompt   = "请提取这张图片中的所有文字内容，只返回文字，不要添加任何其他说明。"
	DefaultTranslatePrompt = "请将以下文本翻译成中文，保持原文的格式和结构："
)

// 可覆盖的提示词，允许运行时根据配置动态调整。
var (
	ExtractPrompt   = DefaultExtractPrompt
	TranslatePrompt = DefaultTranslatePrompt
)
