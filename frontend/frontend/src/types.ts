import {main} from '../wailsjs/go/models';

export type TranslationSource = 'manual' | 'screenshot';

export interface ScreenshotBounds {
	startX: number;
	startY: number;
	endX: number;
	endY: number;
	left: number;
	top: number;
	width: number;
	height: number;
}

export interface TranslationResult {
	originalText: string;
	translatedText: string;
	source: TranslationSource;
	timestamp: string;
	durationMs: number;
	bounds?: ScreenshotBounds;
}

export interface StatusMessage {
	stage: string;
	message: string;
}

export interface SettingsState {
	apiKeyOverride: string;
	apiBaseUrl: string;
	visionApiKeyOverride: string;
	visionApiBaseUrl: string;
	autoCopyResult: boolean;
	keepWindowOnTop: boolean;
	theme: string;
	showToastOnComplete: boolean;
	enableStreamOutput: boolean;
	hotkeyCombination: string;
	extractPrompt: string;
	translatePrompt: string;
	translateModel: string;
	visionModel: string;
	useVisionForTranslation: boolean;
	sourceLanguage: string;
	targetLanguage: string;
}

export const DEFAULT_API_BASE_URL = 'https://open.bigmodel.cn/api/paas/v4';
export const DEFAULT_TRANSLATE_MODEL = 'glm-4.5-flash';
export const DEFAULT_VISION_MODEL = 'glm-4v-flash';

export const DEFAULT_EXTRACT_PROMPT = `你是一个专业的视觉上下文分析专家，负责为高质量的翻译任务准备完整素材。请完成以下工作：

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
{{.VisionDirectInstruction}}`;

export const DEFAULT_TRANSLATE_PROMPT = `你是一个专业的翻译 AI，专门处理图像文本在特定语境下的翻译任务。你将收到一个 JSON 对象：
- "background" 字段提供场景参考；
- "words" 字段包含需要翻译的原始文本（语种：{{.SourceLanguage}}）。

请将 "words" 字段精准翻译为 {{.TargetLanguage}}，并保持原有的段落、换行与符号。遵循以下原则：
1. 仅翻译 "words" 字段，忽略 "background" 字段内容；
2. 依据 "background" 提供的语境选择合适的术语与表达；
3. 保持专有名词、数字与排版一致；
4. 输出中不得包含额外的说明或注释。

{{.VisionModeInstruction}}`;

// 语言映射表
export const LANGUAGE_MAP: Record<string, string> = {
	'auto': '自动检测',
	'zh-CN': '中文',
	'zh-TW': '繁体中文',
	'en': '英文',
	'ja': '日文',
	'ko': '韩文',
	'fr': '法文',
	'de': '德文',
	'es': '西班牙文',
	'ru': '俄文',
	'ar': '阿拉伯文',
	'pt': '葡萄牙文',
	'it': '意大利文',
	'th': '泰文',
	'vi': '越南文',
};

function getLanguageDisplayName(code: string): string {
	return LANGUAGE_MAP[code] ?? code;
}

// 提示词变量接口
export interface PromptVariables {
	sourceLanguage: string;
	targetLanguage: string;
	useVisionForTranslation: boolean;
}

// 处理提取提示词，根据运行模式追加指令
export function processExtractPrompt(basePrompt: string, vars: PromptVariables): string {
	let prompt = replaceLanguagePlaceholders(basePrompt, vars);

	const relay = buildRelayInstruction(vars);
	const direct = buildVisionDirectInstruction(vars);

	prompt = prompt.replace(/\{\{\.RelayInstruction\}\}/g, relay);
	prompt = prompt.replace(/\{\{\.VisionDirectInstruction\}\}/g, direct);

	return prompt.trim();
}

// 处理翻译提示词，替换动态变量
export function processTranslatePrompt(basePrompt: string, vars: PromptVariables): string {
	let prompt = replaceLanguagePlaceholders(basePrompt, vars);

	const visionInstruction = buildVisionModeInstruction(vars);
	prompt = prompt.replace(/\{\{\.VisionModeInstruction\}\}/g, visionInstruction);

	return prompt.trim();
}

