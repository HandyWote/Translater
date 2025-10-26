package prompts

import (
	"fmt"
	"strings"
)

// 默认提示词常量，供配置和服务在用户未自定义时使用
const (
	DefaultExtractPrompt = `你是一个专业的视觉上下文分析专家，负责为高质量的翻译任务准备完整素材。请完成以下工作：

1. 背景描述：详细说明图像中出现的场景、主体、布局、风格以及任何可能影响理解的视觉线索。
2. 原文提取：逐项提取图像中的全部文字内容，保持{{.SourceLanguage}}原文的顺序与格式（包含换行、缩进、符号和大小写）。

输出要求：
- 将结果严格按照 JSON 结构输出，不要添加任何额外说明：
{
  "background": "...",
  "words": "..."
}
- "words" 字段必须只包含识别到的原文内容。

{{.RelayInstruction}}
{{.VisionDirectInstruction}}`

	DefaultTranslatePrompt = `你是一个专业的翻译 AI，专门处理图像文本在特定语境下的翻译任务。你将收到一个 JSON 对象：
- "background" 字段提供场景参考；
- "words" 字段包含需要翻译的原始文本（语种：{{.SourceLanguage}}）。

请将 "words" 字段精准翻译为 {{.TargetLanguage}}，并保持原有的段落、换行与符号。遵循以下原则：
1. 仅翻译 "words" 字段，忽略 "background" 字段内容；
2. 依据 "background" 提供的语境选择合适的术语与表达；
3. 保持专有名词、数字与排版一致；
4. 输出中不得包含额外的说明或注释。

{{.VisionModeInstruction}}`
)

// PromptVariables 用于动态替换提示词中的变量
type PromptVariables struct {
	SourceLanguage          string
	TargetLanguage          string
	UseVisionForTranslation bool
}

// ProcessExtractPrompt 处理提取提示词，根据运行模式追加指令
func ProcessExtractPrompt(basePrompt string, vars PromptVariables) string {
	prompt := replaceLanguagePlaceholders(basePrompt, vars)

	relay := buildRelayInstruction(vars)
	direct := buildVisionDirectInstruction(vars)

	prompt = strings.ReplaceAll(prompt, "{{.RelayInstruction}}", relay)
	prompt = strings.ReplaceAll(prompt, "{{.VisionDirectInstruction}}", direct)

	return strings.TrimSpace(prompt)
}

// ProcessTranslatePrompt 处理翻译提示词，替换动态变量
func ProcessTranslatePrompt(basePrompt string, vars PromptVariables) string {
	prompt := replaceLanguagePlaceholders(basePrompt, vars)

	instruction := buildVisionModeInstruction(vars)
	prompt = strings.ReplaceAll(prompt, "{{.VisionModeInstruction}}", instruction)

	return strings.TrimSpace(prompt)
}

func replaceLanguagePlaceholders(prompt string, vars PromptVariables) string {
	source := getLanguageDisplayName(vars.SourceLanguage)
	target := getLanguageDisplayName(vars.TargetLanguage)

	prompt = strings.ReplaceAll(prompt, "{{.SourceLanguage}}", source)
	prompt = strings.ReplaceAll(prompt, "{{.TargetLanguage}}", target)

	return prompt
}

func buildRelayInstruction(vars PromptVariables) string {
	if vars.UseVisionForTranslation {
		return ""
	}
	target := getLanguageDisplayName(vars.TargetLanguage)
	return "当前未启用视觉直出模式，请确保只返回原始文字 JSON，后续翻译流程会将其转换为" + target + "。"
}

func buildVisionDirectInstruction(vars PromptVariables) string {
	if !vars.UseVisionForTranslation {
		return ""
	}
	target := getLanguageDisplayName(vars.TargetLanguage)
	return "已启用视觉直出模式：完成 JSON 输出后，直接给出按原始版式排布的" + target + "翻译结果，不必再返回原文。"
}

func buildVisionModeInstruction(vars PromptVariables) string {
	target := getLanguageDisplayName(vars.TargetLanguage)
	if vars.UseVisionForTranslation {
		return "视觉直出模式开启：若输入仍包含原文，请直接输出对应的" + target + "译文，并保持与原文一致的排版。"
	}
	return "输入源自 OCR 流程，请只输出翻译后的" + target + "文本，不要重复或拼接原文。"
}

// getLanguageDisplayName 获取语言的显示名称
func getLanguageDisplayName(langCode string) string {
	languageMap := map[string]string{
		"auto":  "自动检测",
		"zh-CN": "中文",
		"zh-TW": "繁体中文",
		"en":    "英文",
		"ja":    "日文",
		"ko":    "韩文",
		"fr":    "法文",
		"de":    "德文",
		"es":    "西班牙文",
		"ru":    "俄文",
		"ar":    "阿拉伯文",
		"pt":    "葡萄牙文",
		"it":    "意大利文",
		"th":    "泰文",
		"vi":    "越南文",
	}

	if displayName, exists := languageMap[langCode]; exists {
		return displayName
	}

	return langCode
}

// BuildVisionDirectTranslationPrompt 构建视觉直出翻译提示词
func BuildVisionDirectTranslationPrompt(vars PromptVariables) string {
	targetLang := getLanguageDisplayName(vars.TargetLanguage)
	sourceLang := getLanguageDisplayName(vars.SourceLanguage)
	if vars.SourceLanguage == "auto" {
		sourceLang = "自动检测到的语言"
	}

	return fmt.Sprintf(`你是一个专业的视觉翻译专家，能够直接从图像中识别文字并翻译为%[1]s。
**核心任务：**
1. 识别图像中的所有文字内容；
2. 将识别的文字从%[2]s转换为%[1]s；
3. 直接输出翻译结果，保留原始格式。
**翻译要求：**
- 保持原文的换行、空格、标点符号等格式；
- 确保翻译准确、自然、符合%[1]s表达习惯；
- 考虑图像上下文，选择最合适的翻译；
- 不要包含任何解释、注释或原始文字。

**输出格式：**
直接输出翻译后的文字，不要添加任何其他内容。`, targetLang, sourceLang)
}

// 可覆盖的提示词，允许运行时根据配置动态调整
var (
	ExtractPrompt   = DefaultExtractPrompt
	TranslatePrompt = DefaultTranslatePrompt
)