function replaceLanguagePlaceholders(prompt: string, vars: PromptVariables): string {
	const sourceName = getLanguageDisplayName(vars.sourceLanguage);
	const targetName = getLanguageDisplayName(vars.targetLanguage);

	return prompt
		.replace(/\{\{\.SourceLanguage\}\}/g, sourceName)
		.replace(/\{\{\.TargetLanguage\}\}/g, targetName);
}

function buildRelayInstruction(vars: PromptVariables): string {
	if (vars.useVisionForTranslation) {
		return '';
	}
	const targetName = getLanguageDisplayName(vars.targetLanguage);
	return `当前未启用视觉直出模式，请确保只返回原始文字 JSON，后续翻译流程会将其转换为${targetName}。`;
}

function buildVisionDirectInstruction(vars: PromptVariables): string {
	if (!vars.useVisionForTranslation) {
		return '';
	}
	const targetName = getLanguageDisplayName(vars.targetLanguage);
	return `已启用视觉直出模式：完成 JSON 输出后，直接给出按原始版式排布的${targetName}翻译结果，不必再返回原文。`;
}

function buildVisionModeInstruction(vars: PromptVariables): string {
	const targetName = getLanguageDisplayName(vars.targetLanguage);
	if (vars.useVisionForTranslation) {
		return `视觉直出模式开启：若输入仍包含原文，请直接输出对应的${targetName}译文，并保持与原文一致的排版。`;
	}
	return `输入源自 OCR 流程，请只输出翻译后的${targetName}文本，不要重复或拼接原文。`;
}

// 构建视觉直出翻译提示词
export function buildVisionDirectTranslationPrompt(vars: PromptVariables): string {
	const targetLang = getLanguageDisplayName(vars.targetLanguage);
	let sourceLang = '自动检测到的语言';
	if (vars.sourceLanguage !== 'auto') {
		sourceLang = getLanguageDisplayName(vars.sourceLanguage);
	}

	return `你是一个专业的视觉翻译专家，能够直接从图像中识别文字并翻译为${targetLang}。
**核心任务：**
1. 识别图像中的所有文字内容；
2. 将识别的文字从${sourceLang}转换为${targetLang}；
3. 直接输出翻译结果，保留原始格式。
**翻译要求：**
- 保持原文的换行、空格、标点符号等格式；
- 确保翻译准确、自然、符合${targetLang}表达习惯；
- 考虑图像上下文，选择最合适的翻译；
- 不要包含任何解释、注释或原始文字。

**输出格式：**
直接输出翻译后的文字，不要添加任何其他内容。`;
}

export function defaultSettingsState(): SettingsState {
	return {
		apiKeyOverride: '',
		apiBaseUrl: DEFAULT_API_BASE_URL,
		visionApiKeyOverride: '',
		visionApiBaseUrl: DEFAULT_API_BASE_URL,
		autoCopyResult: true,
		keepWindowOnTop: false,
		theme: 'system',
		showToastOnComplete: true,
		enableStreamOutput: true,
		hotkeyCombination: 'Alt+T',
		extractPrompt: DEFAULT_EXTRACT_PROMPT,
		translatePrompt: DEFAULT_TRANSLATE_PROMPT,
		translateModel: DEFAULT_TRANSLATE_MODEL,
		visionModel: DEFAULT_VISION_MODEL,
		useVisionForTranslation: true,
		sourceLanguage: 'auto',
		targetLanguage: 'zh-CN',
	};
}

export function mapTranslationResult(data: main.UITranslationResult | any): TranslationResult {
	const converted = data instanceof main.UITranslationResult ? data : main.UITranslationResult.createFrom(data);
	const timestamp = converted.timestamp instanceof Date ? converted.timestamp : new Date(converted.timestamp ?? Date.now());
	const rawBounds: any = (converted as any).bounds;
	const bounds = rawBounds
		? {
			startX: Number(rawBounds.startX) || 0,
			startY: Number(rawBounds.startY) || 0,
			endX: Number(rawBounds.endX) || 0,
			endY: Number(rawBounds.endY) || 0,
			left: Number(rawBounds.left) || 0,
			top: Number(rawBounds.top) || 0,
			width: Number(rawBounds.width) || 1,
			height: Number(rawBounds.height) || 1,
		}
		: undefined;
	return {
		originalText: converted.originalText ?? '',
		translatedText: converted.translatedText ?? '',
		source: (converted.source as TranslationSource) ?? 'manual',
		timestamp: timestamp.toISOString(),
		durationMs: Number.isFinite(converted.durationMs) ? converted.durationMs : 0,
		bounds,
	};
}

export function mapSettings(data: main.SettingsDTO | any): SettingsState {
	const converted = data instanceof main.SettingsDTO ? data : main.SettingsDTO.createFrom(data);
	const defaults = defaultSettingsState();
	return {
		apiKeyOverride: converted.apiKeyOverride ?? '',
		apiBaseUrl: converted.apiBaseUrl || defaults.apiBaseUrl,
		visionApiKeyOverride: converted.visionApiKeyOverride ?? '',
		visionApiBaseUrl: converted.visionApiBaseUrl || converted.apiBaseUrl || defaults.visionApiBaseUrl,
		autoCopyResult: Boolean(converted.autoCopyResult),
		keepWindowOnTop: Boolean(converted.keepWindowOnTop),
		theme: converted.theme || defaults.theme,
		showToastOnComplete: Boolean(converted.showToastOnComplete),
		enableStreamOutput: Boolean((converted as any).enableStreamOutput ?? defaults.enableStreamOutput),
		hotkeyCombination: converted.hotkeyCombination || defaults.hotkeyCombination,
		extractPrompt: converted.extractPrompt || defaults.extractPrompt,
		translatePrompt: converted.translatePrompt || defaults.translatePrompt,
		translateModel: converted.translateModel || defaults.translateModel,
		visionModel: converted.visionModel || defaults.visionModel,
		useVisionForTranslation: Boolean((converted as any).useVisionForTranslation ?? defaults.useVisionForTranslation),
		sourceLanguage: (converted as any).sourceLanguage || defaults.sourceLanguage,
		targetLanguage: (converted as any).targetLanguage || defaults.targetLanguage,
	};
}

export function toSettingsPayload(state: SettingsState): main.SettingsDTO {
	return main.SettingsDTO.createFrom({
		apiKeyOverride: state.apiKeyOverride,
		apiBaseUrl: state.apiBaseUrl,
		visionApiKeyOverride: state.visionApiKeyOverride,
		visionApiBaseUrl: state.visionApiBaseUrl,
		autoCopyResult: state.autoCopyResult,
		keepWindowOnTop: state.keepWindowOnTop,
		theme: state.theme,
		showToastOnComplete: state.showToastOnComplete,
		enableStreamOutput: state.enableStreamOutput,
		hotkeyCombination: state.hotkeyCombination,
		extractPrompt: state.extractPrompt,
		translatePrompt: state.translatePrompt,
		translateModel: state.translateModel,
		visionModel: state.visionModel,
		useVisionForTranslation: state.useVisionForTranslation,
		sourceLanguage: state.sourceLanguage,
		targetLanguage: state.targetLanguage,
	});
}

export function formatTimestamp(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) {
		return iso;
	}
	return `${date.getMonth() + 1}/${date.getDate()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
}

export function formatDuration(durationMs: number): string {
	if (!Number.isFinite(durationMs) || durationMs <= 0) {
		return '';
	}
	if (durationMs < 1000) {
		return `${durationMs} ms`;
	}
	return `${(durationMs / 1000).toFixed(1)} s`;
}
